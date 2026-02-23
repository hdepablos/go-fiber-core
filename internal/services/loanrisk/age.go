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

	rawAge, _ := a.ctx.GetInputValue("age")
	age := 0
	switch v := rawAge.(type) {
	case int:
		age = v
	case int64:
		age = int(v)
	case float64:
		age = int(v)
	}
	// Leer min_age desde la configuración del step (por defecto 18)
	minAge := 18
	if cfg := a.ctx.CurrentStepConfig; cfg != nil {
		if v, ok := cfg["min_age"]; ok {
			switch n := v.(type) {
			case int:
				minAge = n
			case int64:
				minAge = int(n)
			case float64:
				minAge = int(n)
			}
		}
	}
	data := map[string]any{
		"age_processed": fmt.Sprintf("Edad validada: %v", age),
		"min_age":       minAge,
		"is_adult":      age >= minAge,
	}
	result := contracts.StepResult{
		Status: "ok",
		Input:  a.ctx.SnapshotInput(),
		Data:   data,
	}
	a.ctx.SetResult(a.servicePath, result)
	return nil // No hay error
}

// init se ejecuta automáticamente cuando el programa arranca.
// Su única misión es registrar este servicio en el mapa central
// para que el ejecutor pueda encontrarlo por su ruta.
func init() {
	serviceconfig.Register("loanrisk/NewAgeService", NewAgeService)
}
