package bulkexportv1

import (
	"fmt"
	"time"

	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type OrganizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewOrganizeStep() contracts.Service {
	return &OrganizeStep{}
}

func (s *OrganizeStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *OrganizeStep) Execute() error {
	if s.ctx == nil {
		return fmt.Errorf("service context is nil")
	}
	prov, err := DefaultProvider(s.ctx.Ctx)
	if err != nil {
		return err
	}
	pipeline := prov.Pipeline()

	bulkJobID := utils.ToInt64(utils.MustGetInputValue(s.ctx, "bulk_job_id"))
	if bulkJobID <= 0 {
		return fmt.Errorf("bulk_job_id inválido")
	}

	batchSize := utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "batch_size", 5000))
	ttlHours := utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "redis_ttl_hours", 24))
	if ttlHours <= 0 {
		ttlHours = 24
	}

	s3Prefix := fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_prefix", ""))
	s3Bucket := fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", ""))

	res, err := pipeline.Organize(s.ctx.Ctx, OrganizeRequest{
		BulkJobID:     bulkJobID,
		BatchSize:     batchSize,
		RedisTTL:      time.Duration(ttlHours) * time.Hour,
		S3Prefix:      s3Prefix,
		S3Bucket:      s3Bucket,
		ProjectPrefix: projectPrefix(),
		RunID:         fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "run_id", "")),
	})
	if err != nil {
		return err
	}

	s.ctx.SetInputValue("run_id", res.RunID)
	s.ctx.SetInputValue("total_batches", res.TotalBatches)
	s.ctx.SetInputValue("batches_list_key", res.BatchesListKey)
	s.ctx.SetInputValue("parts_list_key", res.PartsListKey)
	s.ctx.SetInputValue("batch_index", 0)
	s.ctx.SetInputValue("is_last_batch", false)
	s.ctx.SetInputValue("s3_prefix", res.S3Prefix)
	s.ctx.SetInputValue("s3_bucket", res.S3Bucket)

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "data organizada",
		Data: map[string]any{
			"run_id":        res.RunID,
			"total_batches": res.TotalBatches,
		},
	})
	return nil
}

type WriteCSVBatchStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewWriteCSVBatchStep() contracts.Service {
	return &WriteCSVBatchStep{}
}

func (s *WriteCSVBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *WriteCSVBatchStep) Execute() error {
	if s.ctx == nil {
		return fmt.Errorf("service context is nil")
	}
	prov, err := DefaultProvider(s.ctx.Ctx)
	if err != nil {
		return err
	}
	pipeline := prov.Pipeline()

	bulkJobID := utils.ToInt64(utils.MustGetInputValue(s.ctx, "bulk_job_id"))
	if bulkJobID <= 0 {
		return fmt.Errorf("bulk_job_id inválido")
	}

	runID := fmt.Sprint(utils.MustGetInputValue(s.ctx, "run_id"))
	totalBatches := utils.ToInt(utils.MustGetInputValue(s.ctx, "total_batches"))
	batchIndex := utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "batch_index", 0))
	batchesListKey := fmt.Sprint(utils.MustGetInputValue(s.ctx, "batches_list_key"))
	partsListKey := fmt.Sprint(utils.MustGetInputValue(s.ctx, "parts_list_key"))
	s3Prefix := fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_prefix", ""))
	s3Bucket := fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", ""))

	res, err := pipeline.WriteCSVBatch(s.ctx.Ctx, WriteBatchRequest{
		BulkJobID:      bulkJobID,
		RunID:          runID,
		BatchIndex:     batchIndex,
		TotalBatches:   totalBatches,
		BatchesListKey: batchesListKey,
		PartsListKey:   partsListKey,
		S3Prefix:       s3Prefix,
		S3Bucket:       s3Bucket,
	})
	if err != nil {
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "csv generado",
		Data: map[string]any{
			"batch_index":     res.NextBatchIndex,
			"is_last_batch":   res.IsLastBatch,
			"processed_count": res.ProcessedCount,
			"s3_part_key":     res.S3PartKey,
		},
	})
	return nil
}

type MergeMultipartStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	fileBase    string
}

func NewMergeMultipartStep() contracts.Service {
	return &MergeMultipartStep{}
}

func (s *MergeMultipartStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["file"]; ok {
			if str, ok := v.(string); ok {
				s.fileBase = str
			}
		}
	}
}

func (s *MergeMultipartStep) Execute() error {
	if s.ctx == nil {
		return fmt.Errorf("service context is nil")
	}
	prov, err := DefaultProvider(s.ctx.Ctx)
	if err != nil {
		return err
	}
	pipeline := prov.Pipeline()

	bulkJobID := utils.ToInt64(utils.MustGetInputValue(s.ctx, "bulk_job_id"))
	if bulkJobID <= 0 {
		return fmt.Errorf("bulk_job_id inválido")
	}

	runID := fmt.Sprint(utils.MustGetInputValue(s.ctx, "run_id"))
	partsListKey := fmt.Sprint(utils.MustGetInputValue(s.ctx, "parts_list_key"))
	s3Prefix := fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_prefix", ""))
	s3Bucket := fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", ""))

	res, err := pipeline.MergeMultipart(s.ctx.Ctx, MergeRequest{
		BulkJobID:      bulkJobID,
		RunID:          runID,
		PartsListKey:   partsListKey,
		S3Prefix:       s3Prefix,
		S3Bucket:       s3Bucket,
		FileBase:       s.fileBase,
		TotalProcessed: utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "total_processed", 0)),
	})
	if err != nil {
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "merge completado",
		Data: map[string]any{
			"s3_final_key": res.S3FinalKey,
			"s3_file_path": res.S3FilePath,
			"file_size":    res.FileSize,
			"run_id":       res.RunID,
			"parts":        res.Parts,
		},
	})
	return nil
}

func init() {
	serviceconfig.Register("bulk/export/v1/organize", NewOrganizeStep)
	serviceconfig.Register("bulk/export/v1/write_csv_batch", NewWriteCSVBatchStep)
	serviceconfig.Register("bulk/export/v1/merge_multipart", NewMergeMultipartStep)
}
