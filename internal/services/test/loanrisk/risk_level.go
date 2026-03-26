package loanrisk

import (
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
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
	// Servicio de ejemplo: calcula un nivel de riesgo en base a edad y salario.
	if _, ok := p.ctx.GetInputValue("is_renovation"); !ok {
		return fmt.Errorf("%w: missing required input key 'is_renovation' for RiskLevel", domain.ErrMissingRequiredKey)
	}

	salary := utils.GetIntInput(p.ctx, "salary")
	age := utils.GetIntInput(p.ctx, "age")

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
	serviceconfig.Register("loanrisk/risk_level", NewRiskLevelService)
}
