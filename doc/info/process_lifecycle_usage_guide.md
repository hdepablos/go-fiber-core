# Guía de Uso del Process Lifecycle Executor

Esta guía explica cómo ejecutar un proceso de negocio completo utilizando el motor de ciclo de vida (`ProcessLifecycleService`) y cómo consumir los resultados de forma sencilla.

## 1. Concepto General

El ejecutor centraliza la lógica de negocio en "Pasos" configurables. Para invocar un proceso, no llamas a los servicios individuales, sino que le pides al motor que ejecute un "Tipo de Proceso" (ej: Evaluación de Riesgo, Validación de Documentos) pasando un conjunto de datos de entrada (`Input`).

## 2. Estructura del Request y Validaciones

Utilizamos el DTO `requests.RunProcessRequest` con validaciones estrictas para garantizar la integridad de los datos. Todos los campos son obligatorios o tienen valores por defecto seguros.

```go
type RunProcessRequest struct {
    ProcessTypeID            int64          // Requerido (> 0)
    SedeID                   *int64         // Requerido (No Nulo)
    OverrideProcessVersionID *int64         // Requerido (No Nulo)
    Roadmap                  *int           // Requerido (No Nulo)
    Input                    map[string]any // Requerido (No Nulo)
    OperatorID               int64          // (Interno) Inyectado automáticamente
}
```

### Reglas de Validación y Garantías del Motor

El método `Run` implementa una validación **ESTRICTA**. El motor **rechazará** cualquier petición que omita campos o envíe valores nulos (`nil`). El cliente es responsable de enviar valores explícitos, incluso si son "vacíos" (como `0` o `{}`).

| Campo | Validación / Regla | Comportamiento si es inválido o falta |
| :--- | :--- | :--- |
| **`process_type_id`** | Debe ser `> 0` | Retorna error `ErrInvalidArgument`. |
| **`sede_id`** | **Requerido**. No puede ser `nil`. | Retorna error `ErrInvalidArgument`.<br>Si no aplica, enviar `0`. |
| **`override_process_version_id`** | **Requerido**. No puede ser `nil`. | Retorna error `ErrInvalidArgument`.<br>Para producción, enviar `0`. |
| **`roadmap`** | **Requerido**. No puede ser `nil`. | Retorna error `ErrInvalidArgument`.<br>Si no aplica, enviar `0`. |
| **`input`** | **Requerido**. No puede ser `nil`. | Retorna error `ErrInvalidArgument`.<br>Si no hay datos, enviar `{}` (mapa vacío). |

> **Nota Importante:** Esta validación aplica para llamadas externas (API) y ejecuciones internas. El objetivo es eliminar ambigüedades: un campo faltante se considera un error de integración, no un valor por defecto.

## 3. Pasos para ejecutar un proceso

### Paso 1: Inyectar el servicio
Asegúrate de tener inyectada la interfaz `processlifecycle.Service` en tu handler o servicio.

```go
type MyHandler struct {
    service processlifecycle.Service
}
```

### Paso 2: Preparar los datos (Input)
Crea un mapa con los datos que el proceso necesita para funcionar.

```go
inputData := map[string]any{
    "amount":       50000,
    "salary":       2500,
    "age":          30,
    "history_good": true,
    // Puedes agregar cualquier dato que los pasos necesiten
}
```

### Paso 3: Configurar el Request
Instancia `RunProcessRequest`.

```go
req := requests.RunProcessRequest{
    ProcessTypeID: 2, // ID del proceso (ej: Loan Risk)
    SedeID:        1,
    Input:         inputData,
    // Roadmap: 0, // Opcional, si usas roadmaps
}
```

### Paso 4: Ejecutar (`Run`)
Llama al método `Run`.

```go
versionID, serviceCtx, err := h.service.Run(ctx, req)
if err != nil {
    // Manejar error (ver sección de errores)
    return err
}
```

## 4. Manejo de Errores (Error Handling)

El sistema utiliza un conjunto estandarizado de errores para comunicar problemas de validación o lógica de negocio.

### Estructura de Respuesta de Error

Cuando ocurre un error, la API retorna un JSON con el siguiente formato:

```json
{
  "status": "error",
  "message": "Error ejecutando proceso lifecycle",
  "data": {
    "error": {
      "code": "CODIGO_DE_ERROR",
      "message": "Descripción detallada del problema"
    }
  }
}
```

### Tipos de Errores Comunes

