# Runtime del Process Lifecycle y Motor de Servicios

Este documento describe cómo funciona en tiempo de ejecución el **Process Lifecycle Manager** y el motor de ejecución de servicios configurables, con foco en:

- `ServiceContext` y el JSON de entrada.
- `StepResult` y los resultados por servicio.
- Configuración de steps (`process_steps.config` / seeder).
- Validación de `required_keys`.
- Manejo de errores (`ErrCritical`, `ErrTolerable`, `error_tolerance`).
- Comandos CLI de ejemplo.

---

## 1. ServiceContext: bolsa de datos de negocio

Archivo: `internal/services/serviceconfig/contracts/service.go`

```go
type ServiceContext struct {
    Ctx               context.Context `json:"-"`
    mu                sync.Mutex
    Input             map[string]any  `json:"input,omitempty"`
    Results           map[string]any  `json:"results"`
    CurrentStepConfig map[string]any  `json:"-"`
}
```

- `Input`:
  - Es la **bolsa global de datos de negocio** del proceso.
  - Nace del JSON de entrada (CLI, API, etc.) y se va enriqueciendo a medida que se ejecutan los servicios.
  - Todos los servicios leen y escriben sobre este JSON usando:
    - `GetInputValue(key string) (any, bool)`
    - `SetInputValue(key string, value any)`
    - `SnapshotInput() map[string]any` (fotografía del input actual, útil para debug).

- `Results`:
  - Mapa con los resultados por servicio, indexado por `servicePath` (por ejemplo `loanrisk/age`).
  - Se llena con `SetResult(path string, result StepResult)`.

- `CurrentStepConfig`:
  - Configuración **específica del step** que se está ejecutando.
  - Viene del campo `config` de la tabla `process_steps` (JSONB).
  - El executor lo parsea y lo asigna antes de llamar a `Execute()` del servicio.

> **Nota sobre Roadmap**: El motor filtra los pasos disponibles según el campo `roadmap` de `process_steps`. Solo se ejecutan los servicios que coinciden con el roadmap solicitado en `Run`.

### Constructores relevantes

- Desde age/salary (modo demo / compatibilidad):

```go
func NewServiceContext(age, salary int) *ServiceContext
func NewServiceContextWithCtx(ctx context.Context, age, salary int) *ServiceContext
```

- Desde un JSON genérico de entrada:

```go
func NewServiceContextFromInput(ctx context.Context, input map[string]any) *ServiceContext
```

Este es el constructor recomendado para flujos reales: el JSON de negocio entra una vez, se carga en `Input` y fluye por todos los servicios.

- **Acceso a Resultados (GetAll)**:

```go
// Retorna un mapa plano con todos los valores de `Data` de todos los pasos ejecutados.
func (c *ServiceContext) GetAll() map[string]any
```
Ex: `{"score": 85, "approved": true}`.

---

## 2. StepResult: foto de cada servicio

Archivo: `internal/services/serviceconfig/contracts/service.go`

```go
type StepResult struct {
    Status  string         `json:"status"`
    Message string         `json:"message,omitempty"`
    Input   map[string]any `json:"input,omitempty"`
    Data    map[string]any `json:"data,omitempty"`
}
```

Semántica:

- `Input`:
  - Fotografía opcional del `ServiceContext.Input` **en el momento de ejecutar el servicio**.
  - Es útil para debug / auditoría (ver qué datos veía cada servicio).
  - Cada servicio decide qué copiar (todo, solo algunas claves, o nada).

- `Data`:
  - Resultado de negocio generado por el servicio (ej: `is_adult`, `calculated_risk`, etc.).
  - Es la parte “de salida” del servicio.

Importante:

- El **input real** del siguiente servicio es siempre `ServiceContext.Input`, no `StepResult.Input`.
- `StepResult.Input` es solo para observabilidad.

---

## 3. Configuración de Steps (process_steps.config)

La tabla `process_steps` tiene un campo `config` (JSONB) donde se parametriza el comportamiento de cada step.

Ejemplo de configuración generada por el seeder de Loan Risk:

