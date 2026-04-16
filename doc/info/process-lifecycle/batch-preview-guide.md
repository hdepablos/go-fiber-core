# Batch Preview Guide

## Objetivo

Documentar como usar `POST /api/v1/process-lifecycle/batch-preview` para:

- preparar una corrida,
- inspeccionar registros concretos,
- aplicar la logica real del batch sobre una seleccion acotada en desarrollo local.

Este documento describe el uso humano y operativo.
Las reglas verificables viven en `doc/specs/process-lifecycle/batch-preview-spec.md`.

## Contexto

El motor `batchflow` expone un preview reutilizable para procesos como `procesar lote generico` y `punitorios`.
El preview trabaja sobre el mismo `DataProvider` y el mismo `Processor` del flujo real.

En local ahora existen dos comportamientos:

- preview puro: simula el resultado y no escribe en base.
- preview con cambios: simula la respuesta y ademas ejecuta la escritura real sobre la seleccion pedida.

## Endpoint

- URL: `POST /api/v1/process-lifecycle/batch-preview`
- Request canónico base: `bruno/api/v1/process-lifecycle/post-batch-preview.bru`
- Requests históricos y variantes operativas:
  - `bruno/legacy/process-lifecycle/test-batch-process/`

## Campos principales del request

- `process_type_id`: proceso que se quiere probar.
- `override_process_version_id`: version exacta a usar cuando se quiere fijar una corrida.
- `sede_id`: sede a usar en la resolución del lifecycle.
- `mode`: `prepare`, `all` o `batch`.
- `input.id`: id del padre de negocio, en estos ejemplos `bulk_job_id`.
- `input.key_redis`: key de sesion del preview.
- `item_ids`: seleccion directa por ids.
- `row_numbers`: seleccion por filas.
- `batch_index`: seleccion de un batch concreto.
- `limit` y `offset`: ventana sobre la seleccion.
- `apply_changes`: si es `true`, aplica el procesamiento real sobre la seleccion renderizada.

## Modos

### `prepare`

Prepara los registros, los parte en batches y guarda el estado temporal en Redis.
No devuelve items y no escribe en base.

Se usa para:

- conocer `summary.total_records`,
- conocer `total_batches`,
- fijar una sesion reutilizable via `key_redis`.

### `all`

Recorre todos los items preparados y devuelve una ventana usando `limit` y `offset`.
Sirve para inspeccion general del lote.

### `batch`

Renderiza una seleccion puntual.
Puede usarse con:

- `batch_index`,
- `item_ids`,
- `row_numbers`.

## `apply_changes` en local

Si `apply_changes` es `true`, el endpoint:

1. resuelve exactamente la misma seleccion del preview,
2. devuelve la respuesta de preview para inspeccion,
3. ejecuta el `ProcessBatch` real sobre esos items.

Esto permite desarrollar una logica batch y verificar en el mismo request:

- que items entraron,
- que status calcula,
- que mensajes generaria,
- y como queda persistido realmente.

### Alcance de `apply_changes`

- actualiza solo los registros seleccionados,
- escribe en tablas del procesamiento real, por ejemplo `bulk_job_items` y `bulk_job_item_messages`,
- no ejecuta el lifecycle completo del padre,
- no corre `Start` ni `Finalize`,
- no debe usarse para cerrar el estado global de un `bulk_job`.

## Casos recomendados

### Variables compartidas en Bruno

La carpeta `test-batch-process` es genérica.
Antes de ejecutar un request conviene ajustar sus variables:

- `process_type_id_batch_current`
- `process_version_id_batch_current`
- `sede_id_batch_current`
- `bulk_job_id_batch_current`
- `redis_key_batch_current`

La carpeta no representa un proceso fijo.
Representa un set reusable de requests para cualquier proceso batch.

### Caso 1: preparar una corrida

Usa:

- `bruno/legacy/process-lifecycle/test-batch-process/preview - prepare.bru`

Objetivo:

- saber cuantos registros hay,
- cuántos batches se generaron,
- fijar la `key_redis` para la sesion.

### Caso 2: inspeccionar 10 registros

Usa:

- `preview - all`
- `preview - batch - item_ids`
- `preview - batch - row_numbers`
- `preview - batch - batch_index`

Objetivo:

- ver el resultado simulado,
- ajustar el criterio de filtros,
- validar mensajes y status sin escribir.

### Caso 3: aplicar cambios reales a pocos registros

Usa:

- `bruno/legacy/process-lifecycle/test-batch-process/preview - batch - item_ids - apply_changes.bru`

Objetivo:

- probar la logica real sobre una muestra controlada,
- sin disparar el lifecycle completo del proceso.

### Caso 4: correr el flujo real filtrado

Usa:

- `bruno/legacy/process-lifecycle/test-batch-process/run - item_ids.bru`
- `bruno/legacy/process-lifecycle/test-batch-process/run - row_numbers.bru`

Objetivo:

- ejecutar el flujo lifecycle real,
- respetando filtros del `DataProvider`,
- y dejando que el resto del proceso siga su comportamiento operativo normal.

## Diferencia entre preview con cambios y run

- `preview` sin cambios:
  - solo inspecciona.
- `preview` con `apply_changes=true`:
  - inspecciona y persiste la logica real de la seleccion,
  - sin cerrar el proceso padre.
- `run`:
  - ejecuta el lifecycle real del proceso,
  - incluyendo politica async, `auto_invoke` y pasos siguientes.

## Filtros soportados en `bulk_job_items`

Para los procesos actuales montados sobre `bulk_job_items`, el `DataProvider` soporta:

- `status_code`
- `reference_key`
- `row_number`
- `id`
- `bulk_job_id`

Ejemplo:

```json
{
  "input": {
    "id": 1,
    "filters": {
      "status_code": "IMPORTED",
      "id": [1, 2, 3, 4, 5]
    }
  }
}
```

## Recomendacion operativa

En desarrollo local:

1. correr `make watch`,
2. usar `prepare`,
3. inspeccionar con `all` o `batch`,
4. usar `apply_changes=true` sobre pocos registros,
5. si el comportamiento ya cierra, probar `run` filtrado.

## Trazabilidad

- Codigo:
  - `internal/handlers/process_lifecycle_handler.go`
  - `internal/services/batchflow/preview_service.go`
  - `internal/utils/shared_helpers.go`
  - `internal/services/punitorios/processor.go`
- Bruno:
  - `bruno/legacy/process-lifecycle/test-batch-process/`
- Spec relacionada:
  - `doc/specs/process-lifecycle/batch-preview-spec.md`
