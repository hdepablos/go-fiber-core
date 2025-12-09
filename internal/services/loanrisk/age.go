package loanrisk

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// Age es la implementación concreta para el servicio de validación de edad.
type Age struct {
	// Contendrá una referencia al contexto que fluye por la cadena.
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewAgeService es el constructor que se registrará en nuestra fábrica.
// Devuelve el tipo de la interfaz, no el struct concreto.
func NewAgeService() contracts.Service {
	return &Age{}
}

// Init inyecta el contexto y la ruta. Cumple con la interfaz Service.
func (a *Age) Init(ctx *contracts.ServiceContext, servicePath string) {
	a.ctx = ctx
	a.servicePath = servicePath
}

// Execute contiene la lógica específica de este servicio.
func (a *Age) Execute() error {
	fmt.Println("🧮 Ejecutando servicio Age")

	// Accede a los datos de entrada de forma segura desde el contexto.
	age := a.ctx.Age
	result := map[string]any{
		"age_processed": fmt.Sprintf("Edad validada: %v", age),
		"is_adult":      age >= 18,
	}

	// Añade su resultado al mapa de resultados compartidos del contexto.
	a.ctx.Results[a.servicePath] = result
	return nil // No hay error
}

// init se ejecuta automáticamente cuando el programa arranca.
// Su única misión es registrar este servicio en el mapa central
// para que el ejecutor pueda encontrarlo por su ruta.
func init() {
	serviceconfig.Register("loanrisk/NewAgeService", NewAgeService)
}
