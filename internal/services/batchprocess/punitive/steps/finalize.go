package steps

import (
	"fmt"

	serviceRuntime "go-fiber-core/internal/services/batchprocess/punitive/runtime"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type finalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewFinalizeStep crea el step encargado del cierre global del proceso.
func NewFinalizeStep() contracts.Service {
	return &finalizeStep{}
}

// Init conserva el contexto necesario para resolver input y publicar el resultado.
func (s *finalizeStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute llama al manager.Finalize para resumir el proceso y dejar listo ParentLifecycle.End.
func (s *finalizeStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
