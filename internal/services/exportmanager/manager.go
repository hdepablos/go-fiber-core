package exportmanager

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Manager interface {
	Start(ctx context.Context, req StartRequest) (StartResult, error)
	ProcessBatch(ctx context.Context, req ProcessBatchRequest) (ProcessBatchResult, error)
	Finalize(ctx context.Context, req FinalizeRequest) (OutputResult, error)
	Fail(ctx context.Context, input Input, cause error) error
}

type manager struct {
	lifecycle       ParentLifecycle
	dataProvider    DataProvider
	headerBuilder   HeaderBuilder
	bodyBuilder     BodyBuilder
	footerBuilder   FooterBuilder
	outputRegistrar OutputRegistrar
	stateStore      StateStore
	store           ObjectStore
	defaultBucket   string
	defaultPrefix   string
}

func NewManager(
	lifecycle ParentLifecycle,
	dataProvider DataProvider,
	headerBuilder HeaderBuilder,
	bodyBuilder BodyBuilder,
	footerBuilder FooterBuilder,
	outputRegistrar OutputRegistrar,
	stateStore StateStore,
	store ObjectStore,
	defaultBucket string,
	defaultPrefix string,
) Manager {
	return &manager{
		lifecycle:       lifecycle,
		dataProvider:    dataProvider,
		headerBuilder:   headerBuilder,
		bodyBuilder:     bodyBuilder,
		footerBuilder:   footerBuilder,
		outputRegistrar: outputRegistrar,
		stateStore:      stateStore,
		store:           store,
		defaultBucket:   defaultBucket,
		defaultPrefix:   defaultPrefix,
	}
}

func (m *manager) Start(ctx context.Context, req StartRequest) (StartResult, error) {
	if err := validateStartInput(req.Input); err != nil {
		return StartResult{}, err
	}
	if strings.TrimSpace(req.Input.RedisKey) == "" {
		req.Input.RedisKey = fmt.Sprintf("run-%s", uuid.NewString())
	}
	if req.BatchSize <= 0 {
		req.BatchSize = 5000
	}
	if req.RedisTTL <= 0 {
		req.RedisTTL = 24 * time.Hour
	}
	if req.S3Bucket == "" {
		req.S3Bucket = m.defaultBucket
	}
	if req.S3Bucket == "" {
		return StartResult{}, fmt.Errorf("s3 bucket no configurado")
	}
	if req.PartPrefix == "" {
		req.PartPrefix = strings.Trim(req.Input.RedisKey, "/")
		if m.defaultPrefix != "" {
			req.PartPrefix = fmt.Sprintf("%s/%s", strings.Trim(m.defaultPrefix, "/"), strings.Trim(req.Input.RedisKey, "/"))
		}
	} else {
		req.PartPrefix = fmt.Sprintf("%s/%s", strings.Trim(req.PartPrefix, "/"), strings.Trim(req.Input.RedisKey, "/"))
	}

	execCtx := m.newExecutionContext(req.Input, Summary{}, req.RedisTTL)
	if m.lifecycle != nil {
		if err := m.lifecycle.Start(ctx, execCtx); err != nil {
			return StartResult{}, err
		}
	}

	load, err := m.dataProvider.LoadBatches(ctx, execCtx, req.BatchSize)
	if err != nil {
		return StartResult{}, err
	}

	batchesListKey, partsListKey, err := m.stateStore.Initialize(ctx, req.Input, load.Batches, load.Summary, load.Metadata, req.RedisTTL)
	if err != nil {
		return StartResult{}, err
	}

	totalBatches := len(load.Batches)
	if totalBatches == 0 {
		totalBatches = 1
	}

	if execCtx.Runtime != nil {
		_ = execCtx.Runtime.Set(ctx, "summary", load.Summary)
		_ = execCtx.Runtime.Set(ctx, "metadata", load.Metadata)
	}

	return StartResult{
		RedisKey:       req.Input.RedisKey,
		BatchesListKey: batchesListKey,
		PartsListKey:   partsListKey,
		TotalBatches:   totalBatches,
		Summary:        load.Summary,
		Metadata:       load.Metadata,
		S3Bucket:       req.S3Bucket,
		PartPrefix:     req.PartPrefix,
	}, nil
}