| Código | Status HTTP | Significado | Ejemplo de Mensaje |
| :--- | :--- | :--- | :--- |
| **`MISSING_REQUIRED_KEY`** | `422 Unprocessable Entity` | Faltan datos obligatorios en el `input` para que un paso pueda ejecutarse. | `"falta una clave requerida en el input: claves faltantes [salary, age] para el servicio 'loanrisk/Validator'"` |
| **`BUSINESS_RULE_VIOLATION`** | `422 Unprocessable Entity` | Los datos existen, pero no cumplen con una regla de negocio específica (ej: edad mínima, saldo insuficiente). | `"violación de regla de negocio: la edad 15 es menor a la mínima requerida (18)"` |
| **`INVALID_ARGUMENT`** | `422 Unprocessable Entity` | Argumentos del request inválidos (ej: IDs negativos, tipos de datos incorrectos). | `"argumento inválido"` |
| **`PROCESS_VERSION_NOT_FOUND`** | `404 Not Found` | No se encontró una versión activa del proceso para ejecutar. | `"el recurso solicitado no fue encontrado"` |
| **`SEDE_NOT_FOUND`** | `404 Not Found` | El `sede_id` proporcionado no existe en la base de datos. | `"la sede especificada no existe"` |
| **`ROADMAP_NOT_FOUND`** | `404 Not Found` | El `roadmap` proporcionado no existe en la base de datos. | `"el roadmap especificado no existe"` |
| **`OVERRIDE_VERSION_NOT_FOUND`** | `404 Not Found` | El `override_process_version_id` proporcionado no existe. | `"la versión de proceso específica no existe"` |
| **`CRITICAL_ERROR`** | `500 Internal Server Error` | Fallo inesperado o crítico del sistema (ej: error de base de datos). | `"error crítico"` |

### Ejemplo: Lanzar un Error de Negocio desde un Servicio

Si estás desarrollando un nuevo servicio (Paso), así es como debes reportar una violación de regla de negocio:

```go
import (
    "fmt"
    "go-fiber-core/internal/domain"
)

func (s *MyService) Execute() error {
    age := s.ctx.GetInputValue("age").(int)
    minAge := 18

    if age < minAge {
        // Envuelve domain.ErrBusinessRuleViolation con tu mensaje específico
        return fmt.Errorf("%w: edad %d insuficiente (mínimo %d)", 
            domain.ErrBusinessRuleViolation, age, minAge)
    }

    return nil
}
```

---

## 5. Consumir Resultados (`GetAll`)
Usa el método `GetAll()` del contexto devuelto para obtener un mapa plano con todos los resultados.

```go
// GetAll() aplana los resultados de todos los pasos en un solo mapa
resultados := serviceCtx.GetAll()

// Acceder a los valores (haciendo type assertion)
score, ok := resultados["score"].(int)
if !ok {
    score = 0 // Valor por defecto si no existe o no es int
}

aprobado, ok := resultados["approved"].(bool)
if !ok {
    aprobado = false
}

// O forma corta si confías en el tipo (pero cuidado, panic si falla sin el segundo retorno)
riesgo, _ := resultados["risk_level"].(string)

fmt.Printf("Resultado: Score=%d, Aprobado=%v\n", score, aprobado)
```

> **Nota sobre Type Assertion (`.(T)`):**
> Como `GetAll()` devuelve un `map[string]any`, Go no sabe de qué tipo es cada valor.
> Usamos `.(tipo)` para decirle al compilador "trata esto como un `int`, `bool`, etc.".
> - `val, ok := map["key"].(int)`: Es seguro. Si falla, `ok` es false y no explota.
> - `val := map["key"].(int)`: Es inseguro. Si el valor no es `int`, el programa hará **panic**.

## 4. Ejemplo Completo

```go
package mypackage

import (
    "context"
    "fmt"
    "go-fiber-core/internal/dtos/requests"
    "go-fiber-core/internal/services/processlifecycle"
)

func ProcesarSolicitudCredito(ctx context.Context, svc processlifecycle.Service) error {
    // 1. Datos de entrada
    datos := map[string]any{
        "monto_solicitado": 10000,
        "salario_mensual":  2000,
        "es_cliente":       true,
    }

    // 2. Crear Request
    peticion := requests.RunProcessRequest{
        ProcessTypeID: 10, // Supongamos ID 10 es "Crédito Consumo"
        SedeID:        1,
        Input:         datos,
    }

    // 3. Ejecutar
    versionID, contexto, err := svc.Run(ctx, peticion)
    if err != nil {
        return fmt.Errorf("error ejecutando proceso: %w", err)
    }

    // 4. Leer Resultados
    // Supongamos que el proceso genera "tasa_interes" y "estado_solicitud"
    resultados := contexto.GetAll()

    tasa, ok := resultados["tasa_interes"].(float64)
    if !ok {
        tasa = 0.0
    }
    
    estado, _ := resultados["estado_solicitud"].(string)

    fmt.Printf("Proceso v%d finalizado. Estado: %s, Tasa: %.2f%%\n", versionID, estado, tasa)
    
    return nil
}
```

## 5. Manejo de Errores

El método `Run` puede devolver los siguientes errores de dominio:

- `domain.ErrNotFound`:
  - No existe una versión activa para el `ProcessTypeID` y `SedeID` indicados.
  - El tipo de proceso está archivado.

- `domain.ErrMissingRequiredKey`:
  - Falta una key obligatoria en el `Input` (validada por la configuración del paso).

- `domain.ErrValueOutOfRange`:
  - Algún valor del input no cumple con las reglas de validación del paso.

- `domain.ErrInternal`:
  - Error inesperado (BD, configuración corrupta, pánico en un servicio).
