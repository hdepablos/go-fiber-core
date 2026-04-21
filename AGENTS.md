# AGENTS.md

<!-- version: 2.0 | updated: 2025 | maintainer: SDD -->

Guía normativa para agentes IA y desarrolladores. Este archivo es el **portal de entrada**: define reglas de alto nivel y delega detalle a specs especializadas.

> **Regla de uso**: Antes de implementar o modificar código, identificar el dominio afectado en la [Matriz de cambios](#matriz-de-cambios), navegar a la spec correspondiente y aplicar sus contratos.

---

## Índice

- [Documentación por defecto](#documentación-por-defecto)
- [Reglas de separación y ubicación documental](#reglas-de-separación-y-ubicación-documental)
- [Índice operativo de specs](#índice-operativo-de-specs)
- [Matriz de cambios](#matriz-de-cambios)
- [Convenciones técnicas (resumen)](#convenciones-técnicas-resumen)
- [Regla de cierre](#regla-de-cierre)
- [Referencias](#referencias)

---

## Documentación por defecto

Cuando una solicitud implique documentar, redocumentar, reorganizar o ampliar documentación del proyecto:

- Crear o actualizar documentación para humanos en `doc/info/`.
- Crear o actualizar documentación normativa para IA y SDD en `doc/specs/`.
- Mantener `README.md` como portal principal y actualizar sus enlaces cuando cambie el mapa documental.
- Actualizar `doc/info/README.md` y `doc/specs/README.md` cuando se agreguen documentos nuevos o cambie la clasificación.

### Metadata mínima obligatoria para specs nuevas

Toda spec nueva en `doc/specs/` debe incluir al inicio:

```md
---
domain: <nombre-del-dominio>
summary: <contrato o comportamiento que norma, en 1-2 líneas>
when_to_read:
  - <disparador funcional concreto>
code_paths:
  - <path/al/directorio/o/archivo>
related_info:
  - doc/info/<dominio>/<guía>.md
related_specs:
  - doc/specs/<dominio>/<otra-spec>.md
status: active | draft | deprecated
---
```

**Campos obligatorios**: `domain`, `summary`, `when_to_read`, `code_paths`, `related_info`.  
**Campos recomendados**: `related_specs`, `status`.

> Si una spec nueva no incluye esta metadata, la documentación debe considerarse **incompleta**.

---

## Reglas de separación y ubicación documental

| Regla | Descripción |
|-------|-------------|
| **Separación** | `doc/info/` explica contexto, uso y troubleshooting. `doc/specs/` define contratos, invariantes y acceptance criteria. No duplicar contenido completo entre ambos. |
| **Ubicación** | No crear `.md` documentales fuera de `doc/info/`, `doc/specs/` o `README.md` raíz. Excepción: templates Markdown funcionales o READMEs técnicos muy locales. |
| **Trazabilidad** | Toda nueva documentación debe enlazar documentos relacionados, evitar duplicación y dejar claro si pertenece a humanos (`info`) o IA/SDD (`specs`). |
| **AGENTS.md sync** | Cada vez que se cree, mueva, renombre o amplíe documentación en `doc/specs/`, también actualizar este archivo: sección [Índice operativo de specs](#índice-operativo-de-specs) y [Referencias](#referencias). Si la spec agrega un dominio nuevo, reflejarlo en la misma solicitud. |

---

## Índice operativo de specs

> Revisar `doc/specs/README.md` antes de diseñar, refactorizar o documentar cambios importantes. Luego navegar a la spec del dominio afectado.

### Gobierno documental

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/documentation-governance-spec.md`](doc/specs/documentation-governance-spec.md) | Cambios en estructura documental, clasificación o trazabilidad |
| [`doc/specs/documentation-defaults-spec.md`](doc/specs/documentation-defaults-spec.md) | Nuevas convenciones de documentación o defaults del proyecto |

### Shared

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/shared/shared-utils-spec.md`](doc/specs/shared/shared-utils-spec.md) | Creación o modificación de helpers y utilidades reutilizables |

### Arquitectura

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/architecture/core-architecture-spec.md`](doc/specs/architecture/core-architecture-spec.md) | Cambios estructurales, refactors de diseño, casos de uso |
| [`doc/specs/architecture/service-design-spec.md`](doc/specs/architecture/service-design-spec.md) | Interfaces, constructores, convenciones de servicios |
| [`doc/specs/architecture/service-runtime-bootstrap-spec.md`](doc/specs/architecture/service-runtime-bootstrap-spec.md) | Wiring, bootstrap, registro de dependencias |
| [`doc/specs/architecture/external-http-client-spec.md`](doc/specs/architecture/external-http-client-spec.md) | Integraciones HTTP externas, adapters |

### API

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/api/http-endpoints-spec.md`](doc/specs/api/http-endpoints-spec.md) | Endpoints HTTP, DTOs, handlers, auth, Bruno |

> **Nota API**: En endpoints multipart, preferir metadata del comando en el body y no en parámetros de ruta cuando represente datos de negocio del request.

### Platform

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/platform/platform-runtime-spec.md`](doc/specs/platform/platform-runtime-spec.md) | Infraestructura, variables de entorno, runtime |
| [`doc/specs/platform/logger-runtime-spec.md`](doc/specs/platform/logger-runtime-spec.md) | Logger, niveles de log, tipos obligatorios |
| [`doc/specs/platform/makefile-automation-spec.md`](doc/specs/platform/makefile-automation-spec.md) | Makefile, targets, automatizaciones operativas |
| [`doc/specs/platform/process-scaffold-cleanup-spec.md`](doc/specs/platform/process-scaffold-cleanup-spec.md) | Scaffolds, cleanup de procesos, convenciones Bruno genéricas |

### Process Lifecycle

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md`](doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md) | Manager, runtime del motor, resolución de versiones, execution keys, cancelación |
| [`doc/specs/process-lifecycle/batch-preview-spec.md`](doc/specs/process-lifecycle/batch-preview-spec.md) | Preview batch, `apply_changes`, selección por `item_ids` o `row_numbers` |
| [`doc/specs/process-lifecycle/batch-fanout-spec.md`](doc/specs/process-lifecycle/batch-fanout-spec.md) | Fanout, shards, capacidad, Redis batch, `auto_invoke` |
| [`doc/specs/process-lifecycle/batch-observability-spec.md`](doc/specs/process-lifecycle/batch-observability-spec.md) | Observabilidad estructurada, rate limit, cancelación distribuida |

> **Nota Process Lifecycle**: Para hooks opcionales de refresco por lote o semántica custom de pendientes derivada desde `bulk_job_items`, revisar `process-lifecycle-runtime-spec.md` y `batch-preview-spec.md` cuando también impacte `apply_changes`.

### Exports

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/exports/export-pipelines-spec.md`](doc/specs/exports/export-pipelines-spec.md) | Export batch, `exportmanager`, generación de archivos, auto-cancel, registro por `execution_key` |

### Data

| Spec | Cuándo leerla |
|------|---------------|
| [`doc/specs/data/database-schema-query-spec.md`](doc/specs/data/database-schema-query-spec.md) | Modelos GORM, migraciones, relaciones, queries, integridad |

---

## Matriz de cambios

> Versión completa con ejemplos y notas: [`doc/specs/agents/change-matrix.md`](doc/specs/agents/change-matrix.md)

| Tipo de cambio | Specs obligatorias |
|----------------|-------------------|
| Documentación o mapa documental | `documentation-governance-spec` · `documentation-defaults-spec` |
| Servicios, casos de uso, interfaces, refactors estructurales | `service-design-spec` · `core-architecture-spec` |
| Wiring, bootstrap, runtime, registro de dependencias | `service-runtime-bootstrap-spec` · `platform-runtime-spec` |
| Infraestructura, Terraform, Helm/K8s, Docker Compose, LocalStack, env vars | `service-runtime-bootstrap-spec` · `platform-runtime-spec` · `makefile-automation-spec`* |
| Integraciones HTTP externas, adapters, `externalhttp/` | `external-http-client-spec` · `logger-runtime-spec` · `batch-observability-spec`* |
| Endpoints HTTP, DTOs, handlers, auth, Bruno | `http-endpoints-spec` |
| Makefile, automatizaciones, `list-scaffolds`, `list-tools` | `makefile-automation-spec` · `process-scaffold-cleanup-spec`* |
| Comandos Go bajo `cmd/tools/`, utilidades DB/Redis/colas | `makefile-automation-spec` · `service-runtime-bootstrap-spec`* |
| Scaffolds, cleanup de procesos, convenciones Bruno | `process-scaffold-cleanup-spec` · `makefile-automation-spec` |
| Process lifecycle, manager, resolución de versiones, execution keys | `process-lifecycle-runtime-spec` |
| Refresco de progreso batch, pendientes desde `bulk_job_items` | `process-lifecycle-runtime-spec` · `batch-preview-spec`* |
| Batch preview, `apply_changes`, `item_ids`, `row_numbers` | `batch-preview-spec` |
| Fanout, shards, Redis batch, `auto_invoke`, throttling/pacing | `batch-fanout-spec` · `batch-observability-spec` · `process-lifecycle-runtime-spec` |
| Cancelación operativa, kill switch, auto-cancel, guards de polling | `process-lifecycle-runtime-spec` · `batch-fanout-spec` · `batch-observability-spec` · `http-endpoints-spec`* |
| Export batch, `exportmanager`, archivos por lote, auto-cancel de exports | `export-pipelines-spec` + `doc/info/exports/exportmanager-bulkexport-v2.md` |
| Base de datos, modelos GORM, migraciones, queries, integridad | `database-schema-query-spec` |
| Helpers o utilidades compartidas | `shared-utils-spec` |
| Logger o ajuste de logging | `logger-runtime-spec` |

`*` = revisar solo cuando el cambio también impacta ese dominio secundario.

---

## Convenciones técnicas (resumen)

> Detalle completo: [`doc/specs/agents/conventions-index.md`](doc/specs/agents/conventions-index.md)

### Servicios

Todo servicio nuevo o refactorizado debe usar: **interface segregation · constructor injection · method receivers · implementación concreta no exportada** (salvo razón fuerte).

```go
type PaymentService interface {
    Process(ctx context.Context, amount float64) error
    Refund(ctx context.Context, id string) error
}

type paymentService struct { db *sql.DB; logger *slog.Logger }

func NewPaymentService(db *sql.DB, logger *slog.Logger) PaymentService {
    return &paymentService{db: db, logger: logger}
}
```

### Batch (`batchflow`)

- Un único `process_type` por dominio de negocio. Diferencias `sequential`/`fanout` viven en `process_versions`.
- Todo proceso batch nuevo sobre `batchflow` que participe en cancelación operativa debe registrar su `Manager` en el registry central resuelto por `execution_key`. No hardcodear en consumers o handlers.
- Scaffold: `make create-batch-process ...`
- Cleanup: `make delete-process kind=batch-process service_slug=...`

### Export batch (`exportmanager`)

- Todo export batch nuevo debe registrar su `Manager` en el registry central de exports resuelto por `execution_key`.
- Si un export legacy no usa `exportmanager.Manager`, documentarlo explícitamente como excepción hasta migrarlo.
- Scaffold: ver `make list-scaffolds`

### Integraciones HTTP externas

- Configuración desde `internal/appconfig/config.yml` sección `apis.xxx`, vía `appConfig.APIConfig("xxx")`.
- Centralizar en `internal/services/externalhttp/`. Adapters delegan en `externalhttp.NewClientFromAPIConfig(...)`.
- No usar `resty.New()` directamente en adapters nuevos.
- Scaffold: `make create-external-adapter adapter_name=<name> config_key=<key>`

### Logger

| Contexto | Convención |
|----------|-----------|
| Producción (Lambda/EKS) | `stdout` — AWS captura via CloudWatch |
| Local | `logger.GetLogger("nombre_proceso")` por proceso específico |
| Debug puntual | `logger.GetLoggerToFile("nombre_proceso", "...")` — nunca un archivo global |

**Tipos de log obligatorios**:
- `log_type=redis_guard` — errores Redis relevantes en el core batch
- `log_type=rate_limit_guard` — rate limit interno y `429` externos
- `log_type=execution_guard` — cancelaciones manuales, auto-cancel, pausas operativas

### Compatibilidad multi-entorno

Todo cambio de infraestructura, runtime, bootstrap, colas, endpoints AWS, vars de entorno, Terraform, Helm/K8s, Docker Compose o LocalStack debe evaluarse para **ambos entornos objetivo**: `lambda` y `eks`. Si el comportamiento difiere, documentarlo explícitamente.

### Makefile — contexto de ejecución

Todo comando nuevo que abra conexiones a Postgres, Redis, colas u otra infraestructura debe ejecutarse dentro del contexto Docker Compose, preferentemente via `$(DC_RUN)` o wrapper equivalente. No asumir que `go run ./cmd/tools/...` desde host funciona igual que dentro del contenedor.

### Catálogos operativos

- `make list-scaffolds` — generadores y scaffolds reutilizables
- `make list-tools` — utilidades operativas frecuentes por dominio

Al agregar scaffold o utilidad operativa nueva, evaluar y actualizar ambos catálogos en la misma solicitud.

### Base de datos

Cuando cambien tablas, relaciones, índices, enums o reglas de integridad, actualizar:
1. `doc/specs/data/database-schema-query-spec.md`
2. Documentación humana en `doc/info/data/` si el cambio altera el entendimiento del modelo

### Endpoints y Bruno

- Todo endpoint nuevo → documentación humana en `doc/info/` con ejemplo de body si aplica.
- Contratos reutilizables → también en `doc/specs/`.
- Colección canónica Bruno en `bruno/api/`. Legacy y exploratorio en `bruno/legacy/`.
- Endpoints protegidos: `auth: bearer` + `{{access_token}}`. Header operativo: `X-Client-Code: bruno`.

---

## Regla de cierre

Antes de cerrar cualquier tarea, verificar explícitamente:

1. ¿Las specs del dominio afectado fueron revisadas?
2. ¿Las specs revisadas siguen alineadas con el cambio implementado?
3. ¿Se actualizó este archivo si se agregó o movió una spec?
4. ¿Se actualizaron `doc/info/README.md` y `doc/specs/README.md` si cambió el mapa documental?

> Si el cambio toca un área de la [Matriz de cambios](#matriz-de-cambios) y su spec no fue revisada, el trabajo debe considerarse **incompleto**.

---

## Referencias

### Documentación de referencia humana (`doc/info/`)

| Documento | Dominio |
|-----------|---------|
| [`doc/info/README.md`](doc/info/README.md) | Índice general |
| [`doc/info/development/service-design-conventions.md`](doc/info/development/service-design-conventions.md) | Arquitectura |
| [`doc/info/development/external-http-service-standard.md`](doc/info/development/external-http-service-standard.md) | Integraciones HTTP |
| [`doc/info/development/service-runtime-and-scaffold.md`](doc/info/development/service-runtime-and-scaffold.md) | Runtime y scaffold |
| [`doc/info/development/process-scaffold-and-cleanup.md`](doc/info/development/process-scaffold-and-cleanup.md) | Scaffold y cleanup |
| [`doc/info/platform/makefile-guide.md`](doc/info/platform/makefile-guide.md) | Makefile |
| [`doc/info/process-lifecycle/runtime.md`](doc/info/process-lifecycle/runtime.md) | Process lifecycle |
| [`doc/info/process-lifecycle/batch-preview-guide.md`](doc/info/process-lifecycle/batch-preview-guide.md) | Batch preview |
| [`doc/info/process-lifecycle/batch-fanout-guide.md`](doc/info/process-lifecycle/batch-fanout-guide.md) | Batch fanout |
| [`doc/info/process-lifecycle/batch-capacity-and-stress-guide.md`](doc/info/process-lifecycle/batch-capacity-and-stress-guide.md) | Capacidad y stress |
| [`doc/info/api/http-endpoints-guide.md`](doc/info/api/http-endpoints-guide.md) | API endpoints |
| [`doc/info/exports/exportmanager-bulkexport-v2.md`](doc/info/exports/exportmanager-bulkexport-v2.md) | Exportmanager |

### Specs normativas (`doc/specs/`)

| Documento | Dominio |
|-----------|---------|
| [`doc/specs/README.md`](doc/specs/README.md) | Índice de specs |
| [`doc/specs/documentation-governance-spec.md`](doc/specs/documentation-governance-spec.md) | Gobierno documental |
| [`doc/specs/documentation-defaults-spec.md`](doc/specs/documentation-defaults-spec.md) | Gobierno documental |
| [`doc/specs/shared/shared-utils-spec.md`](doc/specs/shared/shared-utils-spec.md) | Shared |
| [`doc/specs/architecture/core-architecture-spec.md`](doc/specs/architecture/core-architecture-spec.md) | Arquitectura |
| [`doc/specs/architecture/service-design-spec.md`](doc/specs/architecture/service-design-spec.md) | Arquitectura |
| [`doc/specs/architecture/service-runtime-bootstrap-spec.md`](doc/specs/architecture/service-runtime-bootstrap-spec.md) | Arquitectura |
| [`doc/specs/architecture/external-http-client-spec.md`](doc/specs/architecture/external-http-client-spec.md) | Arquitectura |
| [`doc/specs/api/http-endpoints-spec.md`](doc/specs/api/http-endpoints-spec.md) | API |
| [`doc/specs/platform/platform-runtime-spec.md`](doc/specs/platform/platform-runtime-spec.md) | Platform |
| [`doc/specs/platform/logger-runtime-spec.md`](doc/specs/platform/logger-runtime-spec.md) | Platform |
| [`doc/specs/platform/makefile-automation-spec.md`](doc/specs/platform/makefile-automation-spec.md) | Platform |
| [`doc/specs/platform/process-scaffold-cleanup-spec.md`](doc/specs/platform/process-scaffold-cleanup-spec.md) | Platform |
| [`doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md`](doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md) | Process Lifecycle |
| [`doc/specs/process-lifecycle/batch-preview-spec.md`](doc/specs/process-lifecycle/batch-preview-spec.md) | Process Lifecycle |
| [`doc/specs/process-lifecycle/batch-fanout-spec.md`](doc/specs/process-lifecycle/batch-fanout-spec.md) | Process Lifecycle |
| [`doc/specs/process-lifecycle/batch-observability-spec.md`](doc/specs/process-lifecycle/batch-observability-spec.md) | Process Lifecycle |
| [`doc/specs/exports/export-pipelines-spec.md`](doc/specs/exports/export-pipelines-spec.md) | Exports |
| [`doc/specs/data/database-schema-query-spec.md`](doc/specs/data/database-schema-query-spec.md) | Data |
| [`doc/specs/agents/change-matrix.md`](doc/specs/agents/change-matrix.md) | Agentes IA |
| [`doc/specs/agents/conventions-index.md`](doc/specs/agents/conventions-index.md) | Agentes IA |
