package loanrisk

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// RiskLevel es la implementación para el servicio de cálculo de riesgo.
type RiskLevel struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewRiskLevelService es el constructor que se registrará.
func NewRiskLevelService() contracts.Service {
	return &RiskLevel{}
}

// Init inyecta el contexto y la ruta.
func (p *RiskLevel) Init(ctx *contracts.ServiceContext, servicePath string) {
	p.ctx = ctx
	p.servicePath = servicePath
}

// Execute realiza un cálculo simple de riesgo.
func (p *RiskLevel) Execute() error {
	fmt.Println("📊 Ejecutando servicio RiskLevel")

	rawSalary, _ := p.ctx.GetInputValue("salary")
	salary := 0
	switch v := rawSalary.(type) {
	case int:
		salary = v
	case int64:
		salary = int(v)
	case float64:
		salary = int(v)
	}

	rawAge, _ := p.ctx.GetInputValue("age")
	age := 0
	switch v := rawAge.(type) {
	case int:
		age = v
	case int64:
		age = int(v)
	case float64:
		age = int(v)
	}

	risk := "bajo"
	if salary < 50000 && age > 60 {
		risk = "alto"
	} else if salary < 80000 {
		risk = "medio"
	}

	data := map[string]any{
		"calculated_risk": risk,
	}
	result := contracts.StepResult{
		Status: "ok",
		Input:  p.ctx.SnapshotInput(),
		Data:   data,
	}
	p.ctx.SetResult(p.servicePath, result)
	return nil
}

// init registra el servicio en el mapa central.
func init() {
	serviceconfig.Register("loanrisk/NewRiskLevelService", NewRiskLevelService)
}
