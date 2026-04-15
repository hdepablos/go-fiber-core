package bulkexportv2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type startStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int
	ttlHours    int
	partPrefix  string
}

func NewStartStep() contracts.Service {
	return &startStep{
		batchSize: 5000,
		ttlHours:  24,
	}
}

func (s *startStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["batch_size"]; ok {
			s.batchSize = utils.ToInt(v)
		}
		if v, ok := s.ctx.CurrentStepConfig["redis_ttl_hours"]; ok {
			s.ttlHours = utils.ToInt(v)
		}
		if v, ok := s.ctx.CurrentStepConfig["part_prefix"]; ok {
			if str, ok := v.(string); ok {
				s.partPrefix = str
			}
		}
	}
}

func (s *startStep) Execute() error {
	prov, err := ProviderFromContext(s.ctx.Ctx)
	if err != nil {
		return err
	}

	input, err := buildStartInput(s.ctx)
	if err != nil {
		return err
	}

	res, err := prov.Manager().Start(s.ctx.Ctx, exportmanager.StartRequest{
		Input:      input,
		BatchSize:  s.batchSize,
		RedisTTL:   time.Duration(s.ttlHours) * time.Hour,
		S3Bucket:   fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", "")),
		PartPrefix: s.partPrefix,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetInputValue("id", input.ParentID)
	s.ctx.SetInputValue("key_redis", res.RedisKey)
	s.ctx.SetInputValue("batches_list_key", res.BatchesListKey)
	s.ctx.SetInputValue("parts_list_key", res.PartsListKey)
	s.ctx.SetInputValue("total_batches", res.TotalBatches)
	s.ctx.SetInputValue("batch_index", 0)
	s.ctx.SetInputValue("is_last_batch", false)
	s.ctx.SetInputValue("s3_bucket", res.S3Bucket)
	s.ctx.SetInputValue("part_prefix", res.PartPrefix)
	s.ctx.SetInputValue("total_records", res.Summary.TotalRecords)
	s.ctx.SetInputValue("total_amount", res.Summary.TotalAmount)

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "export manager start completado",
		Data: map[string]any{
			"key_redis":     res.RedisKey,
			"id":            input.ParentID,
			"total_batches": res.TotalBatches,
			"total_records": res.Summary.TotalRecords,
			"total_amount":  res.Summary.TotalAmount,
		},
	})
	return nil
}

type processBatchStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewProcessBatchStep() contracts.Service {
	return &processBatchStep{}
}

func (s *processBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *processBatchStep) Execute() error {
	prov, err := ProviderFromContext(s.ctx.Ctx)
	if err != nil {
		return err
	}
	input, err := buildInput(s.ctx)
	if err != nil {
		return err
	}

	res, err := prov.Manager().ProcessBatch(s.ctx.Ctx, exportmanager.ProcessBatchRequest{
		Input:          input,
		BatchesListKey: fmt.Sprint(utils.MustGetInputValue(s.ctx, "batches_list_key")),
		PartsListKey:   fmt.Sprint(utils.MustGetInputValue(s.ctx, "parts_list_key")),
		S3Bucket:       fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", "")),
		PartPrefix:     fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "part_prefix", "")),
		BatchIndex:     utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "batch_index", 0)),
		TotalBatches:   utils.ToInt(utils.MustGetInputValue(s.ctx, "total_batches")),
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batch procesado",
		Data: map[string]any{
			"batch_index":     res.NextBatchIndex,
			"is_last_batch":   res.IsLastBatch,
			"processed_count": res.ProcessedCount,
			"s3_part_key":     res.S3PartKey,
		},
	})
	return nil
}

type finalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	fileBase    string
}

func NewFinalizeStep() contracts.Service {
	return &finalizeStep{}
}

func (s *finalizeStep) Init(ctx *contracts.ServiceContext, servicePath string) {
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

func (s *finalizeStep) Execute() error {
	prov, err := ProviderFromContext(s.ctx.Ctx)
	if err != nil {
		return err
	}
	input, err := buildInput(s.ctx)
	if err != nil {
		return err
	}

	output, err := prov.Manager().Finalize(s.ctx.Ctx, exportmanager.FinalizeRequest{
		Input:        input,
		PartsListKey: fmt.Sprint(utils.MustGetInputValue(s.ctx, "parts_list_key")),
		S3Bucket:     fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", "")),
		FileBase:     s.fileBase,
		TotalParts:   utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "total_batches", 0)),
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "archivo final generado",
		Data: map[string]any{
			"s3_final_key": output.Key,
			"s3_file_path": output.Path,
			"file_size":    output.FileSize,
			"parts":        output.Parts,
		},
	})
	return nil
}

func buildInput(ctx *contracts.ServiceContext) (exportmanager.Input, error) {
	input := exportmanager.Input{
		RedisKey: fmt.Sprint(utils.MustGetInputValue(ctx, "key_redis")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return exportmanager.Input{}, fmt.Errorf("id inválido")
	}
	if input.RedisKey == "" {
		return exportmanager.Input{}, fmt.Errorf("key_redis inválida")
	}

	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		input.Filters = rawFilters
	}
	return input, nil
}

func buildStartInput(ctx *contracts.ServiceContext) (exportmanager.Input, error) {
	input := exportmanager.Input{
		RedisKey: fmt.Sprint(utils.GetInputValueOrDefault(ctx, "key_redis", "")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return exportmanager.Input{}, fmt.Errorf("id inválido")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		input.Filters = rawFilters
	}
	return input, nil
}

func markFailure(prov Provider, ctx context.Context, input exportmanager.Input, err error) {
	if errors.Is(err, domain.ErrBusinessRuleViolation) || errors.Is(err, domain.ErrInvalidArgument) {
		return
	}
	_ = prov.Manager().Fail(ctx, input, err)
}

func init() {
	serviceconfig.Register("bulk/export/v2/start", NewStartStep)
	serviceconfig.Register("bulk/export/v2/process_batch", NewProcessBatchStep)
	serviceconfig.Register("bulk/export/v2/finalize", NewFinalizeStep)
}
