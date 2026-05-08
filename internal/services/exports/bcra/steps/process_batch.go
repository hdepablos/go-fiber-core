package steps

import (
	"fmt"

	serviceRuntime "go-fiber-core/internal/services/exports/bcra/runtime"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type ProcessBatchStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewProcessBatchStep crea el step que procesa y sube cada parte temporal del archivo.
func NewProcessBatchStep() contracts.Service {
	return &ProcessBatchStep{}
}

// Init conserva el contexto necesario para resolver input y publicar el resultado del batch.
func (s *ProcessBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute entrega el batch actual al manager para construir y almacenar una parte temporal del archivo.
func (s *ProcessBatchStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
