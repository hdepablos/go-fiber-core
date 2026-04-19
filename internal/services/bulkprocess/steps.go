package bulkprocess

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/runtimectx"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

const (
	startExecutionKey        = "bulk/process/generic/start"
	dispatchExecutionKey     = "bulk/process/generic/dispatch_shards"
	processBatchExecutionKey = "bulk/process/generic/process_batch"
	finalizeExecutionKey     = "bulk/process/generic/finalize"
	processBatchStepOrder    = 3
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

type dispatchShardsStep struct {
	ctx            *contracts.ServiceContext
	servicePath    string
	parallelShards int
}

func NewDispatchShardsStep() contracts.Service {
	return &dispatchShardsStep{parallelShards: 1}
}

func (s *dispatchShardsStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	s.parallelShards = resolveParallelShards(ctx)
}

func (s *dispatchShardsStep) Execute() error {
	prov, err := ProviderFromContext(s.ctx.Ctx)
	if err != nil {
		return err
	}
	dispatcherSvc, ok := runtimectx.Dispatcher(s.ctx.Ctx)
	if !ok {
		return fmt.Errorf("dispatcher no disponible en contexto")
	}
	input, err := buildInput(s.ctx)
	if err != nil {
		return err
	}

	totalBatches := utils.ToInt(utils.MustGetInputValue(s.ctx, "total_batches"))
	dispatchRes, err := prov.Manager().DispatchShards(s.ctx.Ctx, batchflow.DispatchRequest{
		Input:          input,
		TotalBatches:   totalBatches,
		ParallelShards: s.parallelShards,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	baseInput := s.ctx.SnapshotInput()
	for shardIndex, batchIndex := range dispatchRes.InitialBatchIndexes {
		shardInput := cloneInput(baseInput)
		shardInput["batch_index"] = batchIndex
		shardInput["shard_index"] = shardIndex
		shardInput["total_shards"] = dispatchRes.TotalShards
		shardInput["is_shard_complete"] = false

		childCtx := contracts.NewServiceContextFromInput(s.ctx.Ctx, shardInput)
		if err := dispatcherSvc.DispatchStep(s.ctx.Ctx, processBatchExecutionKey, processBatchStepOrder, contracts.ExecutionPolicy{}, nil, childCtx); err != nil {
			return err
		}
	}

	s.ctx.SetInputValue("__stop_chain", true)
	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "fan-out de shards despachado",
		Data: map[string]any{
			"parallel_shards":   dispatchRes.TotalShards,
			"dispatched_shards": len(dispatchRes.InitialBatchIndexes),
			"__stop_chain":      true,
		},
	})
	return nil
}

type processBatchStep struct {
	ctx               *contracts.ServiceContext
	servicePath       string
	concurrentBatches int
	dispatchPacing    batchflow.DispatchPacingConfig
	initErr           error
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
		s.dispatchPacing, s.initErr = batchflow.ValidateDispatchPacingStepConfig(s.ctx.CurrentStepConfig)
	}
	if s.concurrentBatches <= 0 {
		s.concurrentBatches = 1
	}
}

func (s *processBatchStep) Execute() error {
	if s.initErr != nil {
		return s.initErr
	}
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
		ShardIndex:        utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "shard_index", 0)),
		TotalShards:       utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "total_shards", 1)),
		DispatchPacing:    s.dispatchPacing,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batch procesado",
		Data: map[string]any{
			"batch_index":               res.NextBatchIndex,
			"is_last_batch":             res.IsLastBatch,
			"is_shard_complete":         res.IsShardComplete,
			"processed_count":           res.ProcessedCount,
			"batches_processed":         res.BatchesProcessed,
			"shard_index":               res.ShardIndex,
			"total_shards":              res.TotalShards,
			"completed_shards":          res.CompletedShards,
			"should_dispatch_next_step": res.ShouldDispatchNextStep,
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

func resolveParallelShards(ctx *contracts.ServiceContext) int {
	if ctx == nil || ctx.CurrentStepConfig == nil {
		return 1
	}
	if v, ok := ctx.CurrentStepConfig["parallel_shards"]; ok {
		if parsed := utils.ToInt(v); parsed > 0 {
			return parsed
		}
	}
	if rawMode, ok := ctx.CurrentStepConfig["execution_mode"].(map[string]any); ok {
		if v, ok := rawMode["parallel_shards"]; ok {
			if parsed := utils.ToInt(v); parsed > 0 {
				return parsed
			}
		}
	}
	return 1
}

func cloneInput(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func init() {
	serviceconfig.Register(startExecutionKey, NewStartStep)
	serviceconfig.Register(dispatchExecutionKey, NewDispatchShardsStep)
	serviceconfig.Register(processBatchExecutionKey, NewProcessBatchStep)
	serviceconfig.Register(finalizeExecutionKey, NewFinalizeStep)
}
