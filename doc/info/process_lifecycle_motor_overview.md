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
  - `loanrisk/age`
  - `loanrisk/salary`
  - `loanrisk/is_renovation`
  - `loanrisk/validation`
- Esta misma cadena se usa en tres lugares:
  - BD: `process_steps.execution_key`.
  - Registro de servicios en Go:
    - `serviceconfig.Register("loanrisk/age", NewAgeService)`.
  - Resultados de ejecución:
    - `ServiceContext.Results["loanrisk/age"]`.

Punto clave: es un **ID lógico estable de servicio**, no necesariamente el nombre literal de una función de Go, aunque por convención se parece (`paquete/nombre`).

### 1.2 `process_steps` y `step_order`

Cada escenario de proceso tiene una versión (`process_versions`) y una lista de pasos (`process_steps`):

- `process_steps.process_version_id`: a qué versión pertenece el step.
- `process_steps.step_order`: orden de ejecución. Los pasos con el mismo número de orden se ejecutan en **paralelo**.
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

## 4. Ejecución Paralela, Timeouts y Orden

Archivo: `internal/services/serviceconfig/executor.go`

### 4.1 ServiceRegistryRow

```go
type ServiceRegistryRow struct {
	Path           string
	Order          int
	ErrorTolerance string
	Config         []byte
	RequiredKeys   []string
	Timeout        time.Duration
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

- `Timeout`:
  - Tiempo máximo de ejecución permitido para este servicio (derivado de `timeout_ms` en JSON).

### 4.2 Ejecución Paralela por Orden

La función `ExecuteServicesInOrder` agrupa los servicios por su `Order` y los ejecuta en paralelo dentro de cada grupo.

```go
func ExecuteServicesInOrder(ctx context.Context, services []ServiceRegistryRow, svcCtx *contracts.ServiceContext) error {
	// ... Ordenar services por Order ...

	// Agrupar por Order y ejecutar cada grupo secuencialmente
	for _, order := range orders {
		groupRows := grouped[order]

		// Ejecución paralela usando errgroup
		g, groupCtx := errgroup.WithContext(ctx)

		for _, serviceConfig := range groupRows {
			// Lanzar goroutine para cada servicio del grupo
			g.Go(func() error {
				return executeOneService(groupCtx, sc, svcCtx)
			})
		}

		// Esperar a que todo el grupo termine
		if err := g.Wait(); err != nil {
			return err // Si hay error crítico, se detiene la cadena completa
		}
	}
	return nil
}
```

**Comportamiento:**
1. Los pasos con el mismo `Order` (ej. 1) inician simultáneamente.
2. El sistema espera a que **todos** los pasos del grupo terminen (éxito o error tolerable) antes de avanzar al siguiente `Order` (ej. 2).
3. Si un paso falla con un error crítico, se cancela el contexto del grupo y se retorna el error, deteniendo la cadena.

### 4.3 Timeouts por Step

Cada step puede definir un `timeout_ms` en su configuración JSON.

- Si se define (ej. `timeout_ms: 1000`):
  - Se crea un `context.WithTimeout` específico para ese servicio.
  - Si el servicio tarda más de lo permitido, el contexto se cancela con `context.DeadlineExceeded`.
  - El error se reporta como "timeout exceeded".

### 4.4 Validación de `required_keys`

Antes de llamar a `Execute()`, el executor valida las claves requeridas:

```go
if len(serviceConfig.RequiredKeys) > 0 && svcCtx != nil {
	for _, key := range serviceConfig.RequiredKeys {
		if _, ok := svcCtx.GetInputValue(key); !ok {
			// Error crítico por falta de claves
			return fmt.Errorf("missing required key...")
		}
	}
}
```

### 4.5 Manejo de errores y `error_tolerance`

La tolerancia a fallos se evalúa tras la ejecución (o timeout) de cada servicio:

- **`"critical"`** (default):
  - Si el servicio falla, el error se propaga, el grupo paralelo se cancela y la ejecución global se detiene.

- **`"tolerable"`**:
  - Si el servicio falla (incluso por timeout), el error se loguea como advertencia (`⚠️`).
  - El servicio retorna `nil` al grupo de ejecución, permitiendo que los otros servicios paralelos continúen y que la cadena avance.

- **`domain.ErrTolerable`**:
  - Si el servicio retorna este error de dominio explícito, se trata como tolerable a menos que la configuración fuerce `critical`.

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
				ErrorTolerance string   `json:"error_tolerance"`
				RequiredKeys   []string `json:"required_keys"`
				TimeoutMs      int64    `json:"timeout_ms"`
			}
			if err := json.Unmarshal(step.Config, &cfg); err != nil {
				return nil, domain.ErrInternal
			}

			// ... validación ErrorTolerance ...

			if cfg.TimeoutMs > 0 {
				timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
			}
		}

		row := serviceconfig.ServiceRegistryRow{
			Path:           step.ExecutionKey,
			Order:          int(step.StepOrder),
			ErrorTolerance: errorTolerance,
			Config:         step.Config,
			RequiredKeys:   cfg.RequiredKeys,
			Timeout:        timeout,
		}
		rows = append(rows, row)
	}

	return rows, nil
}
```

