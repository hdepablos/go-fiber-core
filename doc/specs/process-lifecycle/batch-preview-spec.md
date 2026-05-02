---
domain: process-lifecycle
summary: Contrato del endpoint batch-preview, sus modos de selección, apply_changes local y simulación de dispatch_pacing visible en preview.
when_to_read:
  - cambios en batch preview
  - cambios en apply_changes
  - cambios en item_ids o row_numbers
  - cambios en dispatch_pacing visible en preview
code_paths:
  - internal/handlers/process_lifecycle_handler.go
  - internal/services/batchflow/preview_service.go
  - internal/services/batchflow/contracts.go
  - bruno/api/v1/process-lifecycle/
  - bruno/legacy/process-lifecycle/test-batch-process/
related_info:
  - doc/info/process-lifecycle/batch-preview-guide.md
  - doc/info/process-lifecycle/testing-guide.md
  - doc/info/process-lifecycle/dispatch-pacing-guide.md
related_specs:
  - doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md
  - doc/specs/process-lifecycle/batch-fanout-spec.md
status: active
---

# Batch Preview Spec

## Objetivo

Formalizar el contrato de `POST /api/v1/process-lifecycle/batch-preview`, incluyendo el modo de desarrollo local con `apply_changes`.

## Alcance

Aplica a procesos montados sobre `batchflow` que exponen:

- `DataProvider`,
- `BatchPreviewer`,
- `BatchProcessor`,
- `StateStore`.

Complementa la guia humana:

- `doc/info/process-lifecycle/batch-preview-guide.md`
- `doc/info/process-lifecycle/testing-guide.md`
- `doc/info/process-lifecycle/dispatch-pacing-guide.md`

## Contratos de entrada

El request debe aceptar al menos:

- `process_type_id > 0`
- `sede_id >= 0`
- `override_process_version_id >= 0`
- `roadmap >= 0`
- `mode in {prepare, batch, all}`
- `input.id > 0`

Campos opcionales:

- `input.key_redis`
- `input.filters`
- `batch_size`
- `limit`
- `offset`
- `batch_index`
- `item_ids`
- `row_numbers`
- `apply_changes`
- `apply_changes_metadata`

## Reglas de resolucion

- El endpoint debe resolver la version efectiva del proceso con la misma logica de `process lifecycle`.
- El `preview` debe trabajar contra la version realmente resuelta.
- Si `override_process_version_id > 0`, la respuesta debe reflejar la version resuelta o usada efectivamente.

## Reglas de seleccion

- `prepare` debe cargar registros, particionarlos y persistir el estado temporal en Redis.
- `all` debe seleccionar una ventana sobre el conjunto total preparado usando `limit` y `offset`.
- `batch` debe permitir seleccion puntual por:
  - `batch_index`,
  - `item_ids`,
  - `row_numbers`.
- Si `item_ids` o `row_numbers` se informan, deben convertirse en filtros operativos del provider.

## Invariantes de salida

Toda respuesta exitosa debe incluir:

- `process_type_id`
- `resolved_process_version_id`
- `resolved_execution_keys`
- `mode`
- `redis_key`
- `summary`
- `total_batches`

Reglas adicionales:

- `prepare` debe responder con `rendered_count = 0`.
- `all` y `batch` deben responder `items` y `rendered_count` coherentes con la seleccion.
- `selection.total_items` debe representar el universo previo a `limit` y `offset`.

## Reglas de `apply_changes`

- `apply_changes` solo puede habilitarse en entorno local.
- Si `APP_ENV != local`, el handler debe rechazar la solicitud.
- `apply_changes` no es compatible con `mode = prepare`.
- Si `apply_changes = true`, el endpoint debe:
  1. resolver la misma seleccion del preview,
  2. ejecutar `BatchPreviewer.PreviewBatch`,
  3. ejecutar `BatchProcessor.ProcessBatch` sobre exactamente los mismos items.
- Si el step `process_batch` resuelto tiene `dispatch_pacing`, `apply_changes` debe simular esa configuración sobre exactamente los mismos items, sin dormir el request HTTP.

## Invariantes de persistencia

- `apply_changes` debe escribir solo sobre los items seleccionados.
- `apply_changes` no debe ejecutar `ParentLifecycle.Start`.
- `apply_changes` no debe ejecutar `Finalizer`.
- `apply_changes` no debe cerrar el estado global del padre ni marcarlo como finalizado.
- Si el proceso expone un hook opcional de refresco de progreso por lote, `apply_changes` puede invocarlo sobre exactamente los mismos items persistidos para mantener estado derivado o contadores calculables desde `bulk_job_items`.
- Ese refresco opcional no reemplaza `Finalize` y no debe cambiar por si solo la politica de cierre del padre.
- La respuesta debe indicar `applied_changes = true` cuando la persistencia se haya ejecutado.
- Cuando `dispatch_pacing` aplique, la respuesta debe incluir `apply_changes_metadata.dispatch_pacing`.
- `apply_changes_metadata.dispatch_pacing` debe informar al menos:
  - `enabled`
  - `messages_per_interval`
  - `interval_seconds`
  - `mode`
  - `chunk_count`
  - `chunk_sizes`
  - `waits_ms`
  - `simulated`

## Errores esperados

- `process_type_id inválido`
- `id inválido`
- `preview incompleto para process_type`
- `preview apply_changes no disponible para process_type`
- `apply_changes no es compatible con mode "prepare"`
- `apply_changes solo está disponible cuando APP_ENV=local`
- errores de resolución de versión del lifecycle
- errores del provider, del state store o del processor

## Acceptance Criteria

- Un proceso `batchflow` con provider completo permite `prepare`, `all` y `batch`.
- El mismo conjunto de items renderizado por preview puede persistirse con `apply_changes=true`.
- En local, un request con `item_ids` y `apply_changes=true` modifica solo esos registros.
- En local, un request con `item_ids` y `apply_changes=true` refleja `dispatch_pacing` en `apply_changes_metadata` cuando el step lo define.
- En local, si el proceso publica un refresco opcional de progreso por lote, `apply_changes` puede actualizar agregados del padre sin ejecutar `Start` ni `Finalize`.
- En local, `apply_changes` con `dispatch_pacing` no debe depender del tiempo real transcurrido para completar la respuesta.
- En entornos no locales, el mismo request se rechaza.
- La colección Bruno debe exponer ejemplos de:
  - `prepare`
  - `all`
  - `batch_index`
  - `item_ids`
  - `row_numbers`
  - `item_ids + apply_changes`
  - `item_ids + apply_changes + dispatch_pacing`
  - `run` filtrado

## Trazabilidad

- Codigo:
  - `internal/handlers/process_lifecycle_handler.go`
  - `internal/services/batchflow/contracts.go`
  - `internal/services/batchflow/preview_service.go`
  - `internal/services/batchprocess/punitorios/provider.go`
- Bruno:
  - `bruno/legacy/process-lifecycle/test-batch-process/`
- Documentacion humana:
  - `doc/info/process-lifecycle/batch-preview-guide.md`
  - `doc/info/process-lifecycle/testing-guide.md`
  - `doc/info/process-lifecycle/dispatch-pacing-guide.md`
