package generar_archivo_banco_galicia

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

type StartStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int
	ttlHours    int
	partPrefix  string
	filters     any
}

func NewStartStep() contracts.Service {
	return &StartStep{
		batchSize: 5000,
		ttlHours:  24,
	}
}

func (s *StartStep) Init(ctx *contracts.ServiceContext, servicePath string) {
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
		if v, ok := s.ctx.CurrentStepConfig["filters"]; ok {
			s.filters = v
		}
	}
}

func (s *StartStep) Execute() error {
	prov, err := DefaultProvider(s.ctx.Ctx)
	if err != nil {
		return err
	}

	input, err := buildStartInput(s.ctx, s.filters)
	if err != nil {
		return err
	}
	if input.Filters != nil {
		s.ctx.SetInputValue("filters", input.Filters)
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

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "export manager start completado",
		Data: map[string]any{
			"key_redis":     res.RedisKey,
			"id":            input.ParentID,
			"total_batches": res.TotalBatches,
		},
	})
	return nil
}

type ProcessBatchStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewProcessBatchStep() contracts.Service {
	return &ProcessBatchStep{}
}

func (s *ProcessBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *ProcessBatchStep) Execute() error {
	prov, err := DefaultProvider(s.ctx.Ctx)
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

type FinalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	fileBase    string
}

func NewFinalizeStep() contracts.Service {
	return &FinalizeStep{}
}

func (s *FinalizeStep) Init(ctx *contracts.ServiceContext, servicePath string) {
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

func (s *FinalizeStep) Execute() error {
	prov, err := DefaultProvider(s.ctx.Ctx)
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
		return exportmanager.Input{}, fmt.Errorf("id invalido")
	}
	if input.RedisKey == "" {
		return exportmanager.Input{}, fmt.Errorf("key_redis invalida")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		input.Filters = rawFilters
	}
	return input, nil
}

func buildStartInput(ctx *contracts.ServiceContext, configFilters any) (exportmanager.Input, error) {
	input := exportmanager.Input{
		RedisKey: fmt.Sprint(utils.GetInputValueOrDefault(ctx, "key_redis", "")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return exportmanager.Input{}, fmt.Errorf("id invalido")
	}
	var inputFilters any
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		inputFilters = rawFilters
	}
	mergedFilters, err := mergeFilters(configFilters, inputFilters)
	if err != nil {
		return exportmanager.Input{}, err
	}
	if mergedFilters != nil {
		input.Filters = mergedFilters
	}
	return input, nil
}

func mergeFilters(configFilters any, inputFilters any) (map[string]any, error) {
	merged := make(map[string]any)

	normalizedConfig, err := exportmanager.NormalizeFilters(configFilters)
	if err != nil {
		return nil, err
	}
	for key, value := range normalizedConfig {
		merged[key] = value
	}

	normalizedInput, err := exportmanager.NormalizeFilters(inputFilters)
	if err != nil {
		return nil, err
	}
	for key, value := range normalizedInput {
		merged[key] = value
	}

	if len(merged) == 0 {
		return nil, nil
	}
	return merged, nil
}

func markFailure(prov Provider, ctx context.Context, input exportmanager.Input, err error) {
	if errors.Is(err, domain.ErrBusinessRuleViolation) || errors.Is(err, domain.ErrInvalidArgument) {
		return
	}
	_ = prov.Manager().Fail(ctx, input, err)
}

func init() {
	serviceconfig.Register("bulk/export/generar_archivo_banco_galicia/start", NewStartStep)
	serviceconfig.Register("bulk/export/generar_archivo_banco_galicia/process_batch", NewProcessBatchStep)
	serviceconfig.Register("bulk/export/generar_archivo_banco_galicia/finalize", NewFinalizeStep)
}
