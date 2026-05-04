package steps

import (
	"time"

	serviceRuntime "go-fiber-core/internal/services/batchprocess/punitivecursor/runtime"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

const (
	startExecutionKey        = "bulk/process/punitivecursor/start"
	dispatchExecutionKey     = "bulk/process/punitivecursor/dispatch_shards"
	processBatchExecutionKey = "bulk/process/punitivecursor/process_batch"
	finalizeExecutionKey     = "bulk/process/punitivecursor/finalize"
	processBatchStepOrder    = 3
)

type startStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int
	ttlHours    int
	sourceMode  string
}

// Register publica los steps del proceso en el runtime de serviceconfig.
func Register() {
	serviceconfig.Register(startExecutionKey, NewStartStep)
	serviceconfig.Register(dispatchExecutionKey, NewDispatchShardsStep)
	serviceconfig.Register(processBatchExecutionKey, NewProcessBatchStep)
	serviceconfig.Register(finalizeExecutionKey, NewFinalizeStep)
}

// NewStartStep crea el step que prepara el run y carga los batches iniciales.
func NewStartStep() contracts.Service {
	return &startStep{
		batchSize: 500,
		ttlHours:  24,
		sourceMode: batchflow.SourceModeCursor,
	}
}

// Init absorbe la configuracion del step definida en el seeder o version del proceso.
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
		s.sourceMode = resolveSourceMode(s.ctx)
	}
}

// Execute inicia el flujo, llama al manager.Start y deja en contexto las claves de redis y paginacion.
func (s *startStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
		Input:      input,
		BatchSize:  s.batchSize,
		RedisTTL:   time.Duration(s.ttlHours) * time.Hour,
		SourceMode: s.sourceMode,
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
	s.ctx.SetInputValue("source_mode", s.sourceMode)

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batchflow start completado",
		Data: map[string]any{
			"key_redis":     res.RedisKey,
			"id":            input.ParentID,
			"total_batches": res.TotalBatches,
			"source_mode":   s.sourceMode,
		},
	})
	return nil
}