func (m *manager) ProcessBatch(ctx context.Context, req ProcessBatchRequest) (ProcessBatchResult, error) {
	if err := validateInput(req.Input); err != nil {
		return ProcessBatchResult{}, err
	}
	if req.S3Bucket == "" {
		req.S3Bucket = m.defaultBucket
	}
	if req.S3Bucket == "" {
		return ProcessBatchResult{}, fmt.Errorf("s3 bucket no configurado")
	}
	if req.BatchIndex < 0 || req.BatchIndex >= req.TotalBatches {
		return ProcessBatchResult{}, fmt.Errorf("batch_index fuera de rango: %d (total=%d)", req.BatchIndex, req.TotalBatches)
	}

	summary, err := m.stateStore.LoadSummary(ctx, req.Input)
	if err != nil {
		return ProcessBatchResult{}, err
	}
	execCtx := m.newExecutionContext(req.Input, summary, 24*time.Hour)
	batch, err := m.stateStore.LoadBatch(ctx, req.BatchesListKey, req.BatchIndex)
	if err != nil {
		return ProcessBatchResult{}, err
	}

	lines := make([]string, 0, len(batch.Items)+4)
	if req.BatchIndex == 0 && m.headerBuilder != nil {
		header, err := m.headerBuilder.BuildHeader(ctx, execCtx)
		if err != nil {
			return ProcessBatchResult{}, err
		}
		lines = append(lines, header...)
	}

	for _, item := range batch.Items {
		bodyLines, err := m.bodyBuilder.BuildBodyLines(ctx, execCtx, item)
		if err != nil {
			return ProcessBatchResult{}, err
		}
		lines = append(lines, bodyLines...)
	}

	isLast := req.BatchIndex == req.TotalBatches-1
	if isLast && m.footerBuilder != nil {
		footer, err := m.footerBuilder.BuildFooter(ctx, execCtx)
		if err != nil {
			return ProcessBatchResult{}, err
		}
		if len(footer) > 0 {
			lines = append(lines, footer...)
		}
	}

	if err := m.store.EnsureBucket(ctx, req.S3Bucket); err != nil {
		return ProcessBatchResult{}, err
	}

	partKey := fmt.Sprintf("%s/part-%06d.csv", strings.Trim(req.PartPrefix, "/"), req.BatchIndex)
	if err := m.store.PutObject(ctx, req.S3Bucket, partKey, []byte(strings.Join(lines, "\n")+"\n"), "text/csv"); err != nil {
		return ProcessBatchResult{}, err
	}
	if err := m.stateStore.AppendPartKey(ctx, req.PartsListKey, partKey); err != nil {
		return ProcessBatchResult{}, err
	}

	return ProcessBatchResult{
		NextBatchIndex: req.BatchIndex + 1,
		IsLastBatch:    isLast,
		ProcessedCount: len(batch.Items),
		S3PartKey:      partKey,
	}, nil
}

