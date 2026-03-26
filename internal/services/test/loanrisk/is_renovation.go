package loanrisk

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// IsRenovation es un servicio de ejemplo para completar el flujo.
type IsRenovation struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewIsRenovationService es el constructor que se registrará.
func NewIsRenovationService() contracts.Service {
	return &IsRenovation{}
}

// Init inyecta el contexto y la ruta.
func (r *IsRenovation) Init(ctx *contracts.ServiceContext, servicePath string) {
	r.ctx = ctx
	r.servicePath = servicePath
}

// Execute contiene la lógica del servicio.
func (r *IsRenovation) Execute() error {
	// Servicio de ejemplo: setea is_renovation en el input para que pasos posteriores lo usen.
	r.ctx.SetInputValue("is_renovation", true)

	data := map[string]any{
		"renovation_check": true,
	}
	result := contracts.StepResult{
		Status: "ok",
		Input:  r.ctx.SnapshotInput(),
		Data:   data,
	}
	r.ctx.SetResult(r.servicePath, result)
	return nil
}

// init registra el servicio en el mapa central.
func init() {
	serviceconfig.Register("loanrisk/is_renovation", NewIsRenovationService)
}
