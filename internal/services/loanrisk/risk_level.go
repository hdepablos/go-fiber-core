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

	// Lógica de ejemplo: el riesgo es alto si el salario es bajo y la edad es avanzada.
	risk := "bajo"
	if p.ctx.Salary < 50000 && p.ctx.Age > 60 {
		risk = "alto"
	} else if p.ctx.Salary < 80000 {
		risk = "medio"
	}

	result := map[string]any{
		"calculated_risk": risk,
	}
	p.ctx.Results[p.servicePath] = result
	return nil
}

// init registra el servicio en el mapa central.
func init() {
	serviceconfig.Register("loanrisk/NewRiskLevelService", NewRiskLevelService)
}
