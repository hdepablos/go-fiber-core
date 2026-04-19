# Dispatch Pacing Guide

## Objetivo

Documentar cómo usar `dispatch_pacing` en el step `process_batch` para dosificar el procesamiento de items en tandas controladas.

Este documento describe el comportamiento operativo.
Las reglas verificables relacionadas viven en:

- `doc/specs/process-lifecycle/batch-preview-spec.md`
- `doc/specs/process-lifecycle/batch-fanout-spec.md`

## Qué problema resuelve

Hay escenarios donde no conviene procesar todo el lote de golpe.

Ejemplo:

- hay `1000` items para despachar a SQS,
- se quiere procesar `100`,
- esperar `10` segundos,
- y recién después procesar los siguientes `100`.

`dispatch_pacing` resuelve exactamente ese patrón.

No es un throttle defensivo por request.
Es una estrategia explícita de dosificación por tandas.

## Dónde se configura

El bloque vive dentro del `config` del step `process_batch`.

Ejemplo mínimo:

```json
{
  "dispatch_pacing": {
    "enabled": true,
    "messages_per_interval": 100,
    "interval_seconds": 10
  }
}
```

Ejemplo completo:

```json
{
  "concurrent_batches": 1,
  "dispatch_pacing": {
    "enabled": true,
    "messages_per_interval": 100,
    "interval_seconds": 10
  },
  "execution_mode": {
    "type": "sequential"
  },
  "execution_policy": {
    "mode": "ASYNC",
    "label": "mi proceso",
    "auto_invoke": {
      "enabled": true,
      "cursor_field": "batch_index",
      "stop_condition": "is_last_batch"
    },
    "next_step": "bulk/process/x/finalize"
  }
}
```

## Significado de los campos

- `enabled`: activa o desactiva el pacing.
- `messages_per_interval`: cantidad de items que se procesan por tanda.
- `interval_seconds`: segundos de espera entre tandas sucesivas.

## Comportamiento por default

Si el bloque `dispatch_pacing` no existe, o si `enabled=false`, el comportamiento queda desactivado.

Eso significa:

- no hay espera artificial,
- no hay partición extra por pacing,
- el `process_batch` sigue con su semántica normal.

## Semántica operativa

Si un `Batch.Items` trae `10` items y la configuración es:

```json
{
  "dispatch_pacing": {
    "enabled": true,
    "messages_per_interval": 3,
    "interval_seconds": 5
  }
}
```

el procesamiento queda conceptualmente así:

1. procesa `3`
2. espera hasta la próxima ventana
3. procesa `3`
4. espera hasta la próxima ventana
5. procesa `3`
6. espera hasta la próxima ventana
7. procesa `1`

Resultado esperado:

- `chunk_count = 4`
- `chunk_sizes = [3, 3, 3, 1]`

## Sequential vs Fanout

### Sequential

En `sequential`, el comportamiento es el más intuitivo:

- la misma corrida procesa una tanda,
- espera la siguiente ventana,
- continúa con la siguiente tanda.

### Fanout

En `fanout`, el pacing también funciona, pero la coordinación se hace con Redis usando `input.RedisKey`.

Eso significa que el pacing es:

- global por corrida,
- compartido por todos los shards de esa corrida.

Ejemplo:

- `parallel_shards = 4`
- `messages_per_interval = 100`
- `interval_seconds = 10`

Interpretación correcta:

- entre todos los shards juntos se habilitan `100` items por ventana de `10` segundos.

Interpretación incorrecta:

- cada shard puede procesar `100` cada `10` segundos.

## Relación con `concurrent_batches`

`concurrent_batches` y `dispatch_pacing` controlan cosas distintas.

- `concurrent_batches`: cuántos batches se intentan procesar en la misma invocación.
- `dispatch_pacing`: cuántos items se dejan pasar por ventana temporal.

Cuando `dispatch_pacing` está activo:

- el manager prioriza respetar la ventana,
- y procesa las tandas en orden,
- aunque el proceso tenga capacidad de paralelismo mayor.

## Coordinación en Redis

La ventana se coordina usando Redis con claves derivadas de `input.RedisKey`.

Consecuencias:

- dos corridas distintas con distinta `key_redis` no comparten pacing,
- los shards de una misma corrida sí comparten pacing,
- `preview apply_changes` también usa ese mismo criterio.

## Preview y `apply_changes`

`POST /api/v1/process-lifecycle/batch-preview` soporta `dispatch_pacing` cuando:

- el proceso tiene `process_batch` configurado con ese bloque,
- y el preview se ejecuta con `apply_changes=true`.

En ese caso:

- el preview sigue renderizando los items normalmente,
- y la fase de persistencia real respeta el pacing del step.

La respuesta ahora expone:

- `applied_changes = true`
- `apply_changes_metadata.dispatch_pacing`

Ejemplo de metadata esperada:

```json
{
  "dispatch_pacing": {
    "enabled": true,
    "messages_per_interval": 3,
    "interval_seconds": 5,
    "chunk_count": 4,
    "chunk_sizes": [3, 3, 3, 1],
    "waits_ms": [0, 5000, 5000, 5000],
    "slots": [170000000, 170000001, 170000002, 170000003]
  }
}
```

## Cómo probarlo desde Bruno

Requests recomendados:

- `bruno/api/v1/process-lifecycle/post-batch-preview-apply-changes-pacing.bru`
- `bruno/legacy/process-lifecycle/test-batch-process/preview - batch - item_ids - apply_changes.bru`

Configuración sugerida para test rápido:

```json
{
  "dispatch_pacing": {
    "enabled": true,
    "messages_per_interval": 3,
    "interval_seconds": 5
  }
}
```

Con `10` items, se espera:

- que el request tarde más que un preview sin pacing,
- que la metadata muestre `4` tandas,
- que `waits_ms` refleje las pausas entre ventanas.

## Recomendaciones de uso

- arrancar con valores chicos para verificar el comportamiento,
- usar `preview apply_changes` en local antes de probar `run`,
- en `fanout`, pensar el pacing como límite global de la corrida,
- no mezclar este concepto con throttle HTTP o cooldown.

## Cuándo conviene usarlo

Conviene cuando:

- el destino no debe recibir demasiados mensajes de golpe,
- se necesita cadencia estable,
- se quiere controlar el ritmo total de salida a SQS,
- se quiere validar el comportamiento desde Bruno en una muestra acotada.

No es el mecanismo principal cuando el problema real es:

- retry de dependencias externas,
- backoff por error,
- throttle por status HTTP `429`.

## Trazabilidad

- Código:
  - `internal/services/batchflow/dispatch_pacing.go`
  - `internal/services/batchflow/manager.go`
  - `internal/services/batchflow/preview_service.go`
  - `internal/handlers/process_lifecycle_handler.go`
  - `internal/services/bulkprocess/steps.go`
  - `internal/services/punitorios/steps.go`
  - `internal/services/imputations/steps.go`
- Bruno:
  - `bruno/api/v1/process-lifecycle/post-batch-preview-apply-changes-pacing.bru`
  - `bruno/legacy/process-lifecycle/test-batch-process/preview - batch - item_ids - apply_changes.bru`
- Documentos relacionados:
  - `doc/info/process-lifecycle/batch-preview-guide.md`
  - `doc/info/process-lifecycle/batch-fanout-guide.md`
  - `doc/info/process-lifecycle/testing-guide.md`
