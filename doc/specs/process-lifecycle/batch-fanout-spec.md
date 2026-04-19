# Batch Fanout Spec

## Objetivo

Formalizar el contrato del modo fan-out distribuido del motor `batchflow` para ejecución sobre Lambda y EKS.

## Alcance

Aplica a procesos batch que:

- preparan batches en Redis,
- despachan múltiples shards,
- ejecutan `process_batch` asíncrono con `auto_invoke`,
- finalizan una sola vez al terminar todos los shards.

Complementa:

- `doc/info/process-lifecycle/batch-fanout-guide.md`
- `doc/info/process-lifecycle/batch-fanout-risks.md`
- `doc/info/process-lifecycle/dispatch-pacing-guide.md`

## Contrato mínimo del pipeline

El pipeline batch con fan-out debe contemplar estos pasos lógicos:

1. `start`
2. `dispatch_shards`
3. `process_batch`
4. `finalize`

## Reglas de `start`

- Debe validar el padre de negocio.
- Debe cargar y particionar registros.
- Debe persistir summary y batches en Redis.
- Debe devolver `key_redis`, `batches_list_key` y `total_batches`.

## Reglas de `dispatch_shards`

- Debe calcular `total_shards` efectivo.
- `total_shards` no puede ser menor que `1`.
- `total_shards` no puede ser mayor que `total_batches`.
- Debe registrar el estado global de shards en Redis.
- Debe despachar un mensaje inicial de `process_batch` por shard.
- Debe cortar la cadena síncrona original del lifecycle.

## Reglas de reparto

La estrategia inicial soportada debe ser `stride`.

Para `parallel_shards = N`, el shard `s` debe procesar índices:

- `s`
- `s + N`
- `s + 2N`
- `s + 3N`

y así sucesivamente, respetando `total_batches`.

## Reglas de `process_batch`

- Debe aceptar al menos:
  - `batch_index`
  - `shard_index`
  - `total_shards`
  - `total_batches`
- Debe poder procesar más de un batch por invocación usando `concurrent_batches`.
- Debe poder respetar `dispatch_pacing` cuando esa configuración exista en el step.
- En fan-out, el próximo cursor del shard debe avanzar por stride:
  - `next_batch_index = current_batch_index + concurrent_batches * total_shards`

## Reglas de `auto_invoke`

- En fan-out, `auto_invoke.stop_condition` debe basarse en fin de shard.
- La condición recomendada es `is_shard_complete`.
- No debe usarse `is_last_batch` como stop condition global del fan-out.

## Reglas de coordinación global

Redis debe mantener como mínimo:

- total de shards
- cantidad de shards completados
- marca idempotente por shard
- lock de finalize

Si `dispatch_pacing` está activo:

- la coordinación de la ventana debe ser global por corrida,
- usando una clave derivada de `input.RedisKey`,
- y no debe interpretarse como cupo independiente por shard.

## Reglas de finalización

- Cada shard puede detectar que terminó su propia rama.
- Solo el último shard global debe habilitar el dispatch de `finalize`.
- `finalize` debe ejecutarse una sola vez por corrida.
- La protección de finalize debe usar coordinación distribuida.

## Invariantes

- Un shard completado no debe incrementar dos veces el contador global.
- Un reintento de un shard ya marcado no debe volver a habilitar finalize.
- `finalize` no puede limpiarse antes de que todos los shards terminen.
- La limpieza Redis debe ocurrir después del finalize único.

## Compatibilidad operativa

El contrato debe ser compatible con:

- Lambda
- EKS
- workers asíncronos múltiples
- semántica at-least-once de SQS

## Configuración mínima esperada

El step `process_batch` debe soportar:

- `concurrent_batches`
- `dispatch_pacing.enabled`
- `dispatch_pacing.messages_per_interval`
- `dispatch_pacing.interval_seconds`
- `parallel_shards`
- `execution_mode.type`
- `execution_mode.parallel_shards`
- `execution_mode.strategy`
- `execution_policy.mode`
- `execution_policy.auto_invoke.cursor_field`
- `execution_policy.auto_invoke.stop_condition`
- `execution_policy.next_step`

## Reglas de body HTTP

El body de `run` no necesita campos específicos de fan-out si la configuración vive en steps.
Debe seguir siendo compatible con:

- `process_type_id`
- `sede_id`
- `override_process_version_id`
- `roadmap`
- `input.id`
- `input.filters`

## Limitaciones explícitas

- El fan-out no reemplaza el throttle global para APIs externas.
- El fan-out no garantiza mejora lineal de performance.
- La cantidad de shards debe ajustarse a la capacidad del sistema y del proveedor externo.
- `dispatch_pacing` no multiplica el throughput por shard; limita el ritmo global de la corrida.

## Acceptance Criteria

- Un proceso batch puede despachar más de un shard inicial.
- Cada shard continúa su propia cadena con `auto_invoke`.
- El último shard global dispara `finalize` una sola vez.
- El mismo contrato funciona sobre Lambda y EKS.
- El mismo contrato puede combinarse con `dispatch_pacing` global por corrida.
- La documentación humana debe incluir una guía de riesgos separada.

## Trazabilidad

- `internal/services/batchflow/contracts.go`
- `internal/services/batchflow/manager.go`
- `internal/services/batchflow/state_store.go`
- `internal/services/bulkprocess/steps.go`
- `internal/services/punitorios/steps.go`
- `cmd/sqs-consumer/main.go`
- `doc/info/process-lifecycle/batch-fanout-guide.md`
- `doc/info/process-lifecycle/batch-fanout-risks.md`
- `doc/info/process-lifecycle/dispatch-pacing-guide.md`
