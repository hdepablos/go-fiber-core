# Motor de Process Lifecycle: visión general

Este documento resume cómo funciona el motor de **Process Lifecycle** y el **motor de ejecución de servicios**. Está pensado como guía rápida para entender:

- Qué es `execution_key` y cómo se resuelven los servicios.
- Cómo se construyen y usan `ServiceContext` y `StepResult`.
- Cómo funciona la validación de `required_keys`.
- Cómo se respeta el orden de ejecución (`step_order`).
- Cómo se integran BD, runtime y CLI/API.

---

## 1. Conceptos principales

### 1.1 `execution_key`

- Campo en la tabla `process_steps` que identifica de forma lógica un servicio.
- Ejemplos:
  - `loanrisk/NewAgeService`
  - `loanrisk/NewSalaryService`
  - `loanrisk/NewIsRenovationService`
  - `loanrisk/NewValidationService`
- Esta misma cadena se usa en tres lugares:
  - BD: `process_steps.execution_key`.
  - Registro de servicios en Go:
    - `serviceconfig.Register("loanrisk/NewAgeService", NewAgeService)`.
  - Resultados de ejecución:
    - `ServiceContext.Results["loanrisk/NewAgeService"]`.

Punto clave: es un **ID lógico estable de servicio**, no necesariamente el nombre literal de una función de Go, aunque por convención se parece (`paquete/NewXService`).

### 1.2 `process_steps` y `step_order`

Cada escenario de proceso tiene una versión (`process_versions`) y una lista de pasos (`process_steps`):

- `process_steps.process_version_id`: a qué versión pertenece el step.
- `process_steps.step_order`: orden en el que el step debe ejecutarse.
- `process_steps.roadmap`: segmento del roadmap al que pertenece el paso.
- `process_steps.execution_key`: ID lógico del servicio que se ejecutará.
- `process_steps.config` (JSONB): configuración específica del step.

El motor siempre filtra por `roadmap` y ordena los steps por `step_order` antes de ejecutar.

---

## 2. ServiceContext y StepResult

Archivo: `internal/services/serviceconfig/contracts/service.go`

### 2.1 `ServiceContext`

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
  - Bolsa global de datos de negocio del proceso.
  - Nace del JSON de entrada (CLI, API, etc.) y se va enriqueciendo.
  - Todos los servicios leen y escriben sobre este mapa.

- `Results`:
  - Mapa de resultados por servicio, indexado por `execution_key`.
  - Valor típico: `StepResult`.

- `CurrentStepConfig`:
  - Configuración específica del step actual, proveniente de `process_steps.config`.
  - El executor la llena antes de llamar a `Execute()`.

### 2.2 `StepResult`

```go
type StepResult struct {
	Status    string         `json:"status"`
	Message   string         `json:"message,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	StepOrder int            `json:"step_order,omitempty"`
}
```

- `Status`:
  - Estado del servicio (`"ok"`, `"error"`, etc. según convención).

- `Message`:
  - Mensaje opcional para contexto adicional.

- `Input`:
  - Fotografía opcional del `ServiceContext.Input` en el momento de ejecutar el servicio.
  - Es solo para observabilidad y debug.

- `Data`:
  - Resultado de negocio propio del servicio (ej. `calculated_risk`, `is_adult`, etc.).

- `StepOrder`:
  - Orden de ejecución del step (copiado desde `process_steps.step_order`).
  - Se rellena automáticamente por el executor, no por el servicio.

---

## 3. Registro y resolución de servicios

Archivo típico de servicio: `internal/services/loanrisk/age.go`

```go
type Age struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewAgeService() contracts.Service {
	return &Age{}
}

func (a *Age) Init(ctx *contracts.ServiceContext, servicePath string) {
	a.ctx = ctx
	a.servicePath = servicePath
}

func (a *Age) Execute() error {
	// Lógica de negocio, lectura/escritura de Input y armado de StepResult
}

