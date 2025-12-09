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
	fmt.Println("🔍 Ejecutando servicio de Validación Especial")

	// --- LÓGICA DE ERROR TOLERABLE ---
	// Simulamos una regla de negocio: si el cliente tiene exactamente 50 años,
	// se marca para una revisión especial, pero el flujo principal no se detiene.
	if v.ctx.Age == 50 {
		// Devolvemos nuestro mensaje de error específico envuelto con el tipo ErrTolerable.
		return fmt.Errorf("%w: cliente con %d años requiere revisión manual de promoción", domain.ErrTolerable, v.ctx.Age)
	}

	result := map[string]any{
		"special_validation_passed": true,
	}
	v.ctx.Results[v.servicePath] = result
	return nil
}

// init registra este nuevo servicio en el mapa central.
func init() {
	serviceconfig.Register("loanrisk/NewValidationService", NewValidationService)
}
