package bulkexportv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/models"
)

type OrganizeRequest struct {
	BulkJobID     int64
	BatchSize     int
	RedisTTL      time.Duration
	S3Prefix      string
	S3Bucket      string
	ProjectPrefix string
	RunID         string
}

type OrganizeResult struct {
	RunID          string
	TotalBatches   int
	BatchesListKey string
	PartsListKey   string
	S3Prefix       string
	S3Bucket       string
}

type WriteBatchRequest struct {
	BulkJobID      int64
	RunID          string
	BatchIndex     int
	TotalBatches   int
	BatchesListKey string
	PartsListKey   string
	S3Prefix       string
	S3Bucket       string
}

type WriteBatchResult struct {
	NextBatchIndex int
	IsLastBatch    bool
	ProcessedCount int
	S3PartKey      string
}

type MergeRequest struct {
	BulkJobID      int64
	RunID          string
	PartsListKey   string
	S3Prefix       string
	S3Bucket       string
	FileBase       string
	TotalProcessed int
}

type MergeResult struct {
	S3FinalKey string
	S3FilePath string
	FileSize   int64
	Parts      int
	RunID      string
}

type ExportPipeline interface {
	Organize(ctx context.Context, req OrganizeRequest) (OrganizeResult, error)
	WriteCSVBatch(ctx context.Context, req WriteBatchRequest) (WriteBatchResult, error)
	MergeMultipart(ctx context.Context, req MergeRequest) (MergeResult, error)
}

type exportPipeline struct {
	bulkJobs      BulkJobReader
	bulkJobWriter BulkJobWriter
	items         BulkJobItemReader
	outputs       BulkJobOutputWriter
	cache         Cache
	csvBuilder    CSVBuilder
	store         ObjectStore
	runIDProvider RunIDProvider
	defaultBucket string
}

type RunIDProvider interface {
	NewRunID() string
}

func NewExportPipeline(
	bulkJobs BulkJobReader,
	bulkJobWriter BulkJobWriter,
	items BulkJobItemReader,
	outputs BulkJobOutputWriter,
	cache Cache,
	csvBuilder CSVBuilder,
	store ObjectStore,
	runIDProvider RunIDProvider,
	defaultBucket string,
) ExportPipeline {
	return &exportPipeline{
		bulkJobs:      bulkJobs,
		bulkJobWriter: bulkJobWriter,
		items:         items,
		outputs:       outputs,
		cache:         cache,
		csvBuilder:    csvBuilder,
		store:         store,
		runIDProvider: runIDProvider,
		defaultBucket: defaultBucket,
	}
}

func (s *exportPipeline) Organize(ctx context.Context, req OrganizeRequest) (OrganizeResult, error) {
	if req.BulkJobID <= 0 {
		return OrganizeResult{}, fmt.Errorf("bulk_job_id inválido")
	}
	if req.BatchSize <= 0 {
		req.BatchSize = 5000
	}
	if req.RedisTTL <= 0 {
		req.RedisTTL = 24 * time.Hour
	}
	if req.ProjectPrefix == "" {
		req.ProjectPrefix = "go-fiber-core"
	}
	if req.RunID == "" {
		if s.runIDProvider == nil {
			return OrganizeResult{}, fmt.Errorf("run_id_provider missing")
		}
		req.RunID = s.runIDProvider.NewRunID()
	}

	status, err := s.bulkJobs.GetStatus(ctx, req.BulkJobID)
	if err != nil {
		return OrganizeResult{}, err
	}
	if status != models.BulkJobStatusImported {
		return OrganizeResult{}, fmt.Errorf("%w: Verifique el proceso con el id %d ya fue procesado actualmente con el status %s de la tabla bulk_jobs", domain.ErrBusinessRuleViolation, req.BulkJobID, status)
	}
	if err := s.bulkJobWriter.UpdateStatus(ctx, req.BulkJobID, models.BulkJobStatusProcessing); err != nil {
		return OrganizeResult{}, err
	}

	if req.S3Prefix == "" {
		req.S3Prefix = fmt.Sprintf("exports/bulk_jobs/%d/v1", req.BulkJobID)
	}
	if req.S3Bucket == "" {
		req.S3Bucket = s.defaultBucket
	}
	if req.S3Bucket == "" {
		return OrganizeResult{}, fmt.Errorf("s3 bucket no configurado")
	}

	batchesListKey := fmt.Sprintf("%s:bulk_export:v1:%s:batches", req.ProjectPrefix, req.RunID)
	partsListKey := fmt.Sprintf("%s:bulk_export:v1:%s:s3_parts", req.ProjectPrefix, req.RunID)

	_ = s.cache.Del(ctx, batchesListKey, partsListKey)

	lastID := int64(0)
	batchIndex := 0
	for {
		ids, err := s.items.ListIDsAfter(ctx, req.BulkJobID, lastID, req.BatchSize)
		if err != nil {
			return OrganizeResult{}, err
		}
		if len(ids) == 0 {
			break
		}

		batchKey := fmt.Sprintf("%s:bulk_export:v1:%s:batch:%06d", req.ProjectPrefix, req.RunID, batchIndex)
		payload, err := json.Marshal(ids)
		if err != nil {
			return OrganizeResult{}, err
		}
		if err := s.cache.SetBytes(ctx, batchKey, payload, req.RedisTTL); err != nil {
			return OrganizeResult{}, err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			return OrganizeResult{}, err
		}

		lastID = ids[len(ids)-1]
		batchIndex++
	}

	if batchIndex == 0 {
		batchKey := fmt.Sprintf("%s:bulk_export:v1:%s:batch:%06d", req.ProjectPrefix, req.RunID, batchIndex)
		if err := s.cache.SetBytes(ctx, batchKey, []byte("[]"), req.RedisTTL); err != nil {
			return OrganizeResult{}, err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			return OrganizeResult{}, err
		}
		batchIndex = 1
	}

	_ = s.cache.Expire(ctx, batchesListKey, req.RedisTTL)
	_ = s.cache.Expire(ctx, partsListKey, req.RedisTTL)

	return OrganizeResult{
		RunID:          req.RunID,
		TotalBatches:   batchIndex,
		BatchesListKey: batchesListKey,
		PartsListKey:   partsListKey,
		S3Prefix:       req.S3Prefix,
		S3Bucket:       req.S3Bucket,
	}, nil
}

