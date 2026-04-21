---
domain: agents
summary: Índice de convenciones técnicas del proyecto. Referencia rápida de patrones obligatorios para servicios, batch, exports, integraciones HTTP, logger, Makefile, endpoints y base de datos.
when_to_read:
  - al crear un servicio, proceso batch, export o adapter HTTP nuevo
  - cuando hay duda sobre el patrón correcto en algún dominio técnico
  - al hacer onboarding de nuevos colaboradores o agentes
code_paths:
  - internal/services/
  - internal/services/externalhttp/
  - internal/services/batchflow/
  - internal/appconfig/config.yml
  - cmd/tools/
  - Makefile
related_info:
  - doc/info/development/service-design-conventions.md
  - doc/info/development/external-http-service-standard.md
  - doc/info/development/service-runtime-and-scaffold.md
  - doc/info/development/process-scaffold-and-cleanup.md
  - doc/info/platform/makefile-guide.md
related_specs:
  - doc/specs/architecture/service-design-spec.md
  - doc/specs/architecture/external-http-client-spec.md
  - doc/specs/platform/logger-runtime-spec.md
  - doc/specs/platform/makefile-automation-spec.md
  - doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md
  - doc/specs/agents/change-matrix.md
status: active
---

# conventions-index.md

Índice de convenciones técnicas del proyecto. Este archivo es la referencia rápida; las specs de cada dominio son la fuente normativa completa.

---

## Servicios

### Patrón obligatorio

Todo servicio nuevo o refactorizado debe usar:

1. **Interface segregation**: interfaces pequeñas, enfocadas en el contrato.
2. **Constructor injection**: dependencias explícitas en `New...`.
3. **Method receivers**: implementación vía receivers sobre struct concreta.
4. **Implementación no exportada**: la struct concreta no debe exportarse salvo razón fuerte y documentada.

```go
// Definir interface pequeña y enfocada
type PaymentService interface {
    Process(ctx context.Context, amount float64) error
    Refund(ctx context.Context, id string) error
}

// Struct concreta no exportada
type paymentService struct {
    db     *sql.DB
    logger *slog.Logger
}

// Constructor con dependencias explícitas
func NewPaymentService(db *sql.DB, logger *slog.Logger) PaymentService {
    return &paymentService{db: db, logger: logger}
}

// Implementación via method receivers
func (s *paymentService) Process(ctx context.Context, amount float64) error { return nil }
func (s *paymentService) Refund(ctx context.Context, id string) error       { return nil }
```

**Spec normativa**: `doc/specs/architecture/service-design-spec.md`

---

## Batch (`batchflow`)

### Modelo de versionado

- Un único `process_type` por dominio de negocio.
- Las diferencias `sequential` o `fanout` viven en `process_versions` y su configuración, **no** en nombres distintos de `process_type`.
- La versión base es secuencial. La versión companion fanout reutiliza las mismas execution keys del negocio.

### Registro de managers (obligatorio)

Todo proceso batch nuevo sobre `batchflow` que participe en cancelación operativa o auto-cancel con `lifecycle.Fail(...)` **debe** registrar su `Manager` en el registry central resuelto por `execution_key`.

- No dejar esa resolución hardcodeada en consumers o handlers por proceso nuevo.
- Si el scaffold no genera esa estructura por defecto, corregir el scaffold antes de crear procesos nuevos.

### Comandos de scaffold

```bash
# Crear proceso batch nuevo
make create-batch-process ...

# Clonar process_version existente
make clone-process-version source_version_id=... operator_id=... [with_pacing=true pacing_messages=... pacing_interval=...]

# Agregar pacing a versión ya existente
make add-process-pacing source_version_id=... operator_id=... pacing_messages=... pacing_interval=...

# Limpiar proceso scaffold
make delete-process kind=batch-process service_slug=...

# Ver catálogo completo
make list-scaffolds
```

**Notas**:
- El scaffold batch no debe crear carpetas Bruno específicas por proceso. Reutilizar `bruno/legacy/process-lifecycle/test-batch-process`.
- `clone-process-version` y `add-process-pacing` se presentan como operaciones hijas del dominio `batch-process` en `make list-scaffolds`.

