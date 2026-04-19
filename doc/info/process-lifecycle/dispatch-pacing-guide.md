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
- dejar que el motor re-invoque el siguiente tramo `10` segundos después,
- y recién entonces procesar los siguientes `100`.

`dispatch_pacing` resuelve exactamente ese patrón.

No es un throttle defensivo por request.
No usa `sleep` dentro de la Lambda para el `run` real.
Es una estrategia explícita de dosificación por tandas entre invocaciones.

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
- `messages_per_interval`: cantidad máxima de items que una invocación puede procesar antes de re-encolarse.
- `interval_seconds`: delay entre una invocación y la siguiente.

## Reglas del contrato

Si `dispatch_pacing.enabled=true`, el step debe cumplir:

- `execution_policy.mode = ASYNC`
- `execution_policy.auto_invoke.enabled = true`
- `interval_seconds` entre `1` y `10`

No hace falta duplicar `delay_seconds` en `auto_invoke`.
Cuando `dispatch_pacing` está activo, el delay efectivo de la re-invocación sale de `dispatch_pacing.interval_seconds`.

Si alguien configura `execution_policy.auto_invoke.delay_seconds`, debe coincidir con `dispatch_pacing.interval_seconds`.

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

1. invocación 1 procesa `3`
2. el consumer re-encola el mismo step con delay de `5` segundos
3. invocación 2 procesa `3`
4. se vuelve a re-encolar con delay
5. invocación 3 procesa `3`
6. se vuelve a re-encolar con delay
7. invocación 4 procesa `1`

Resultado esperado:

- `chunk_count = 4`
- `chunk_sizes = [3, 3, 3, 1]`

## Comportamiento en Lambda

En `run` real:

- no se duerme la Lambda,
- no se bloquea la invocación esperando la siguiente ventana,
- cada invocación procesa una sola tanda,
- y `auto_invoke` agenda la siguiente con delay.

Esto evita agotar el timeout de invocación cuando el lote total es grande.

## Sequential vs Fanout

### Sequential

En `sequential`, el comportamiento es el más intuitivo:

- la misma corrida procesa una tanda,
- termina esa invocación,
- y la siguiente tanda se dispara por `auto_invoke` con delay.

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

- entre todos los shards juntos se habilitan `100` items por re-invocación de `10` segundos.

Interpretación incorrecta:

- cada shard puede procesar `100` cada `10` segundos.

## Relación con `concurrent_batches`

`concurrent_batches` y `dispatch_pacing` controlan cosas distintas.

- `concurrent_batches`: batches que el proceso intentaría procesar en una invocación normal.
- `dispatch_pacing`: items máximos permitidos por invocación.

Cuando `dispatch_pacing` está activo, el pacing tiene prioridad operativa.
En la práctica, el avance queda reducido a una sola tanda por invocación.

## Coordinación en Redis

El avance parcial del batch se coordina usando runtime Redis derivado de `input.RedisKey`.

Consecuencias:

- dos corridas distintas con distinta `key_redis` no comparten pacing,
- los shards de una misma corrida sí comparten pacing,
- el `run` real conserva el offset parcial del batch entre invocaciones.

## Preview y `apply_changes`

`POST /api/v1/process-lifecycle/batch-preview` soporta `dispatch_pacing` cuando:

- el proceso tiene `process_batch` configurado con ese bloque,
- y el preview se ejecuta con `apply_changes=true`.

En ese caso:

- el preview sigue renderizando los items normalmente,
- y `apply_changes` simula el pacing del step sin hacer esperas reales.

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
    "mode": "preview_simulated",
    "simulated": true,
    "chunk_count": 4,
    "chunk_sizes": [3, 3, 3, 1],
    "waits_ms": [0, 5000, 5000, 5000]
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

- que el request responda rápido, porque el preview no duerme,
- que la metadata muestre `4` tandas,
- que `waits_ms` refleje las pausas simuladas entre invocaciones.

## Recomendaciones de uso

- arrancar con valores chicos para verificar el comportamiento,
- usar `preview apply_changes` en local antes de probar `run`,
- en `fanout`, pensar el pacing como límite global de la corrida,
- no usar `interval_seconds` altos; el contrato actual permite hasta `10`,
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
