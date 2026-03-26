package loanrisk

import (
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// Validation es la implementación para un servicio de validación especial.
type Validation struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewValidationService es el constructor que se registrará.
func NewValidationService() contracts.Service {
	return &Validation{}
}

// Init inyecta el contexto y la ruta.
func (v *Validation) Init(ctx *contracts.ServiceContext, servicePath string) {
	v.ctx = ctx
	v.servicePath = servicePath
}

// Execute contiene la lógica para devolver un error tolerable.
func (v *Validation) Execute() error {
	// Servicio de ejemplo: valida una condición de negocio y puede devolver un error tolerable.
	rawAge, _ := v.ctx.GetInputValue("age")
	age := 0
	switch a := rawAge.(type) {
	case int:
		age = a
	case int64:
		age = int(a)
	case float64:
		age = int(a)
	}

	if age == 50 {
		return fmt.Errorf("%w: cliente con %d años requiere revisión manual de promoción", domain.ErrTolerable, age)
	}

	data := map[string]any{
		"special_validation_passed": true,
	}
	result := contracts.StepResult{
		Status: "ok",
		Input:  v.ctx.SnapshotInput(),
		Data:   data,
	}
	v.ctx.SetResult(v.servicePath, result)
	return nil
}

// init registra este nuevo servicio en el mapa central.
func init() {
	serviceconfig.Register("loanrisk/validation", NewValidationService)
}