func (s *exportPipeline) WriteCSVBatch(ctx context.Context, req WriteBatchRequest) (WriteBatchResult, error) {
	if req.BulkJobID <= 0 {
		return WriteBatchResult{}, fmt.Errorf("bulk_job_id inválido")
	}
	if req.RunID == "" {
		return WriteBatchResult{}, fmt.Errorf("run_id missing")
	}
	if req.TotalBatches <= 0 {
		return WriteBatchResult{}, fmt.Errorf("total_batches missing/invalid")
	}
	if req.BatchIndex < 0 || req.BatchIndex >= req.TotalBatches {
		return WriteBatchResult{}, fmt.Errorf("batch_index fuera de rango: %d (total=%d)", req.BatchIndex, req.TotalBatches)
	}
	if req.BatchesListKey == "" {
		return WriteBatchResult{}, fmt.Errorf("batches_list_key missing")
	}
	if req.PartsListKey == "" {
		return WriteBatchResult{}, fmt.Errorf("parts_list_key missing")
	}
	if req.S3Prefix == "" {
		req.S3Prefix = fmt.Sprintf("exports/bulk_jobs/%d/v1", req.BulkJobID)
	}
	if req.S3Bucket == "" {
		req.S3Bucket = s.defaultBucket
	}
	if req.S3Bucket == "" {
		return WriteBatchResult{}, fmt.Errorf("s3 bucket no configurado")
	}

	batchKey, err := s.cache.LIndex(ctx, req.BatchesListKey, int64(req.BatchIndex))
	if err != nil {
		return WriteBatchResult{}, err
	}
	idsJSON, err := s.cache.GetBytes(ctx, batchKey)
	if err != nil {
		return WriteBatchResult{}, err
	}

	var ids []int64
	_ = json.Unmarshal(idsJSON, &ids)

	items, err := s.items.FindByIDs(ctx, req.BulkJobID, ids)
	if err != nil {
		return WriteBatchResult{}, err
	}

	includeHeader := req.BatchIndex == 0
	csvBytes, err := s.csvBuilder.Build(items, includeHeader)
	if err != nil {
		return WriteBatchResult{}, err
	}

	if err := s.store.EnsureBucket(ctx, req.S3Bucket); err != nil {
		return WriteBatchResult{}, err
	}

	partKey := fmt.Sprintf("%s/run-%s/part-%06d.csv", req.S3Prefix, req.RunID, req.BatchIndex)
	if err := s.store.PutObject(ctx, req.S3Bucket, partKey, csvBytes, "text/csv"); err != nil {
		return WriteBatchResult{}, err
	}

	if err := s.cache.RPush(ctx, req.PartsListKey, partKey); err != nil {
		return WriteBatchResult{}, err
	}

	isLast := req.BatchIndex == req.TotalBatches-1
	return WriteBatchResult{
		NextBatchIndex: req.BatchIndex + 1,
		IsLastBatch:    isLast,
		ProcessedCount: len(items),
		S3PartKey:      partKey,
	}, nil
}

