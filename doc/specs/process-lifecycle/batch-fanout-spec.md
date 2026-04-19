---
domain: process-lifecycle
summary: Contrato del modo fan-out distribuido de batchflow, su coordinación por shards, finalize único y compatibilidad con dispatch_pacing basado en auto_invoke con delay.
when_to_read:
  - cambios en fanout batch
  - cambios en shards o auto_invoke
  - cambios en finalize distribuido
  - cambios en dispatch_pacing dentro de fanout
  - cambios en cancelacion de corridas batch
  - cambios en auto-cancel por errores repetidos
code_paths:
  - internal/services/batchflow/
  - internal/services/bulkprocess/steps.go
  - internal/services/punitorios/steps.go
  - internal/services/imputations/steps.go
  - cmd/sqs-consumer/main.go
related_info:
  - doc/info/process-lifecycle/batch-fanout-guide.md
  - doc/info/process-lifecycle/batch-fanout-risks.md
  - doc/info/process-lifecycle/dispatch-pacing-guide.md
related_specs:
  - doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md
  - doc/specs/process-lifecycle/batch-observability-spec.md
status: active
---

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
- Si `dispatch_pacing` está activo y un batch queda parcial, el shard debe poder re-invocar el mismo `batch_index` hasta agotar la tanda completa.

## Reglas de `auto_invoke`

- En fan-out, `auto_invoke.stop_condition` debe basarse en fin de shard.
- La condición recomendada es `is_shard_complete`.
- No debe usarse `is_last_batch` como stop condition global del fan-out.
- Si `dispatch_pacing` está activo, el delay de la re-invocación debe salir de `dispatch_pacing.interval_seconds`.
- `dispatch_pacing.interval_seconds` debe validarse en rango seguro para Lambda; el contrato actual admite `1..10`.
- Antes de re-encolar un shard por `auto_invoke`, el worker debe poder verificar si la corrida fue cancelada.
- Si la corrida fue cancelada, `auto_invoke` no debe re-encolar el mismo shard ni disparar `next_step`.

## Reglas de coordinación global

Redis debe mantener como mínimo:

- total de shards
- cantidad de shards completados
- marca idempotente por shard
- lock de finalize
- estado de cancelacion de la corrida
- lock de stop/cancel para evitar side effects repetidos

Si `dispatch_pacing` está activo:

- el avance parcial del batch debe persistirse usando estado derivado de `input.RedisKey`,
- la coordinación del ritmo debe ser global por corrida,
- y no debe interpretarse como cupo independiente por shard.

## Reglas de finalización

- Cada shard puede detectar que terminó su propia rama.
- Solo el último shard global debe habilitar el dispatch de `finalize`.
- `finalize` debe ejecutarse una sola vez por corrida.
- La protección de finalize debe usar coordinación distribuida.
- Si la corrida ya fue cancelada, `finalize` no debe consolidar ni reactivar el flujo como si fuera exitoso.

## Invariantes

- Un shard completado no debe incrementar dos veces el contador global.
- Un reintento de un shard ya marcado no debe volver a habilitar finalize.
- `finalize` no puede limpiarse antes de que todos los shards terminen.
- La limpieza Redis debe ocurrir después del finalize único.
- Una corrida cancelada no debe seguir generando mensajes `auto_invoke`.
- Un auto-cancel por threshold no debe depender del worker exacto que detectó el error.
- Si el proceso batch define `ParentLifecycle.Fail`, el auto-cancel debe invocarlo para dejar el padre en estado consistente.

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
- `execution_policy.auto_invoke.delay_seconds` opcional, pero si existe y `dispatch_pacing` está activo debe coincidir con `dispatch_pacing.interval_seconds`
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
- `dispatch_pacing` no debe implementarse con espera bloqueante dentro de una Lambda de producción.
- El fan-out no reemplaza una guarda operativa de cancelacion o auto-cancel por errores repetidos.

## Acceptance Criteria

- Un proceso batch puede despachar más de un shard inicial.
- Cada shard continúa su propia cadena con `auto_invoke`.
- El último shard global dispara `finalize` una sola vez.
- El mismo contrato funciona sobre Lambda y EKS.
- El mismo contrato puede combinarse con `dispatch_pacing` global por corrida.
- La documentación humana debe incluir una guía de riesgos separada.
- Una corrida fan-out cancelada deja de re-encolar shards y no dispara `finalize` adicional.

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