func init() {
	serviceconfig.Register("loanrisk/NewAgeService", NewAgeService)
}
```

### 3.1 Registro

- Cada servicio se registra en el `init()` del archivo con:
  - `serviceconfig.Register("<execution_key>", <constructor>)`.
- Esto llena un mapa global:
  - `map[string]func() contracts.Service`.

### 3.2 Resolución

- El executor, al leer un `execution_key` desde `process_steps`, lo busca en este mapa.
- Si lo encuentra, obtiene la fábrica (`factory`) y crea la instancia del servicio con `factory()`.

---

## 4. Ejecución en orden y required_keys

Archivo: `internal/services/serviceconfig/executor.go`

### 4.1 ServiceRegistryRow

```go
type ServiceRegistryRow struct {
	Path           string
	Order          int
	ErrorTolerance string
	Config         []byte
	RequiredKeys   []string
}
```

- `Path`:
  - Es el `execution_key` (`loanrisk/NewAgeService`).

- `Order`:
  - `step_order` del step.

- `ErrorTolerance`:
  - Política de manejo de errores del step (`critical`, `tolerable`, `inherit`).

- `Config`:
  - Copia de `process_steps.config` como JSON.

- `RequiredKeys`:
  - Lista de claves que deben existir en `ServiceContext.Input` antes de ejecutar el servicio.

### 4.2 Ejecución en orden

```go
func ExecuteServicesInOrder(ctx context.Context, services []ServiceRegistryRow, svcCtx *contracts.ServiceContext) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if svcCtx != nil {
		svcCtx.Ctx = ctx
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Order < services[j].Order
	})

	for _, serviceConfig := range services {
		// Respeto de cancelación de contexto
		// Resolución de la fábrica por Path (execution_key)
		// Carga de CurrentStepConfig en svcCtx
		// Validación de required_keys (ver siguiente sección)
		// Ejecución del servicio
		// Manejo de errores según ErrorTolerance y ErrCritical / ErrTolerable
		// Relleno de StepOrder en el StepResult correspondiente
	}

	return nil
}
```

### 4.3 Validación de `required_keys`

Antes de llamar a `Execute()`, el executor valida las claves requeridas:

```go
if len(serviceConfig.RequiredKeys) > 0 && svcCtx != nil {
	for _, key := range serviceConfig.RequiredKeys {
		if _, ok := svcCtx.GetInputValue(key); !ok {
			execErr = fmt.Errorf("missing required key '%s' for service '%s': %w", key, serviceConfig.Path, domain.ErrCritical)
			break
		}
	}
}
```

Comportamiento:

- Si falta alguna clave requerida:
  - Se genera un error envuelto en `domain.ErrCritical`.
  - El executor trata esto como error crítico y detiene la cadena.

### 4.4 Manejo de errores

Se usan dos tipos de errores de negocio especiales:

- `domain.ErrCritical`:
  - Corta siempre la cadena (o según configuración).

- `domain.ErrTolerable`:
  - Puede permitir continuar la cadena según `error_tolerance` del step.

Combinado con `ErrorTolerance`:

- `"critical"`:
  - Cualquier error se trata como crítico y detiene la cadena.

- `"tolerable"`:
  - Errores no críticos permiten continuar.

- `"inherit"` o vacío:
  - Se usa la política base de `ErrCritical` vs `ErrTolerable`.

---

## 5. Construcción del registro desde la BD

Archivo: `internal/services/processlifecycle/process_lifecycle_service.go`

### 5.1 Estructura Step

```go
type Step struct {
	Name         string          `json:"name"`
	ExecutionKey string          `json:"execution_key"`
	Config       json.RawMessage `json:"config"`
	StepOrder    int32           `json:"step_order"`
}
```

### 5.2 `BuildServiceRegistryFromSteps`

```go
func BuildServiceRegistryFromSteps(steps []Step) ([]serviceconfig.ServiceRegistryRow, error) {
	rows := make([]serviceconfig.ServiceRegistryRow, 0, len(steps))

	for _, step := range steps {
		errorTolerance := "inherit"

		if len(step.Config) > 0 {
			var cfg struct {
				ErrorTolerance string `json:"error_tolerance"`
				// Aquí también podrían definirse required_keys, etc.
			}
			if err := json.Unmarshal(step.Config, &cfg); err != nil {
				return nil, domain.ErrInternal
			}

			if cfg.ErrorTolerance != "" {
				e := strings.ToLower(cfg.ErrorTolerance)
				switch e {
				case "critical", "tolerable", "inherit":
					errorTolerance = e
				default:
					errorTolerance = "inherit"
				}
			}
		}

		row := serviceconfig.ServiceRegistryRow{
			Path:           step.ExecutionKey,
			Order:          int(step.StepOrder),
			ErrorTolerance: errorTolerance,
			Config:         step.Config,
		}
		rows = append(rows, row)
	}

	return rows, nil
}
```

Aquí se traduce la definición de la BD (`Step`) al formato que entiende el executor (`ServiceRegistryRow`).

---

## 6. Orquestación completa: RunResolvedProcess

Archivo: `internal/services/processlifecycle/process_lifecycle_service.go`

```go
func (s *service) RunResolvedProcess(ctx context.Context, processTypeID int64, input map[string]any, overrideProcessVersionID *int64, roadmap int) (int64, *contracts.ServiceContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var sedeID int64 = 1
	// Sede se puede obtener desde input["sede_id"]

	processVersionID, steps, err := s.ResolveProcessVersion(ctx, processTypeID, sedeID, overrideProcessVersionID, roadmap)
	if err != nil {
		return 0, nil, err
	}

	registryRows, err := BuildServiceRegistryFromSteps(steps)
	if err != nil {
		return 0, nil, err
	}

	serviceCtx := contracts.NewServiceContextFromInput(ctx, input)

	if err := serviceconfig.ExecuteServicesInOrder(ctx, registryRows, serviceCtx); err != nil {
		return processVersionID, serviceCtx, err
	}

	return processVersionID, serviceCtx, nil
}
```

Resumen del flujo:

1. Resolver la versión de proceso vigente y los pasos asociados al roadmap (`ResolveProcessVersion`).
2. Convertir los steps de BD a `ServiceRegistryRow` (`BuildServiceRegistryFromSteps`).
3. Construir `ServiceContext` a partir del `input`.
4. Ejecutar servicios en orden (`ExecuteServicesInOrder`).
5. Devolver:
   - `processVersionID`.
   - `ServiceContext` con:
     - `Input` final (bolsa de datos de negocio).
     - `Results` con resultados por servicio.

Este es el punto de entrada que usan tanto CLI como posibles endpoints API para ejecutar un proceso completo basado en configuración.

