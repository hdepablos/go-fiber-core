package loanrisk

import (
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
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
	// Servicio de ejemplo: valida salario mínimo y propaga datos al input.
	salary := utils.GetIntInput(s.ctx, "salary")
	minSalary := utils.GetIntConfig(s.ctx.CurrentStepConfig, "min_salary", 1)

	if salary < minSalary {
		return fmt.Errorf("%w: salario %d menor al mínimo permitido %d", domain.ErrValueOutOfRange, salary, minSalary)
	}

	data := map[string]any{
		"salary_checked":       true,
		"min_salary":           minSalary,
		"salary_bracket_k_usd": salary / 1000,
	}

	for k, v := range data {
		s.ctx.SetInputValue(k, v)
	}

	result := contracts.StepResult{
		Status: "ok",
		Input:  s.ctx.SnapshotInput(),
		Data:   data,
	}
	s.ctx.SetResult(s.servicePath, result)
	return nil
}

// init registra el servicio en el mapa central.
func init() {
	serviceconfig.Register("loanrisk/salary", NewSalaryService)
}