Archivo: `internal/database/seeders/process_lifecycle_manager_seeder.go`

```go
loanSteps := []stepDef{
    {
        Order:        1,
        Name:         "Age validation",
        ExecutionKey: "loanrisk/NewAgeService",
        Config:       `{"error_tolerance":"inherit","required_keys":["age"],"min_age":40}`,
    },
    {
        Order:        3,
        Name:         "Salary validation",
        ExecutionKey: "loanrisk/NewSalaryService",
        Config:       `{"error_tolerance":"critical","required_keys":["salary"],"min_salary":2500000}`,
    },
    // ...
}
```

Campos relevantes:

- `error_tolerance`:
  - `"inherit"` (por defecto): comportamiento estándar, los errores se manejan con la política base.
  - `"tolerable"`: errores se loguean y la cadena continúa.
  - `"critical"`: errores se consideran críticos y cortan la cadena.

- `required_keys`:
  - Arreglo de strings.
  - Lista de claves que deben existir en `ServiceContext.Input` antes de ejecutar el servicio.
  - Si falta alguna, el executor genera un `ErrCritical` y no llama al servicio.

- Otros campos específicos del servicio:
  - `min_age`, `min_salary`, etc.
  - El propio servicio los lee vía `ctx.CurrentStepConfig`.

---

## 4. Validación de required_keys

Archivo: `internal/services/serviceconfig/executor.go`

Antes de ejecutar un servicio, el executor valida las `RequiredKeys`:

```go
if len(serviceConfig.RequiredKeys) > 0 && svcCtx != nil {
    for _, key := range serviceConfig.RequiredKeys {
        if _, ok := svcCtx.GetInputValue(key); !ok {
            execErr = fmt.Errorf(
                "missing required key '%s' for service '%s': %w",
                key,
                serviceConfig.Path,
                domain.ErrCritical,
            )
            break
        }
    }
}
```

Comportamiento:

- Si falta una clave requerida:
  - Se genera un error envuelto con `domain.ErrCritical`.
  - Se aplica la política de errores (ver siguiente sección).
  - El servicio **no se ejecuta**.

Esto permite parametrizar qué datos de entrada son obligatorios para cada step sin cambiar código.

---

## 4. Ejecución y Caché (Run)

El método `Run` (`/api/v1/process-lifecycle/run`) implementa una estrategia inteligente de caché para equilibrar rendimiento y consistencia:

1.  **Modo Producción** (`override_process_version_id = null`):
    - Utiliza **Redis** (Cache-Aside) para resolver la versión vigente.
    - Esto maximiza el rendimiento en entornos productivos de alto tráfico.

2.  **Modo Override / Test** (`override_process_version_id > 0`):
    - **Ignora Redis** completamente.
    - Consulta siempre directamente a PostgreSQL (`resolve_process_version`).
    - Garantiza que al probar una versión específica (DRAFT o TEST), siempre se ejecute la configuración más reciente de la base de datos, sin interferencia de cachés antiguas.

## 5. Manejo de errores en la cadena de servicios

Archivo: `internal/services/serviceconfig/executor.go`

La lógica central está en `ExecuteServicesInOrder`:

```go
if execErr != nil {
    if errors.Is(execErr, domain.ErrCritical) {
        // Siempre corta
        log.Printf("🔴 Error crítico en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
        return execErr
    }
    if errors.Is(execErr, domain.ErrTolerable) {
        switch serviceConfig.ErrorTolerance {
        case "critical":
            log.Printf("🔴 Error tolerable tratado como crítico en '%s' por configuración. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
            return execErr
        case "tolerable", "inherit", "":
            log.Printf("⚠️ Error tolerable en '%s'. La ejecución continuará. Error: %v", serviceConfig.Path, execErr)
            continue
        default:
            log.Printf("🛑 Error tolerable con configuración desconocida en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
            return execErr
        }
    }
    switch serviceConfig.ErrorTolerance {
    case "tolerable":
        log.Printf("⚠️ Error tolerable por configuración en '%s'. La ejecución continuará. Error: %v", serviceConfig.Path, execErr)
        continue
    case "critical":
        log.Printf("🔴 Error crítico por configuración en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
        return execErr
    default:
        log.Printf("🛑 Error no clasificado en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
        return execErr
    }
}
```

