package steps

import (
	"fmt"

	serviceRuntime "go-fiber-core/internal/services/batchprocess/punitivecursor/runtime"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/runtimectx"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type dispatchShardsStep struct {
	ctx            *contracts.ServiceContext
	servicePath    string
	parallelShards int
}

// NewDispatchShardsStep crea el step que reparte el trabajo inicial entre shards.
func NewDispatchShardsStep() contracts.Service {
	return &dispatchShardsStep{parallelShards: 1}
}

// Init lee cuantos shards paralelos debe lanzar este proceso.
func (s *dispatchShardsStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	s.parallelShards = resolveParallelShards(ctx)
}

// Execute hace el fan-out inicial y deja que el consumo asincrono continue fuera del request HTTP.
func (s *dispatchShardsStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
		SourceMode:     resolveSourceMode(s.ctx),
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	baseInput := s.ctx.SnapshotInput()
	for shardIndex, batchIndex := range dispatchRes.InitialBatchIndexes {
		// Cada shard recibe una copia del input base con su batch_index de arranque.
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
