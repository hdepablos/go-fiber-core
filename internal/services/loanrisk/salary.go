package loanrisk

import (
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// Salary es la implementación para el servicio de validación de salario.
type Salary struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewSalaryService es el constructor que se registrará.
func NewSalaryService() contracts.Service {
	return &Salary{}
}

// Init inyecta el contexto y la ruta.
func (s *Salary) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute contiene la lógica para devolver un error crítico.
func (s *Salary) Execute() error {
	fmt.Println("💰 Ejecutando servicio Salary")

	// --- LÓGICA DE ERROR CRÍTICO ---
	// Si el salario no es válido, es un problema grave para un préstamo y debe detener todo.
	if s.ctx.Salary <= 0 {
		// Devolvemos nuestro mensaje de error específico envuelto con el tipo ErrCritical.
		return fmt.Errorf("%w: el salario debe ser un valor positivo, pero se recibió %d", domain.ErrCritical, s.ctx.Salary)
	}

	result := map[string]any{
		"salary_checked":       true,
		"salary_bracket_k_usd": s.ctx.Salary / 1000,
	}

	s.ctx.Results[s.servicePath] = result
	return nil
}

// init registra el servicio en el mapa central.
func init() {
	serviceconfig.Register("loanrisk/NewSalaryService", NewSalaryService)
}
