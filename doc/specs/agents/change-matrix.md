---
domain: agents
summary: Matriz declarativa de specs a revisar según el tipo de cambio. Fuente de verdad para decisiones de revisión previa a implementación.
when_to_read:
  - antes de implementar o modificar cualquier código
  - cuando hay duda sobre qué specs aplican a un cambio
  - cuando se incorpora un colaborador o agente nuevo al proyecto
code_paths:
  - doc/specs/
  - AGENTS.md
related_info:
  - doc/info/README.md
related_specs:
  - doc/specs/documentation-governance-spec.md
  - doc/specs/agents/conventions-index.md
status: active
---

# change-matrix.md

Matriz completa de revisión por tipo de cambio. Complementa el resumen en `AGENTS.md`.

> **Regla de uso**: Identificar el tipo de cambio, revisar **todas** las specs marcadas como obligatorias, y las secundarias (`*`) cuando el cambio también impacte ese dominio.

---

## Documentación

### Documentación o mapa documental

**Specs obligatorias**:
- `doc/specs/documentation-governance-spec.md`
- `doc/specs/documentation-defaults-spec.md`

**Criterio de aplicación**: cualquier cambio en estructura de carpetas doc, clasificación info/specs, nuevos documentos, reorganización de índices, cambios en `README.md` raíz o en `doc/info/README.md` / `doc/specs/README.md`.

---

## Arquitectura y servicios

### Servicios, casos de uso, diseño de interfaces o refactors estructurales

**Specs obligatorias**:
- `doc/specs/architecture/service-design-spec.md`
- `doc/specs/architecture/core-architecture-spec.md`

**Criterio**: creación de nuevos servicios, rediseño de interfaces existentes, cambios en contratos entre capas, refactors que muevan lógica entre servicios o repositorios.

---

### Wiring, bootstrap, runtime de servicios o registro de dependencias

**Specs obligatorias**:
- `doc/specs/architecture/service-runtime-bootstrap-spec.md`
- `doc/specs/platform/platform-runtime-spec.md`

**Criterio**: cambios en el orden de inicialización, nuevo registro de dependencias, modificación del contenedor de inyección, cambios en cómo se resuelven las dependencias en runtime.

---

### Infraestructura, Terraform, Helm/K8s, Docker Compose, LocalStack o env vars que afecten runtime

**Specs obligatorias**:
- `doc/specs/architecture/service-runtime-bootstrap-spec.md`
- `doc/specs/platform/platform-runtime-spec.md`

**Spec secundaria** (cuando también cambie la forma de ejecutar comandos o utilidades operativas):
- `doc/specs/platform/makefile-automation-spec.md`

**Criterio**: cualquier cambio que modifique variables de entorno, servicios de infraestructura, configuración de contenedores o entornos de ejecución. Evaluar impacto en ambos entornos: `lambda` y `eks`.

---

### Integraciones HTTP externas o cambios en `internal/services/externalhttp/` y adapters

**Specs obligatorias**:
- `doc/specs/architecture/external-http-client-spec.md`
- `doc/specs/platform/logger-runtime-spec.md`

**Spec secundaria** (cuando también impacte batch, rate limit, Redis o fanout):
- `doc/specs/process-lifecycle/batch-observability-spec.md`

**Criterio**: nuevo adapter HTTP externo, cambio en cliente compartido `externalhttp`, modificación de configuración de APIs externas, cambio en manejo de `429`, autenticación o errores de red.

---

## API

### Endpoints HTTP, DTOs, handlers, auth de endpoints o requests Bruno

**Specs obligatorias**:
- `doc/specs/api/http-endpoints-spec.md`

**Criterio**: endpoint nuevo, modificación de contrato HTTP existente (path, method, body, headers, auth), cambio en DTOs de request/response, nuevos requests Bruno canónicos.

---

## Platform

### Makefile, automatizaciones operativas, catálogos `list-scaffolds` o `list-tools`

**Specs obligatorias**:
- `doc/specs/platform/makefile-automation-spec.md`

**Spec secundaria** (si también afecta scaffolds o cleanup):
- `doc/specs/platform/process-scaffold-cleanup-spec.md`

**Criterio**: nuevo target en Makefile, modificación de targets existentes, cambio en catálogos operativos, nueva automatización de CI/CD.

**Cuando cambie cualquiera de los catálogos, revisar también**:
- `doc/info/platform/makefile-guide.md`
- `doc/info/development/process-scaffold-and-cleanup.md`
- `doc/specs/platform/makefile-automation-spec.md`
- `doc/specs/platform/process-scaffold-cleanup-spec.md`

---

### Comandos Go bajo `cmd/tools/`, utilidades operativas conectadas a DB/Redis/colas

**Specs obligatorias**:
- `doc/specs/platform/makefile-automation-spec.md`

**Spec secundaria** (cuando también cambie el contexto de ejecución o bootstrap compartido):
- `doc/specs/architecture/service-runtime-bootstrap-spec.md`

**Criterio**: nueva utilidad operativa Go, modificación de herramienta existente, cambio en cómo se accede a Postgres/Redis/colas desde herramientas CLI.

