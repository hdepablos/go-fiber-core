package imputation

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// CapitalInterest es la implementación del servicio.
type CapitalInterest struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewCapitalInterestService es el constructor.
func NewCapitalInterestService() contracts.Service {
	return &CapitalInterest{}
}

// Init inicializa el servicio con su contexto.
func (s *CapitalInterest) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute contiene la lógica de negocio.
func (s *CapitalInterest) Execute() error {
	// TODO: Implementar lógica aquí
	fmt.Printf("🚀 Ejecutando servicio: %s\n", s.servicePath)

	// Ejemplo de lectura de input
	// val, ok := s.ctx.GetInputValue("some_key")

	// Ejemplo de resultado
	result := contracts.StepResult{
		Status: "ok",
		Data: map[string]any{
			"executed": true,
		},
	}

	s.ctx.SetResult(s.servicePath, result)
	return nil
}

func init() {
	serviceconfig.Register("test/imputation/capital_interest", NewCapitalInterestService)
}