**Spec normativa**: `doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md` · `doc/specs/process-lifecycle/batch-fanout-spec.md`

---

## Export batch (`exportmanager`)

### Registro de managers (obligatorio)

Todo export batch nuevo sobre `exportmanager` que participe en cancelación operativa o auto-cancel con `ParentLifecycle.Fail(...)` **debe** registrar su `Manager` en el registry central de exports resuelto por `execution_key`.

- No dejar esa resolución hardcodeada en consumers o handlers por export nuevo.
- Si un export legacy no usa `exportmanager.Manager`, documentarlo explícitamente como excepción hasta migrarlo.
- Si el scaffold no genera esa estructura por defecto, corregir el scaffold antes de crear exports nuevos.

**Spec normativa**: `doc/specs/exports/export-pipelines-spec.md`

---

## Integraciones HTTP externas

### Fuente de configuración

- Configuración de APIs externas desde `internal/appconfig/config.yml`, sección `apis.xxx`.
- Resolución canónica: `appConfig.APIConfig("xxx")`.
- Los adapters deben recibir `config.ApiConfig`. **No** hardcodear `url`, `token` ni leer variables de entorno directamente.

### Cliente HTTP compartido

- Toda llamada HTTP externa reutilizable se centraliza en `internal/services/externalhttp/`.
- Los adapters delegan en `externalhttp.NewClientFromAPIConfig(...)`.
- **No usar** `resty.New()` directamente dentro de adapters nuevos.
- Si una integración necesita salirse del patrón, dejar justificación explícita en código y documentación.
- Si el servicio externo usa otro esquema de autenticación (distinto de bearer token), extender el servicio común en vez de romper el patrón en el adapter.

### Scaffold de adapters externos

```bash
make create-external-adapter adapter_name=customer_api config_key=customer_api
```

**Spec normativa**: `doc/specs/architecture/external-http-client-spec.md`

---

## Logger

### Por entorno

| Entorno | Convención |
|---------|-----------|
| Producción (Lambda / EKS) | `stdout` — AWS captura via CloudWatch. No usar archivos locales como destino principal. |
| Local (desarrollo) | `logger.GetLogger("nombre_proceso")` — separar por proceso específico. No usar logger genérico único. |
| Debug puntual | `logger.GetLoggerToFile("nombre_proceso", "...")` — archivo local específico por proceso, nunca un archivo global. |

### Tipos de log obligatorios

| `log_type` | Cuándo emitir |
|------------|---------------|
| `redis_guard` | Errores relevantes de Redis en el core batch. No deben pasar silenciosamente. |
| `rate_limit_guard` | Rate limit interno del core y `429` de dependencias externas. No deben pasar silenciosamente. |
| `execution_guard` | Cancelaciones manuales, auto-cancel por threshold, pausas o cortes operativos relevantes. |

### Reglas de emisión

- El logging transversal vive en el core compartido correspondiente, **no** en cada adapter o flujo.
- Si una solicitud implica capacidad, stress, Redis o fanout, revisar y mantener alineados los documentos operativos y normativos de observabilidad batch.

**Spec normativa**: `doc/specs/platform/logger-runtime-spec.md`

---

## Makefile y herramientas

### Contexto de ejecución

Todo comando nuevo del Makefile o de `cmd/tools/` que abra conexiones a Postgres, Redis, colas u otra infraestructura debe:

- Definirse con un contexto de ejecución explícito.
- Ejecutarse dentro del contexto Docker Compose, preferentemente via `$(DC_RUN)` o wrapper equivalente.
- No asumir que `go run ./cmd/tools/...` desde host funciona igual que dentro del contenedor.

Si un comando debe soportar host además de Docker, la documentación y el contrato operativo deben explicitar qué configuración local lo hace válido.

### Catálogos operativos

| Comando | Propósito |
|---------|-----------|
| `make list-scaffolds` | Generadores y scaffolds reutilizables con opciones, variantes y cleanup |
| `make list-tools` | Utilidades operativas frecuentes agrupadas por dominio |