Resumen:

- **Errores `ErrCritical`**:
  - Siempre detienen la cadena, independientemente de `error_tolerance`.

- **Errores `ErrTolerable`**:
  - Si `error_tolerance = "critical"` → se consideran críticos y cortan.
  - Si `error_tolerance = "tolerable"` o `"inherit"` (o vacío) → se loguean y la cadena continúa.

- **Errores no tipados (otros)**:
  - Se manejan según `error_tolerance`:
    - `"tolerable"` → continúa.
    - `"critical"` o default → corta.

El campo `mode` ha sido eliminado del motor; toda la política se expresa mediante:

- Tipo de error (`ErrCritical` / `ErrTolerable` / otros).
- `error_tolerance` por step.

---

## 6. Servicios Loan Risk y uso de config

Ejemplos de servicios que usan tanto `Input` como `CurrentStepConfig`.

### Age Service

Archivo: `internal/services/loanrisk/age.go`

```go
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

    // min_age viene del config del step (por defecto 18)
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
    return nil
}
```

### Salary Service

Archivo: `internal/services/loanrisk/salary.go`

```go
func (s *Salary) Execute() error {
    fmt.Println("💰 Ejecutando servicio Salary")

    rawSalary, _ := s.ctx.GetInputValue("salary")
    salary := 0
    switch v := rawSalary.(type) {
    case int:
        salary = v
    case int64:
        salary = int(v)
    case float64:
        salary = int(v)
    }

    // min_salary viene del config del step (por defecto 1)
    minSalary := 1
    if cfg := s.ctx.CurrentStepConfig; cfg != nil {
        if v, ok := cfg["min_salary"]; ok {
            switch n := v.(type) {
            case int:
                minSalary = n
            case int64:
                minSalary = int(n)
            case float64:
                minSalary = int(n)
            }
        }
    }

    if salary < minSalary {
        return fmt.Errorf(
            "%w: salario %d menor al mínimo permitido %d",
            domain.ErrCritical,
            salary,
            minSalary,
        )
    }

    data := map[string]any{
        "salary_checked":       true,
        "min_salary":           minSalary,
        "salary_bracket_k_usd": salary / 1000,
    }

    result := contracts.StepResult{
        Status: "ok",
        Input:  s.ctx.SnapshotInput(),
        Data:   data,
    }
    s.ctx.SetResult(s.servicePath, result)
    return nil
}
```

---

## 8. Modo Test y Métricas de Rendimiento

El motor incluye un modo especial de **Test / Diagnóstico** que se activa cuando se solicita explícitamente ejecutar una versión específica (override) en lugar de la versión productiva vigente.

### 8.1. Activación

Para activar este modo, se debe enviar `override_process_version_id > 0` en el request al endpoint `POST /api/v1/process-lifecycle/run`.

```json
{
  "process_type_id": 1,
  "sede_id": 1,
  "override_process_version_id": 5, // <--- Activa modo Test
  "input": { ... }
}
```

### 8.2. Comportamiento Diferenciado

| Característica | Modo Producción (Default) | Modo Test (Override > 0) |
| :--- | :--- | :--- |
| **Resolución de Versión** | **Redis (Cache)**. Maximiza velocidad. | **PostgreSQL (Directo)**. Garantiza consistencia inmediata. |
| **Métricas Globales** | Desactivadas (Cero overhead). | Activadas (`performance` object). |
| **Métricas por Step** | Desactivadas. | Activadas (`duration_us` en `details`). |
| **Respuesta API** | Limpia (solo negocio). | Enriquecida con tiempos y recursos. |

### 8.3. Estructura de Respuesta en Modo Test

Cuando el modo test está activo, la respuesta JSON se enriquece con métricas de rendimiento detalladas.

