package imputation

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// CapitalInterest es la implementación del servicio.
type capitalInterest struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewCapitalInterestService es el constructor.
func NewCapitalInterestService() contracts.Service {
	return &capitalInterest{}
}

// Init inicializa el servicio con su contexto.
func (s *capitalInterest) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute contiene la lógica de negocio.
func (s *capitalInterest) Execute() error {
	// Servicio de ejemplo para pruebas: en un caso real aquí se calcularía capital/interés,
	// leyendo inputs del contexto y escribiendo resultados en StepResult.

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
