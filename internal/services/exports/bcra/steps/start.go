package steps

import (
	"fmt"
	"time"

	serviceRuntime "go-fiber-core/internal/services/exports/bcra/runtime"
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
}

// Register publica los steps del export en el runtime de serviceconfig.
func Register() {
	serviceconfig.Register("bulk/export/bcra/start", NewStartStep)
	serviceconfig.Register("bulk/export/bcra/process_batch", NewProcessBatchStep)
	serviceconfig.Register("bulk/export/bcra/finalize", NewFinalizeStep)
}

// NewStartStep crea el step que prepara la corrida y reserva el estado temporal del export.
func NewStartStep() contracts.Service {
	return &StartStep{
		batchSize: 5000,
		ttlHours:  24,
	}
}

// Init absorbe la configuracion del step definida en el seeder o version del proceso.
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
	}
}

// Execute inicia el export, genera las claves runtime y deja listo el primer batch para procesar.
func (s *StartStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