```json
{
  "status": "success",
  "data": {
    "process_version_id": 5,
    "input": { ... },

    // 1. Resultado Consolidado (Output final del proceso)
    "result": {
      "score": 85,
      "approved": true
    },

    // 2. Detalle paso a paso (Incluye tiempos individuales)
    "details": {
      "loanrisk/age": {
        "status": "ok",
        "data": {
          "is_adult": true,
          "duration_us": 45  // <--- Tiempo en Microsegundos (µs)
        }
      },
      "loanrisk/salary": {
        "status": "ok",
        "data": {
          "salary_checked": true,
          "duration_us": 120 // <--- 120 µs
        }
      }
    },

    // 3. Métricas Globales del Sistema
    "performance": {
      "execution_id": "uuid-v4...",
      "total_duration_ms": 5,       // Tiempo total (incluye overhead)
      "db_read_ms": 2,              // Tiempo acumulado en lecturas DB
      "db_write_ms": 1,             // Tiempo acumulado en escrituras DB
      "db_total_queries": 3,        // Cantidad de queries ejecutadas
      "memory_used_mb": 1.5,        // Memoria asignada (Heap Alloc)
      "goroutines": 12              // Gorutinas activas
    }
  }
}
```

### 8.4. Unidades de Medida

*   **Steps Individuales (`duration_us`)**: Se miden en **Microsegundos (µs)**.
    *   Razón: Los servicios de lógica de negocio en Go son extremadamente rápidos (a menudo < 1ms). Usar milisegundos daría `0` en la mayoría de los casos.
    *   Ejemplo: `45` significa 45 microsegundos (0.045 ms).

*   **Total Global (`total_duration_ms`)**: Se mide en **Milisegundos (ms)**.
    *   Razón: Representa la latencia total desde la perspectiva del motor, incluyendo overhead de orquestación, red y base de datos.

---

## 9. Comandos CLI relevantes

### 7.1. Seeder de Process Lifecycle

Seeder que crea el tipo de proceso “Loan risk lifecycle” y sus steps parametrizados:

- Comando (todos los seeders):

```bash
go run ./cmd/cmd-cli/main.go seed
```

- Solo lifecycle:

```bash
go run ./cmd/cmd-cli/main.go seed --only process_lifecycle_manager
```

### 7.2. Comando `run-loanrisk-lifecycle`

Archivo: `cmd/cmd-cli/cmd/loanrisk_lifecycle.go`

Ejecuta el proceso “Loan risk lifecycle” usando un payload JSON como `Input`:

```bash
go run ./cmd/cmd-cli/main.go run-loanrisk-lifecycle \
  --payload '{"age":60,"salary":4000000,"sede_id":1}'
```

Si no se pasa `--payload`, usa valores por defecto:

```json
{
  "age": 50,
  "salary": 100000,
  "sede_id": 1
}
``+

Salida (simplificada):

```json
{
  "process_version_id": 123,
  "input": {
    "age": 60,
    "salary": 4000000,
    "sede_id": 1
  },
  "results": {
    "loanrisk/NewAgeService": {
      "status": "ok",
      "input": { "...": "..." },
      "data": {
        "age_processed": "Edad validada: 60",
        "min_age": 40,
        "is_adult": true
      }
    },
    "loanrisk/NewSalaryService": {
      "status": "ok",
      "input": { "...": "..." },
      "data": {
        "salary_checked": true,
        "min_salary": 2500000,
        "salary_bracket_k_usd": 4000
      }
    }
  }
}
```

---

Con este diseño:

- El **input de negocio** fluye como un solo JSON (`ServiceContext.Input`).
- Cada servicio:
  - Valida las claves obligatorias vía `required_keys`.
  - Usa configuración dinámica desde `CurrentStepConfig`.
  - Devuelve un `StepResult` con `Data` de negocio y `Input` opcional para debug.
- El motor de ejecución:
  - Ordena y ejecuta los servicios secuencialmente.
  - Aplica políticas de error expresadas por:
    - Tipo de error (`ErrCritical` / `ErrTolerable`).
    - `error_tolerance` configurado por step.

