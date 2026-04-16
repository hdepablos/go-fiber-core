package bulkprocess

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type startStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int
	ttlHours    int
}

func NewStartStep() contracts.Service {
	return &startStep{
		batchSize: 500,
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
	if input.Filters != nil {
		s.ctx.SetInputValue("filters", input.Filters)
	}

	res, err := prov.Manager().Start(s.ctx.Ctx, batchflow.StartRequest{
		Input:     input,
		BatchSize: s.batchSize,
		RedisTTL:  time.Duration(s.ttlHours) * time.Hour,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetInputValue("id", input.ParentID)
	s.ctx.SetInputValue("key_redis", res.RedisKey)
	s.ctx.SetInputValue("batches_list_key", res.BatchesListKey)
	s.ctx.SetInputValue("total_batches", res.TotalBatches)
	s.ctx.SetInputValue("batch_index", 0)
	s.ctx.SetInputValue("is_last_batch", false)

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batchflow start completado",
		Data: map[string]any{
			"key_redis":     res.RedisKey,
			"id":            input.ParentID,
			"total_batches": res.TotalBatches,
		},
	})
	return nil
}

type processBatchStep struct {
	ctx               *contracts.ServiceContext
	servicePath       string
	concurrentBatches int
}

func NewProcessBatchStep() contracts.Service {
	return &processBatchStep{concurrentBatches: 1}
}

func (s *processBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["concurrent_batches"]; ok {
			s.concurrentBatches = utils.ToInt(v)
		}
	}
	if s.concurrentBatches <= 0 {
		s.concurrentBatches = 1
	}
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

	res, err := prov.Manager().ProcessBatch(s.ctx.Ctx, batchflow.ProcessRequest{
		Input:             input,
		BatchesListKey:    fmt.Sprint(utils.MustGetInputValue(s.ctx, "batches_list_key")),
		BatchIndex:        utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "batch_index", 0)),
		TotalBatches:      utils.ToInt(utils.MustGetInputValue(s.ctx, "total_batches")),
		ConcurrentBatches: s.concurrentBatches,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batch procesado",
		Data: map[string]any{
			"batch_index":       res.NextBatchIndex,
			"is_last_batch":     res.IsLastBatch,
			"processed_count":   res.ProcessedCount,
			"batches_processed": res.BatchesProcessed,
		},
	})
	return nil
}

type finalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewFinalizeStep() contracts.Service {
	return &finalizeStep{}
}

func (s *finalizeStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
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

	result, err := prov.Manager().Finalize(s.ctx.Ctx, batchflow.FinalizeRequest{
		Input:          input,
		BatchesListKey: fmt.Sprint(utils.MustGetInputValue(s.ctx, "batches_list_key")),
		TotalBatches:   utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "total_batches", 0)),
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "proceso batch finalizado",
		Data: map[string]any{
			"metadata": result.Metadata,
			"summary":  result.Summary,
		},
	})
	return nil
}

func buildInput(ctx *contracts.ServiceContext) (batchflow.Input, error) {
	input := batchflow.Input{
		RedisKey: fmt.Sprint(utils.MustGetInputValue(ctx, "key_redis")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return batchflow.Input{}, fmt.Errorf("id inválido")
	}
	if input.RedisKey == "" {
		return batchflow.Input{}, fmt.Errorf("key_redis inválida")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		input.Filters = rawFilters
	}
	return input, nil
}

func buildStartInput(ctx *contracts.ServiceContext) (batchflow.Input, error) {
	input := batchflow.Input{
		RedisKey: fmt.Sprint(utils.GetInputValueOrDefault(ctx, "key_redis", "")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return batchflow.Input{}, fmt.Errorf("id inválido")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		input.Filters = rawFilters
	}
	return input, nil
}

func markFailure(prov Provider, ctx context.Context, input batchflow.Input, err error) {
	if errors.Is(err, domain.ErrBusinessRuleViolation) || errors.Is(err, domain.ErrInvalidArgument) {
		return
	}
	_ = prov.Manager().Fail(ctx, input, err)
}

func init() {
	serviceconfig.Register("bulk/process/generic/start", NewStartStep)
	serviceconfig.Register("bulk/process/generic/process_batch", NewProcessBatchStep)
	serviceconfig.Register("bulk/process/generic/finalize", NewFinalizeStep)
}
