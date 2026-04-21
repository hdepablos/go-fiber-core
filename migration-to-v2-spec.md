---
domain: agents
summary: Spec de migración única para actualizar doc/info/, doc/specs/ y sus índices a la nueva estructura AGENTS.md v2 con dominios agents/, metadata SDD y trazabilidad completa.
when_to_read:
  - antes de ejecutar cualquier tarea de migración documental de esta iteración
  - cuando haya duda sobre qué cambiar en un archivo específico
  - cuando se requiera validar que la migración está completa
code_paths:
  - doc/info/
  - doc/specs/
  - AGENTS.md
  - README.md
related_info:
  - doc/info/README.md
related_specs:
  - doc/specs/documentation-governance-spec.md
  - doc/specs/documentation-defaults-spec.md
  - doc/specs/agents/change-matrix.md
  - doc/specs/agents/conventions-index.md
status: active
---

# migration-to-v2-spec.md

Instrucciones precisas para migrar el repositorio documental a la estructura definida en `AGENTS.md` v2.

> **Regla de uso para el agente**: Leer esta spec completa antes de modificar cualquier archivo. Ejecutar las tareas en el orden definido. No saltear ningún paso. Verificar el checklist de cierre antes de considerar la migración completa.

---

## Contexto del cambio

La migración introduce tres cambios estructurales:

1. **Nuevo dominio `agents/`** en `doc/specs/agents/` con archivos ya creados que necesitan estar enlazados correctamente en los índices.
2. **Metadata SDD obligatoria** en todas las specs de `doc/specs/` que aún no la tienen.
3. **Actualización de índices** en `doc/info/README.md`, `doc/specs/README.md` y `AGENTS.md` para reflejar la nueva estructura.

---

## Estado actual detectado

### Archivos nuevos YA CREADOS (no mover, solo enlazar)

```
doc/specs/agents/change-matrix.md        ← ya existe con metadata completa
doc/specs/agents/conventions-index.md    ← ya existe con metadata completa
AGENTS.md                                ← ya reemplazado con versión v2
```

### Archivo en ubicación incorrecta

```
doc/info/agents/change-matrix.md   ← DEBE ELIMINARSE
```

Este archivo es una copia del que ya vive en `doc/specs/agents/change-matrix.md`. Es material normativo (spec), no informativo (info). Su ubicación correcta es únicamente `doc/specs/agents/`.

### Specs SIN metadata SDD (requieren agregar el bloque frontmatter)

```
doc/specs/architecture/core-architecture-spec.md
doc/specs/architecture/service-runtime-bootstrap-spec.md
doc/specs/exports/export-pipelines-spec.md
doc/specs/platform/logger-runtime-spec.md
doc/specs/platform/platform-runtime-spec.md
doc/specs/documentation-defaults-spec.md
doc/specs/documentation-governance-spec.md
doc/specs/process-lifecycle/batch-observability-spec.md
```

### Specs CON metadata SDD (no requieren cambios de metadata)

```
doc/specs/agents/change-matrix.md
doc/specs/agents/conventions-index.md
doc/specs/api/http-endpoints-spec.md
doc/specs/architecture/external-http-client-spec.md
doc/specs/architecture/service-design-spec.md
doc/specs/data/database-schema-query-spec.md
doc/specs/platform/makefile-automation-spec.md
doc/specs/platform/process-scaffold-cleanup-spec.md
doc/specs/process-lifecycle/batch-fanout-spec.md
doc/specs/process-lifecycle/batch-preview-spec.md
doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md
doc/specs/shared/shared-utils-spec.md
```

### Índices que requieren actualización

```
doc/specs/README.md       ← agregar sección agents/ al índice
doc/info/README.md        ← eliminar referencia a doc/info/agents/ si existe
```

---

## Tareas de migración (ejecutar en orden)

### TAREA 1 — Eliminar archivo mal ubicado

**Archivo a eliminar**: `doc/info/agents/change-matrix.md`

**Razón**: Es material normativo (spec), no informativo. La fuente de verdad es `doc/specs/agents/change-matrix.md`.

```bash
rm doc/info/agents/change-matrix.md
rmdir doc/info/agents/   # solo si queda vacío
```

**Verificación previa**: Confirmar que `doc/specs/agents/change-matrix.md` existe y tiene contenido completo antes de eliminar.

---

### TAREA 2 — Agregar metadata SDD a specs sin frontmatter

Para cada archivo, prepender el bloque `---` al inicio del archivo existente sin modificar el resto del contenido.

#### 2.1 `doc/specs/documentation-governance-spec.md`

Prepender al inicio:

