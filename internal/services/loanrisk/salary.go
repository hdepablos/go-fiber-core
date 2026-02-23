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

	rawSalary, ok := s.ctx.GetInputValue("salary")
	if !ok {
		return fmt.Errorf("%w: missing required input key 'salary' for Salary", domain.ErrMissingRequiredKey)
	}

	salary := 0
	switch v := rawSalary.(type) {
	case int:
		salary = v
	case int64:
		salary = int(v)
	case float64:
		salary = int(v)
	}

	// Leer min_salary desde la configuración del step (por defecto 1)
	minSalary := 1
	if cfg := s.ctx.CurrentStepConfig; cfg != nil {
		if v, ok := cfg["min_salary"]; ok {
			switch n := v.(type) {
			case int:
				minSalary = n
			case int64:
				minSalary = int(n)
			case float64:
				minSalary = int(n)
			}
		}
	}

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
	serviceconfig.Register("loanrisk/NewSalaryService", NewSalaryService)
}
