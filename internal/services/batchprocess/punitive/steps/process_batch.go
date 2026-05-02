package steps

import (
	serviceRuntime "go-fiber-core/internal/services/batchprocess/punitive/runtime"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
	"fmt"
)

type processBatchStep struct {
	ctx               *contracts.ServiceContext
	servicePath       string
	concurrentBatches int
	dispatchPacing    batchflow.DispatchPacingConfig
	initErr           error
}

// NewProcessBatchStep crea el step que procesa lotes y coordina el auto-dispatch del siguiente.
func NewProcessBatchStep() contracts.Service {
	return &processBatchStep{concurrentBatches: 1}
}

// Init absorbe la concurrencia y la configuracion opcional de pacing para el step.
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

// Execute entrega el lote actual al manager para que invoque Processor.ProcessBatch.
func (s *processBatchStep) Execute() error {
	if s.initErr != nil {
		return s.initErr
	}
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