```yaml
---
domain: documentation
summary: Estructura documental oficial del repositorio, reglas de ubicación, clasificación, no duplicación y vinculación entre doc/info/ y doc/specs/.
when_to_read:
  - cambios en estructura de carpetas documentales
  - creación de nuevos dominios documentales
  - reorganización de índices o mapa documental
  - cuando haya duda sobre dónde ubicar un documento nuevo
code_paths:
  - doc/info/
  - doc/specs/
  - README.md
related_info:
  - doc/info/README.md
related_specs:
  - doc/specs/documentation-defaults-spec.md
status: active
---

```

#### 2.2 `doc/specs/documentation-defaults-spec.md`

Prepender al inicio:

```yaml
---
domain: documentation
summary: Comportamiento por defecto para documentar en el repositorio, separación info/specs, convenciones de clasificación por dominio y reglas de vinculación.
when_to_read:
  - ante cualquier solicitud de documentar, redocumentar o reorganizar
  - cuando haya duda sobre si crear info, spec o ambos
  - onboarding de nuevos colaboradores o agentes
code_paths:
  - doc/info/
  - doc/specs/
  - AGENTS.md
related_info:
  - doc/info/README.md
related_specs:
  - doc/specs/documentation-governance-spec.md
status: active
---

```

#### 2.3 `doc/specs/architecture/core-architecture-spec.md`

Prepender al inicio:

```yaml
---
domain: architecture
summary: Reglas base de arquitectura transversal para autenticación por provider, resolución de secretos y consistencia con Redis locking.
when_to_read:
  - cambios en providers de autenticación
  - cambios en estrategia de secretos o configuración
  - cambios en estrategia de caché o locking Redis
  - refactors transversales de arquitectura
code_paths:
  - internal/services/auth/
  - internal/appconfig/
related_info:
  - doc/info/architecture/authentication-providers.md
  - doc/info/architecture/configuration-secrets.md
  - doc/info/architecture/redis-locking-strategy.md
  - doc/info/architecture/process-architecture-evolution.md
related_specs:
  - doc/specs/architecture/service-design-spec.md
status: active
---

```

#### 2.4 `doc/specs/architecture/service-runtime-bootstrap-spec.md`

Prepender al inicio:

```yaml
---
domain: architecture
summary: Reglas del runtime de servicios y scaffold de export managers para evitar globals, mantener dispatcher y providers resolubles por contexto.
when_to_read:
  - cambios en bootstrap HTTP o SQS
  - cambios en dispatcher o providers de exportación
  - cambios en runtimebootstrap
  - scaffold de export managers o batch processes
code_paths:
  - internal/runtimebootstrap/bootstrap.go
  - internal/services/runtimectx/
  - internal/services/serviceconfig/executor.go
  - cmd/api/main.go
  - cmd/sqs-consumer/main.go
  - cmd/tools/export-manager-scaffold/
related_info:
  - doc/info/development/service-runtime-and-scaffold.md
  - doc/info/development/service-design-conventions.md
related_specs:
  - doc/specs/architecture/service-design-spec.md
  - doc/specs/platform/platform-runtime-spec.md
status: active
---

```

#### 2.5 `doc/specs/exports/export-pipelines-spec.md`

Prepender al inicio:

```yaml
---
domain: exports
summary: Contratos y acceptance criteria para pipelines de exportación por lotes con exportmanager, redis, S3, cancelación operativa y auto-cancel.
when_to_read:
  - exports batch nuevos sobre exportmanager
  - cambios en exportmanager.Manager
  - cambios en registro de exports por execution_key
  - cambios en auto-cancel de exports
  - cambios en generación de archivos por lote
code_paths:
  - internal/services/exportmanager/
  - internal/services/generar_archivo_banco_galicia/
  - internal/services/test/bulkexportV2/
related_info:
  - doc/info/exports/exportmanager-bulkexport-v2.md
  - doc/info/exports/exportmanager-generar-archivo-banco-galicia.md
  - doc/info/exports/bulk-export-generate-file-v1-async.md
related_specs:
  - doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md
status: active
---

```

#### 2.6 `doc/specs/platform/logger-runtime-spec.md`

Prepender al inicio:

```yaml
---
domain: platform
summary: Convención de runtime del logger por entorno, granularidad por proceso, tipos de log obligatorios y reglas de destino para producción y local.
when_to_read:
  - cambios en logger o logging estructurado
  - nuevas implementaciones que necesiten logging
  - cambios en tipos de log guard redis_guard rate_limit_guard execution_guard
  - observabilidad de batch o integraciones HTTP
code_paths:
  - internal/logger/
  - internal/logger/logger.go
  - internal/logger/guard_logs.go
related_info:
  - doc/info/operations/logs.md
related_specs:
  - doc/specs/process-lifecycle/batch-observability-spec.md
  - doc/specs/architecture/external-http-client-spec.md
status: active
---

```

#### 2.7 `doc/specs/platform/platform-runtime-spec.md`

Prepender al inicio:

