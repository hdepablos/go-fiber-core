# Guía de Uso del Process Lifecycle Executor

Esta guía explica cómo ejecutar un proceso de negocio completo utilizando el motor de ciclo de vida (`ProcessLifecycleService`) y cómo consumir los resultados de forma sencilla.

## 1. Concepto General

El ejecutor centraliza la lógica de negocio en "Pasos" configurables. Para invocar un proceso, no llamas a los servicios individuales, sino que le pides al motor que ejecute un "Tipo de Proceso" (ej: Evaluación de Riesgo, Validación de Documentos) pasando un conjunto de datos de entrada (`Input`).

## 2. Estructura del Request

Utilizamos el DTO `requests.RunProcessRequest` para estandarizar la llamada.

```go
type RunProcessRequest struct {
    ProcessTypeID            int64          // ID del tipo de proceso (ej: 1=Loan, 2=Validation)
    SedeID                   int64          // Sede contextual (default: 1)
    Roadmap                  int            // (Opcional) Segmento de pasos a ejecutar
    Input                    map[string]any // Datos de negocio iniciales
    OverrideProcessVersionID *int64         // (Opcional) Forzar una versión específica
    OperatorID               int64          // (Interno) ID del operador, inyectado automáticamente
}
```

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

### Paso 5: Consumir Resultados (`GetAll`)
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
