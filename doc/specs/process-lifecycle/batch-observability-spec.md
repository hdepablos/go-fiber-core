# Batch Observability Spec

## Objetivo

Formalizar los requisitos mínimos de observabilidad para procesos batch con Redis, fanout y dependencias HTTP.

## Alcance

Aplica a:

- `batchflow`,
- procesos batch scaffold generados sobre `batchflow`,
- data providers o adapters que hagan llamadas HTTP dentro del flujo batch.

## Familias de logs requeridas

El sistema debe emitir al menos tres familias de logs estructurados:

- `log_type=redis_guard`
- `log_type=rate_limit_guard`
- `log_type=execution_guard`

## Contratos mínimos de `redis_guard`

Los eventos `redis_guard` deben usarse para fallos de operación Redis relevantes para la ejecución batch.

Campos mínimos esperados:

- `log_type=redis_guard`
- `event_type=redis_operation_error`
- `operation`
- `component`

Campos recomendados cuando existan:

- `redis_key`
- `parent_id`
- `batch_index`
- `shard_index`
- `total_shards`
- `summary_key`
- `batch_key`
- `registry_key`

## Contratos mínimos de `rate_limit_guard`

Los eventos `rate_limit_guard` deben cubrir tanto limitación interna como respuestas externas `429`.

Campos mínimos esperados:

- `log_type=rate_limit_guard`
- `event_type`
- `scope`

### Scope interno

Para limitación interna del core:

- `scope=internal`
- `event_type` debe ser uno de:
  - `internal_rate_limit_cooldown`
  - `internal_rate_limit_max_inflight`
  - `internal_rate_limit_window`

Campos recomendados:

- `throttle_key`
- `max_requests`
- `current_requests`
- `max_in_flight`
- `current_in_flight`
- `per_seconds`

### Scope externo

Para dependencias HTTP:

- `scope=external`
- `event_type` debe ser uno de:
  - `external_http_429`
  - `external_dependency_timeout`
  - `external_dependency_error`

Campos recomendados:

- `source`
- `endpoint`
- `method`
- `status_code`
- `retry_after`

## Contratos mínimos de `execution_guard`

Los eventos `execution_guard` deben usarse para:

- cancelaciones manuales de corridas,
- auto-cancel por umbral de errores repetidos,
- y pausas o cortes operativos del polling cuando un mismo error de infraestructura se repite muchas veces.

Campos mínimos esperados:

- `log_type=execution_guard`
- `event_type`
- `component`

Campos recomendados cuando existan:

- `run_key`
- `parent_id`
- `reason`
- `requested_by`
- `source`
- `fingerprint`
- `error_count`
- `threshold`
- `cooldown`

## Invariantes

- Los errores Redis del core batch no deben perderse sin un evento `redis_guard`.
- Un `429` de dependencia externa no debe pasar silenciosamente sin un evento `rate_limit_guard`.
- Una cancelacion manual o automatica relevante no debe pasar silenciosamente sin un evento `execution_guard`.
- El logging de observabilidad no debe reemplazar el retorno de error original.
- La emisión de logs no debe alterar la semántica del flujo batch.
- La búsqueda operativa en producción debe poder hacerse filtrando por `log_type`.

## Reglas para stress test y capacidad

Antes de promover una configuración fanout a producción se debe poder relevar:

- baseline secuencial,
- corrida fanout conservadora,
- corrida fanout objetivo,
- presencia o ausencia de `redis_guard`,
- presencia o ausencia de `rate_limit_guard`.

No se debe promover una configuración si durante la prueba aparecen de forma sostenida:

- errores Redis,
- `external_http_429`,
- o `internal_rate_limit_*`.
- Si aparecen cancelaciones automáticas o pausas de polling, también deben poder relevarse con `execution_guard`.

## Acceptance Criteria

- El core batch registra errores Redis estructurados bajo `redis_guard`.
- El core batch registra limitación interna bajo `rate_limit_guard`.
- Los adapters HTTP instrumentados registran `429` externos bajo `rate_limit_guard`.
- El runtime batch registra cancelaciones y auto-cancel bajo `execution_guard`.
- La documentación humana explica cómo filtrar estos logs en producción.
- Existe una guía humana para checklist de capacidad y stress test.

## Trazabilidad

- `internal/logger/guard_logs.go`
- `internal/services/batchflow/state_store.go`
- `internal/services/batchflow/throttle.go`
- `internal/adapters/backoffice_adapter.go`
- `internal/adapters/discord_adapter.go`
- `doc/info/process-lifecycle/batch-capacity-and-stress-guide.md`
- `doc/info/operations/logs.md`