```yaml
---
domain: platform
summary: Contrato operativo mínimo para desarrollo, configuración y despliegue en modos local, Lambda y EKS, variables de entorno y cambio de modo.
when_to_read:
  - cambios en infraestructura o entornos de despliegue
  - cambios en variables de entorno o configuración runtime
  - cambios en modos de ejecución lambda o eks
  - onboarding de DevOps o cambios en terraform o helm
code_paths:
  - terraform/
  - k8s/
  - docker-compose.yml
  - .env
related_info:
  - doc/info/development/development-workflow.md
  - doc/info/platform/devops-guide.md
  - doc/info/platform/manage-env-vars.md
  - doc/info/platform/eks-prerequisites.md
  - doc/info/platform/hybrid-deployment.md
related_specs:
  - doc/specs/architecture/service-runtime-bootstrap-spec.md
status: active
---

```

#### 2.8 `doc/specs/process-lifecycle/batch-observability-spec.md`

Prepender al inicio:

```yaml
---
domain: process-lifecycle
summary: Requisitos mínimos de observabilidad para batch con Redis, fanout y dependencias HTTP mediante log_type redis_guard, rate_limit_guard y execution_guard.
when_to_read:
  - cambios en observabilidad batch
  - cambios en log_type redis_guard o rate_limit_guard
  - cambios en auto-cancel o execution_guard
  - stress test o análisis de capacidad batch
  - cambios en adapters HTTP que participan en batch
code_paths:
  - internal/logger/guard_logs.go
  - internal/services/batchflow/state_store.go
  - internal/services/batchflow/throttle.go
  - internal/adapters/
related_info:
  - doc/info/process-lifecycle/batch-capacity-and-stress-guide.md
  - doc/info/operations/logs.md
related_specs:
  - doc/specs/process-lifecycle/batch-fanout-spec.md
  - doc/specs/architecture/external-http-client-spec.md
status: active
---

```

---

### TAREA 3 — Actualizar `doc/specs/README.md`

Localizar la sección "## Specs Disponibles" y agregar una nueva subsección `### Agents` con los links a los archivos nuevos.

La sección debe insertarse como **primera subsección** dentro de "Specs Disponibles", antes de "### Gobierno documental":

```markdown
### Agents

- [change-matrix.md](agents/change-matrix.md)
- [conventions-index.md](agents/conventions-index.md)
```

---

### TAREA 4 — Actualizar `doc/info/README.md`

Verificar si el archivo contiene alguna referencia a `agents/change-matrix.md` o a la carpeta `agents/`. Si existe alguna referencia, eliminarla.

El dominio `agents` no tiene documentación en `doc/info/`. Toda esa documentación es normativa y vive exclusivamente en `doc/specs/agents/`.

No agregar ninguna sección nueva de `agents` en `doc/info/README.md`.

---

## Checklist de cierre

Antes de considerar la migración completa, verificar cada ítem:

| # | Archivo | Verificación |
|---|---------|-------------|
| 1 | `doc/info/agents/change-matrix.md` | no existe |
| 2 | `doc/info/agents/` | no existe (si quedó vacío) |
| 3 | `doc/specs/agents/change-matrix.md` | existe con frontmatter completo |
| 4 | `doc/specs/agents/conventions-index.md` | existe con frontmatter completo |
| 5 | `doc/specs/documentation-governance-spec.md` | tiene bloque `---` al inicio |
| 6 | `doc/specs/documentation-defaults-spec.md` | tiene bloque `---` al inicio |
| 7 | `doc/specs/architecture/core-architecture-spec.md` | tiene bloque `---` al inicio |
| 8 | `doc/specs/architecture/service-runtime-bootstrap-spec.md` | tiene bloque `---` al inicio |
| 9 | `doc/specs/exports/export-pipelines-spec.md` | tiene bloque `---` al inicio |
| 10 | `doc/specs/platform/logger-runtime-spec.md` | tiene bloque `---` al inicio |
| 11 | `doc/specs/platform/platform-runtime-spec.md` | tiene bloque `---` al inicio |
| 12 | `doc/specs/process-lifecycle/batch-observability-spec.md` | tiene bloque `---` al inicio |
| 13 | `doc/specs/README.md` | incluye sección `### Agents` con 2 links |
| 14 | `doc/info/README.md` | no tiene referencias a `agents/` |
| 15 | `AGENTS.md` | referencia ambos archivos en `doc/specs/agents/` |

---

## Qué NO cambiar

El agente NO debe modificar:

- El cuerpo/contenido de ninguna spec existente (solo prepender frontmatter).
- Los archivos de `doc/info/**` excepto eliminar `doc/info/agents/change-matrix.md` y limpiar referencias en `doc/info/README.md`.
- Los archivos de código (`internal/`, `cmd/`, `bruno/`).
- El `README.md` raíz del proyecto.
- `AGENTS.md` (ya está actualizado).

---

## Estado de esta spec

Esta spec es **temporal**. Una vez que el checklist de cierre esté 100% verificado, puede archivarse cambiando `status: active` a `status: archived`.
