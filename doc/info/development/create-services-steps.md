# Guía de Creación de Servicios (Steps)

Esta guía explica cómo crear nuevos servicios ("steps") para ser utilizados en los flujos de procesos (Roadmaps).

## 🛠️ Comando Automatizado (Recomendado)

Hemos creado un comando para automatizar todo el proceso. Simplemente ejecuta:

```bash
make create-step name=carpeta/servicio
```

Esto generará el archivo `.go` con todo el boilerplate necesario e inyectará automáticamente los imports en `cmd/api/main.go` y `cmd/cmd-cli/main.go`.

### Convención de Nombres (Singular vs Plural)

En Go, la convención es usar nombres de paquetes (carpetas) cortos, concisos y preferiblemente en **SINGULAR**, ya que representan un dominio o concepto único.

*   ✅ **Correcto (Singular):** `test/concurrent/step1`
    *   Crea: `internal/services/test/concurrent/step1.go`
    *   Paquete: `package concurrent`
    *   Uso: `concurrent.NewStep1Service`

*   ✅ **Correcto (Singular):** `credit/validation`
    *   Crea: `internal/services/credit/validation.go`
    *   Paquete: `package credit`
    *   Uso: `credit.NewValidationService`

*   ❌ **Evitar (Plural):** `credits/validation` (salvo que sea estrictamente una colección de utilidades)

### Soporte para Rutas Profundas y Nombres Compuestos

El comando soporta cualquier nivel de profundidad y convierte nombres con guiones bajos a CamelCase automáticamente.

```bash
make create-step name=credit/imputation/capital_interest
```
*   Archivo: `internal/services/credit/imputation/capital_interest.go`
*   Paquete: `package imputation`
*   Struct Generado: `type CapitalInterest struct { ... }`
*   Constructor: `NewCapitalInterestService()`
*   Registro: `serviceconfig.Register("credit/imputation/capital_interest", NewCapitalInterestService)`

---

## 📝 Creación Manual (Si prefieres no usar el comando)

Si necesitas hacerlo manualmente, sigue estos pasos:

### 1. Crear el Archivo del Servicio

Crea un nuevo archivo `.go` en `internal/services/<tu_dominio>/<tu_servicio>.go`.

Ejemplo: `internal/services/credit/check.go`

```go
package credit

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// CreditCheck es la estructura de tu servicio
type CreditCheck struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// Constructor requerido
func NewCreditCheckService() contracts.Service {
	return &CreditCheck{}
}

// Init inicializa el contexto
func (s *CreditCheck) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute contiene tu lógica de negocio
func (s *CreditCheck) Execute() error {
	fmt.Printf("🚀 Ejecutando servicio: %s\n", s.servicePath)

	// Tu lógica aquí...

	// Guardar resultado
	result := contracts.StepResult{
		Status: "ok",
		Data: map[string]any{
			"executed": true,
		},
	}
	
	s.ctx.SetResult(s.servicePath, result)
	return nil
}

// Auto-registro
func init() {
	serviceconfig.Register("credit/check", NewCreditCheckService)
}
```

### 2. Registrar el Servicio (Wiring)

Para que el binario compile e incluya tu nuevo servicio (ya que Go elimina código no usado), debes importarlo con un "blank import" (`_`) en el `main.go`.

Edita `cmd/api/main.go` y `cmd/cmd-cli/main.go`:

```go
import (
    // ... otros imports
    _ "go-fiber-core/internal/services/credit" // <--- Agrega esta línea
)
```

> **Nota:** El comando `make create-step` hace este paso automáticamente.