func (m *manager) Finalize(ctx context.Context, req FinalizeRequest) (OutputResult, error) {
	if err := validateInput(req.Input); err != nil {
		return OutputResult{}, err
	}
	if req.S3Bucket == "" {
		req.S3Bucket = m.defaultBucket
	}
	if req.S3Bucket == "" {
		return OutputResult{}, fmt.Errorf("s3 bucket no configurado")
	}

	summary, err := m.stateStore.LoadSummary(ctx, req.Input)
	if err != nil {
		return OutputResult{}, err
	}
	execCtx := m.newExecutionContext(req.Input, summary, 24*time.Hour)
	partKeys, err := m.stateStore.LoadPartKeys(ctx, req.PartsListKey)
	if err != nil {
		return OutputResult{}, err
	}
	if len(partKeys) == 0 {
		return OutputResult{}, fmt.Errorf("no hay partes en redis para merge: %s", req.PartsListKey)
	}

	finalKey := buildFinalKey(req.FileBase, req.Input.ParentID, req.Input.RedisKey, m.defaultPrefix)
	uploadID, err := m.store.CreateMultipartUpload(ctx, req.S3Bucket, finalKey, "text/csv")
	if err != nil {
		return OutputResult{}, err
	}

	const partSize = 8 * 1024 * 1024
	buf := bytes.NewBuffer(make([]byte, 0, partSize))
	tmp := make([]byte, 64*1024)
	completed := make([]CompletedPart, 0)
	partNumber := int32(1)

	flush := func() error {
		if buf.Len() == 0 {
			return nil
		}
		out, err := m.store.UploadPart(ctx, req.S3Bucket, finalKey, uploadID, partNumber, append([]byte(nil), buf.Bytes()...))
		if err != nil {
			return err
		}
		completed = append(completed, CompletedPart{ETag: out, PartNumber: partNumber})
		partNumber++
		buf.Reset()
		return nil
	}

	abort := func() {
		_ = m.store.AbortMultipartUpload(context.Background(), req.S3Bucket, finalKey, uploadID)
	}

	for _, key := range partKeys {
		rc, err := m.store.GetObject(ctx, req.S3Bucket, key)
		if err != nil {
			abort()
			return OutputResult{}, err
		}
		for {
			n, rerr := rc.Read(tmp)
			if n > 0 {
				_, _ = buf.Write(tmp[:n])
				if buf.Len() >= partSize {
					if err := flush(); err != nil {
						_ = rc.Close()
						abort()
						return OutputResult{}, err
					}
				}
			}
			if rerr != nil {
				if rerr == io.EOF {
					break
				}
				_ = rc.Close()
				abort()
				return OutputResult{}, rerr
			}
		}
		_ = rc.Close()
	}

	if err := flush(); err != nil {
		abort()
		return OutputResult{}, err
	}
	if err := m.store.CompleteMultipartUpload(ctx, req.S3Bucket, finalKey, uploadID, completed); err != nil {
		abort()
		return OutputResult{}, err
	}

	info, err := m.store.HeadObject(ctx, req.S3Bucket, finalKey)
	if err != nil {
		return OutputResult{}, err
	}

	output := OutputResult{
		Bucket:      req.S3Bucket,
		Key:         finalKey,
		Path:        fmt.Sprintf("s3://%s/%s", req.S3Bucket, finalKey),
		FileSize:    info.ContentLength,
		ContentType: "text/csv",
		Parts:       len(partKeys),
	}

	for _, key := range partKeys {
		if err := m.store.DeleteObject(ctx, req.S3Bucket, key); err != nil {
			return OutputResult{}, err
		}
	}
	if err := m.stateStore.Cleanup(ctx, req.Input, fmt.Sprintf("%s:batches", req.Input.RedisKey), req.PartsListKey); err != nil {
		return OutputResult{}, err
	}

	if m.outputRegistrar != nil {
		if err := m.outputRegistrar.Register(ctx, execCtx, output); err != nil {
			return OutputResult{}, err
		}
	}
	if m.lifecycle != nil {
		if err := m.lifecycle.End(ctx, execCtx, output); err != nil {
			return OutputResult{}, err
		}
	}

	return output, nil
}

func (m *manager) Fail(ctx context.Context, input Input, cause error) error {
	if m.lifecycle == nil {
		return nil
	}
	return m.lifecycle.Fail(ctx, m.newExecutionContext(input, Summary{}, 24*time.Hour), cause)
}

func validateInput(input Input) error {
	if err := validateStartInput(input); err != nil {
		return err
	}
	if strings.TrimSpace(input.RedisKey) == "" {
		return fmt.Errorf("key_redis inválida")
	}
	return nil
}

func validateStartInput(input Input) error {
	if input.ParentID <= 0 {
		return fmt.Errorf("id inválido")
	}
	return nil
}

func buildFinalKey(fileBase string, parentID int64, redisKey string, defaultPrefix string) string {
	base := strings.TrimSpace(fileBase)
	if base != "" {
		base = strings.TrimSuffix(base, ".csv")
		return fmt.Sprintf("%s-%d.csv", base, parentID)
	}
	if defaultPrefix != "" {
		return fmt.Sprintf("%s/%s-%d.csv", strings.Trim(defaultPrefix, "/"), strings.Trim(redisKey, "/"), parentID)
	}
	return fmt.Sprintf("%s-%d.csv", strings.Trim(redisKey, "/"), parentID)
}

func (m *manager) newExecutionContext(input Input, summary Summary, ttl time.Duration) ExecutionContext {
	execCtx := ExecutionContext{
		Input:   input,
		Summary: summary,
	}
	if provider, ok := m.stateStore.(RuntimeValuesProvider); ok {
		execCtx.Runtime = provider.RuntimeValues(input, ttl)
	}
	return execCtx
}