Al agregar scaffold o utilidad operativa nueva, evaluar y actualizar ambos catálogos en la misma solicitud. Cuando cambien, revisar también:
- `doc/info/platform/makefile-guide.md`
- `doc/specs/platform/makefile-automation-spec.md`

**Spec normativa**: `doc/specs/platform/makefile-automation-spec.md` · `doc/specs/platform/process-scaffold-cleanup-spec.md`

---

## Compatibilidad multi-entorno

Todo cambio de infraestructura, runtime, bootstrap, colas, endpoints AWS, variables de entorno, Terraform, Helm/K8s, Docker Compose o LocalStack debe evaluarse pensando en ambos entornos objetivo: **`lambda`** y **`eks`**.

- No optimizar una corrección solo para el entorno que está fallando si eso rompe o deja ambiguo el otro.
- Si una decisión operativa cambia URLs, nombres de colas, wiring, variables de entorno o comportamiento de bootstrap, verificar explícitamente cómo queda en `lambda` y en `eks`.
- Si un cambio debe comportarse distinto entre `lambda` y `eks`, esa diferencia debe quedar **justificada y documentada** — no dejarla implícita.

**Spec normativa**: `doc/specs/architecture/service-runtime-bootstrap-spec.md` · `doc/specs/platform/platform-runtime-spec.md`

---

## Endpoints y Bruno

### Convenciones generales

- Todo endpoint nuevo → documentación humana en `doc/info/` con ejemplo de request si usa body.
- Si el endpoint define un contrato reutilizable o relevante para automatización → también en `doc/specs/`.
- Todo endpoint nuevo o modificado debe tener request Bruno canónico cuando forme parte del API operable.

### Organización en Bruno

| Tipo | Ubicación |
|------|-----------|
| Colección canónica | `bruno/api/` |
| Endpoints `/api/v1/...` | `bruno/api/v1/...` siguiendo la URL real |
| Endpoints fuera de `/api/v1` | Agrupar por path real |
| Históricos, exploratorios, variantes de prueba | `bruno/legacy/` — no mezclar con la colección principal |

### Auth y headers

- Endpoints protegidos: `auth: bearer` + `{{access_token}}`.
- Login: actualizar `access_token` y, cuando aplique, `refresh_token`.
- Header operativo por defecto: `X-Client-Code: bruno`.

### Request bodies

- Todo endpoint `POST .../paginated` → estructura base compatible con `PaginationRequest`.
- Endpoints multipart → dejar claro `content-type`, nombre del campo archivo y variables necesarias.
- Ejemplos de body en documentación y Bruno alineados con DTOs reales del código.

> **Nota**: En endpoints multipart, preferir metadata del comando en el body y no en parámetros de ruta cuando represente datos de negocio del request.

**Spec normativa**: `doc/specs/api/http-endpoints-spec.md`

---

## Base de datos

### Regla de actualización obligatoria

Cada vez que se agregue o modifique una migración SQL que impacte tablas, columnas, relaciones, índices, enums, constraints o reglas de integridad:

1. Actualizar `doc/specs/data/database-schema-query-spec.md`.
2. Si el cambio también altera el entendimiento humano del modelo, actualizar `doc/info/data/`.

El objetivo es que futuras solicitudes de reportes, consultas SQL, joins o sentencias de mantenimiento puedan resolverse desde la documentación base sin relevar todo desde cero.

> Si una migración cambia el modelo y no se actualizan las specs de `data`, el trabajo debe considerarse **incompleto**.

**Spec normativa**: `doc/specs/data/database-schema-query-spec.md`

---

## Shared / utilidades reutilizables

### Regla de documentación

Cada vez que se cree o modifique una función compartida relevante para uso transversal:

1. Revisar y actualizar `doc/specs/shared/shared-utils-spec.md`.
2. Si el cambio introduce una utilidad nueva, documentar su contrato, límites y casos de uso.
3. Si una función compartida cambia de semántica y la spec no se actualiza, el trabajo debe considerarse **incompleto**.

**Spec normativa**: `doc/specs/shared/shared-utils-spec.md`