---

### Scaffolds, cleanup de procesos, batch/export scaffolds o convenciones Bruno genéricas

**Specs obligatorias**:
- `doc/specs/platform/process-scaffold-cleanup-spec.md`
- `doc/specs/platform/makefile-automation-spec.md`

**Criterio**: nuevo scaffold, modificación de scaffold existente, cambio en convenciones de nombres o estructura generada por scaffolds.

---

## Process Lifecycle

### Process lifecycle, manager, runtime del motor, resolución de versiones o execution keys

**Specs obligatorias**:
- `doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md`

**Criterio**: cambio en el motor batch, en cómo se resuelven versiones de proceso, en execution keys, en el registro de managers, en flujo de estados de corridas.

---

### Refresco de progreso batch o semántica de pendientes por lote desde `bulk_job_items`

**Specs obligatorias**:
- `doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md`

**Spec secundaria** (si también cambia `apply_changes`):
- `doc/specs/process-lifecycle/batch-preview-spec.md`

**Criterio**: hooks de refresco por lote, cambios en cómo se calculan o actualizan los pendientes derivados de `bulk_job_items`.

---

### Batch preview, `apply_changes`, selección por `item_ids`, `row_numbers` o preview batch

**Specs obligatorias**:
- `doc/specs/process-lifecycle/batch-preview-spec.md`

**Criterio**: cambio en la lógica de preview, en cómo se seleccionan items para procesar, en el contrato de `apply_changes`.

---

### Fanout, shards, capacidad, Redis batch, observabilidad batch, `auto_invoke` con delay o throttling/pacing

**Specs obligatorias**:
- `doc/specs/process-lifecycle/batch-fanout-spec.md`
- `doc/specs/process-lifecycle/batch-observability-spec.md`
- `doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md`

**Criterio**: cambio en estrategia de fanout, configuración de shards, lógica de pacing/throttling, interacción con Redis para coordinación batch.

---

### Cancelación operativa de corridas batch, kill switch distribuida, auto-cancel o guards de polling

**Specs obligatorias**:
- `doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md`
- `doc/specs/process-lifecycle/batch-fanout-spec.md`
- `doc/specs/process-lifecycle/batch-observability-spec.md`

**Spec secundaria** (si también se expone endpoint operativo):
- `doc/specs/api/http-endpoints-spec.md`

**Criterio**: nuevo mecanismo de cancelación, cambio en threshold de auto-cancel, modificación de guards de polling distribuido.

---

## Exports

### Export batch, `exportmanager`, generación de archivos por lotes, auto-cancel de exports o registro por `execution_key`

**Specs obligatorias**:
- `doc/specs/exports/export-pipelines-spec.md`
- `doc/info/exports/exportmanager-bulkexport-v2.md`

**Criterio**: nuevo export batch, modificación de `exportmanager`, cambio en generación de archivos, integración de auto-cancel en exports.

---

## Data

### Base de datos, modelos GORM, migraciones, relaciones, queries e integridad

**Specs obligatorias**:
- `doc/specs/data/database-schema-query-spec.md`

**Criterio**: cualquier migración SQL, cambio en modelos GORM, nueva tabla, relación, índice, enum, constraint o regla de integridad. Si el cambio también altera el entendimiento humano del modelo, actualizar `doc/info/data/`.

> **Regla**: Si una migración cambia el modelo y no se actualiza esta spec, el trabajo debe considerarse **incompleto**.

---

## Shared

### Helpers o utilidades compartidas

**Specs obligatorias**:
- `doc/specs/shared/shared-utils-spec.md`

**Criterio**: nueva función compartida, modificación de helper existente con impacto en múltiples dominios, cambio de semántica en utilidades reutilizables.

> **Regla**: Si una función compartida cambia de semántica y la spec no se actualiza, el trabajo debe considerarse **incompleto**.

---

## Resumen rápido (lookup por keyword)

| Keyword en el cambio | Specs mínimas a revisar |
|---------------------|------------------------|
| `externalhttp`, adapter HTTP | `external-http-client-spec` |
| `batchflow`, proceso batch | `process-lifecycle-runtime-spec` |
| `exportmanager`, export batch | `export-pipelines-spec` |
| `apply_changes`, preview | `batch-preview-spec` |
| fanout, shard, pacing | `batch-fanout-spec` · `batch-observability-spec` |
| cancelación, kill switch, auto-cancel | `process-lifecycle-runtime-spec` · `batch-fanout-spec` |
| migración SQL, GORM | `database-schema-query-spec` |
| endpoint, handler, DTO | `http-endpoints-spec` |
| scaffold, `make create-*` | `process-scaffold-cleanup-spec` · `makefile-automation-spec` |
| logger, log | `logger-runtime-spec` |
| bootstrap, wiring | `service-runtime-bootstrap-spec` · `platform-runtime-spec` |
| helper, util compartido | `shared-utils-spec` |
| doc, spec, README | `documentation-governance-spec` · `documentation-defaults-spec` |
