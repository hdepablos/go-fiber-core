package steps

import (
	"fmt"

	serviceRuntime "go-fiber-core/internal/services/exports/bcra/runtime"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type FinalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	fileBase    string
}

// NewFinalizeStep crea el step encargado de unir partes y publicar el archivo final.
func NewFinalizeStep() contracts.Service {
	return &FinalizeStep{}
}

// Init absorbe la configuracion final del nombre base del archivo a generar.
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

// Execute llama al manager.Finalize para ensamblar el archivo final y registrar su salida.
func (s *FinalizeStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