func (s *exportPipeline) MergeMultipart(ctx context.Context, req MergeRequest) (MergeResult, error) {
	if req.BulkJobID <= 0 {
		return MergeResult{}, fmt.Errorf("bulk_job_id inválido")
	}
	if req.RunID == "" {
		return MergeResult{}, fmt.Errorf("run_id missing")
	}
	if req.PartsListKey == "" {
		return MergeResult{}, fmt.Errorf("parts_list_key missing")
	}
	if req.S3Prefix == "" {
		req.S3Prefix = fmt.Sprintf("exports/bulk_jobs/%d/v1", req.BulkJobID)
	}
	if req.S3Bucket == "" {
		req.S3Bucket = s.defaultBucket
	}
	if req.S3Bucket == "" {
		return MergeResult{}, fmt.Errorf("s3 bucket no configurado")
	}

	partKeys, err := s.cache.LRange(ctx, req.PartsListKey, 0, -1)
	if err != nil {
		return MergeResult{}, err
	}
	if len(partKeys) == 0 {
		return MergeResult{}, fmt.Errorf("no hay partes en redis para merge: %s", req.PartsListKey)
	}

	if err := s.store.EnsureBucket(ctx, req.S3Bucket); err != nil {
		return MergeResult{}, err
	}

	finalKey := buildFinalObjectKey(req.FileBase, req.BulkJobID, req.S3Prefix, req.RunID)
	uploadID, err := s.store.CreateMultipartUpload(ctx, req.S3Bucket, finalKey, "text/csv")
	if err != nil {
		return MergeResult{}, err
	}

	const partSize = 8 * 1024 * 1024
	buf := bytesBuffer(partSize)
	tmp := make([]byte, 64*1024)

	var completed []CompletedPart
	partNumber := int32(1)

	flush := func() error {
		if buf.Len() == 0 {
			return nil
		}
		b := buf.BytesCopy()
		etag, err := s.store.UploadPart(ctx, req.S3Bucket, finalKey, uploadID, partNumber, b)
		if err != nil {
			return err
		}
		completed = append(completed, CompletedPart{
			ETag:       etag,
			PartNumber: partNumber,
		})
		partNumber++
		buf.Reset()
		return nil
	}

	abort := func() {
		_ = s.store.AbortMultipartUpload(context.Background(), req.S3Bucket, finalKey, uploadID)
	}

	for _, k := range partKeys {
		rc, err := s.store.GetObject(ctx, req.S3Bucket, k)
		if err != nil {
			abort()
			return MergeResult{}, err
		}
		for {
			n, rerr := rc.Read(tmp)
			if n > 0 {
				_, _ = buf.Write(tmp[:n])
				if buf.Len() >= partSize {
					if err := flush(); err != nil {
						_ = rc.Close()
						abort()
						return MergeResult{}, err
					}
				}
			}
			if rerr != nil {
				if rerr == io.EOF {
					break
				}
				_ = rc.Close()
				abort()
				return MergeResult{}, rerr
			}
		}
		_ = rc.Close()
	}

	if err := flush(); err != nil {
		abort()
		return MergeResult{}, err
	}

	if err := s.store.CompleteMultipartUpload(ctx, req.S3Bucket, finalKey, uploadID, completed); err != nil {
		abort()
		return MergeResult{}, err
	}

	info, err := s.store.HeadObject(ctx, req.S3Bucket, finalKey)
	if err != nil {
		return MergeResult{}, err
	}

	if err := s.store.DeletePrefix(ctx, req.S3Bucket, fmt.Sprintf("exports/bulk_jobs/%d/", req.BulkJobID)); err != nil {
		return MergeResult{}, err
	}

	metadata, err := json.Marshal(map[string]any{
		"bucket":          req.S3Bucket,
		"key":             finalKey,
		"run_id":          req.RunID,
		"parts":           len(partKeys),
		"processed_count": req.TotalProcessed,
	})
	if err != nil {
		return MergeResult{}, err
	}

	fileSize := info.ContentLength
	output := &models.BulkJobOutput{
		BulkJobID: req.BulkJobID,
		Type:      "csv",
		FilePath:  fmt.Sprintf("s3://%s/%s", req.S3Bucket, finalKey),
		FileSize:  &fileSize,
		Status:    models.BulkJobOutputStatusGenerated,
		Metadata:  metadata,
	}
	if err := s.outputs.Create(ctx, output); err != nil {
		return MergeResult{}, err
	}
	if err := s.bulkJobWriter.UpdateStatus(ctx, req.BulkJobID, models.BulkJobStatusProcessed); err != nil {
		return MergeResult{}, err
	}

	return MergeResult{
		S3FinalKey: finalKey,
		S3FilePath: fmt.Sprintf("s3://%s/%s", req.S3Bucket, finalKey),
		FileSize:   fileSize,
		Parts:      len(partKeys),
		RunID:      req.RunID,
	}, nil
}

type byteBuffer struct {
	buf []byte
	n   int
}

func bytesBuffer(capacity int) *byteBuffer {
	return &byteBuffer{buf: make([]byte, 0, capacity)}
}

func (b *byteBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	b.n = len(b.buf)
	return len(p), nil
}

func (b *byteBuffer) Len() int { return b.n }

func (b *byteBuffer) Reset() {
	b.buf = b.buf[:0]
	b.n = 0
}

func (b *byteBuffer) BytesCopy() []byte {
	out := make([]byte, len(b.buf))
	copy(out, b.buf)
	return out
}

func buildFinalObjectKey(fileBase string, bulkJobID int64, s3Prefix string, runID string) string {
	base := strings.TrimSpace(fileBase)
	if base == "" {
		return fmt.Sprintf("%s/run-%s/final.csv", s3Prefix, runID)
	}
	base = strings.TrimLeft(base, "/")
	if !strings.HasPrefix(base, "exports/") {
		base = "exports/" + base
	}
	base = strings.TrimSuffix(base, ".csv")
	return fmt.Sprintf("%s-%d.csv", base, bulkJobID)
}
