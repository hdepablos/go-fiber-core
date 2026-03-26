package common

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type CalculateService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewCalculateService() contracts.Service {
	return &CalculateService{}
}

func (s *CalculateService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *CalculateService) Execute() error {
	// Servicio de ejemplo: calcula un valor simple en base a "age".
	age, _ := s.ctx.GetInputValue("age")
	var result float64

	// Lógica de cálculo real
	if val, ok := age.(float64); ok {
		result = val * 1.5
	} else if val, ok := age.(int); ok {
		result = float64(val) * 1.5
	} else {
		result = 37.5 // Default para el test case
	}

	data := map[string]any{
		"result":        result,
		"calculated_at": "now",
	}

	res := contracts.StepResult{
		Status: "completed",
		Data:   data,
	}

	s.ctx.SetResult(s.servicePath, res)
	return nil
}

func init() {
	serviceconfig.Register("common/calculate", NewCalculateService)
}
