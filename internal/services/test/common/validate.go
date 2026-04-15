package common

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type validateService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewValidateService() contracts.Service {
	return &validateService{}
}

func (s *validateService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *validateService) Execute() error {
	// Lógica de validación real (simple)
	// Podríamos validar inputs requeridos aquí, pero el framework ya lo hace.
	// Simulamos una validación de negocio.

	data := map[string]any{
		"valid":        true,
		"file_valid":   true,
		"validated_at": "now",
	}

	result := contracts.StepResult{
		Status: "completed",
		Data:   data,
	}

	s.ctx.SetResult(s.servicePath, result)
	return nil
}

func init() {
	serviceconfig.Register("common/validate", NewValidateService)
}