Aquí se traduce la definición de la BD (`Step`) al formato que entiende el executor (`ServiceRegistryRow`), incluyendo la extracción de `timeout_ms` y `required_keys`.

---

## 6. Orquestación completa: Run

Archivo: `internal/services/processlifecycle/process_lifecycle_service.go`

```go
func (s *service) Run(ctx context.Context, req requests.RunProcessRequest) (int64, *contracts.ServiceContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 1. Validar y normalizar Input (sede_id, roadmap, operator_id)
	if req.SedeID <= 0 { req.SedeID = 1 }
	if req.Input == nil { req.Input = make(map[string]any) }
	
	req.Input["sede_id"] = req.SedeID
	req.Input["roadmap"] = req.Roadmap
	if req.OperatorID > 0 { req.Input["operator_id"] = req.OperatorID }

	// 2. Resolver versión y pasos
	processVersionID, steps, err := s.ResolveProcessVersion(ctx, req.ProcessTypeID, req.SedeID, req.OverrideProcessVersionID, req.Roadmap)
	if err != nil {
		return 0, nil, err
	}

	// 3. Convertir a registro de servicios
	registryRows, err := BuildServiceRegistryFromSteps(steps)
	if err != nil {
		return 0, nil, err
	}

	// 4. Crear contexto de ejecución
	serviceCtx := contracts.NewServiceContextFromInput(ctx, req.Input)

	// 5. Ejecutar servicios
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
     - `GetAll()`: Mapa aplanado de resultados para consumo sencillo.

Este es el punto de entrada estandarizado (`RunProcessRequest`) para ejecutar un proceso completo.

---

## 7. Resumen de Capacidades (Features)

Lista rápida de características contempladas en el motor:

- **Orquestación Dinámica**: Flujos definidos en base de datos (`process_steps`), permitiendo cambios sin desplegar código.
- **Ejecución Paralela**: Los pasos con el mismo `step_order` se ejecutan simultáneamente para mejorar el rendimiento.
- **Timeouts por Step**: Cada paso tiene un límite de tiempo configurable (`timeout_ms`) que cancela automáticamente procesos lentos.
- **Tolerancia a Fallos**: Configuración granular de errores (`error_tolerance: "critical" | "tolerable"`) para decidir si detener o continuar el flujo.
- **Contexto Thread-Safe**: El `ServiceContext` protege los datos compartidos contra condiciones de carrera en ejecuciones paralelas.
- **Validación de Inputs**: El sistema verifica automáticamente que existan las `required_keys` antes de ejecutar un servicio.
- **Herencia de Configuración por Sede**: Si una sede no tiene versión personalizada, hereda automáticamente la configuración global (`sede_id=NULL`).
- **Versionado Completo**: Soporte para `process_versions` y overrides para pruebas.
- **Segmentación**: Filtrado de ejecución por `roadmap` (ej. segmentos de negocio) y `sede_id`.
- **Inyección de Configuración**: Cada paso puede recibir parámetros JSON dinámicos (`config`) inyectados en tiempo de ejecución.

---

## 8. Catálogo de Errores de Dominio

Lista de errores estándar definidos en `internal/domain/errors.go` que el motor reconoce y gestiona:

- **`ErrCritical`**: Error irrecuperable. Detiene inmediatamente la cadena de ejecución (a menos que `error_tolerance: "tolerable"` lo anule).
- **`ErrTolerable`**: Error leve. Permite que la ejecución continúe, registrando solo una advertencia (a menos que `error_tolerance: "critical"` lo fuerce).
- **`ErrMissingRequiredKey`**: Lanzado automáticamente por el motor cuando falta una clave definida en `required_keys`.
- **`ErrBusinessRuleViolation`**: Error genérico para reglas de negocio no cumplidas (ej. "Edad insuficiente").
- **`ErrSedeNotFound` / `ErrRoadmapNotFound`**: Errores de validación inicial al resolver la versión del proceso.
- **`ErrOverrideVersionNotFound`**: Error específico cuando se solicita una versión de prueba que no existe.

---

## 9. Endpoint General de Ejecución

Existe un endpoint unificado para ejecutar cualquier tipo de proceso (`process_type_id`), ya sea en modo producción o prueba (usando overrides).

**Ruta**: `POST /api/v1/process-lifecycle/run`

**Payload JSON (`RunProcessRequest`):**

```json
{
  "process_type_id": 1,             // ID del tipo de proceso a ejecutar
  "sede_id": 1,                     // ID de la sede (0 o null para global)
  "roadmap": 0,                     // Segmento del roadmap (0=default)
  "override_process_version_id": 0, // ID de versión específica para TEST (0 o null para usar PROD vigente)
  "input": {                        // Datos de entrada para el proceso
    "user_id": 123,
    "amount": 5000
  }
}
```

**Comportamiento:**
- **Modo Producción**: Enviar `override_process_version_id: 0` (o null). El sistema resolverá automáticamente la versión vigente PROD para esa sede.
- **Modo Test**: Enviar `override_process_version_id: <ID>`. El sistema forzará la ejecución de esa versión específica (útil para probar borradores).



