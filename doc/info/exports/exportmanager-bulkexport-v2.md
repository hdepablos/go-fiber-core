# ExportManager + BulkExportV2

## Objetivo

`exportmanager` es una capa reusable para procesos de generación de archivos.

La idea es que el framework resuelva la parte técnica:

- control del flujo
- cambio de estados
- batching
- persistencia temporal en Redis
- adapters compartidos de Redis
- adapters compartidos de S3
- generación de partes en S3
- merge final
- registro de outputs

Y que el developer del caso concreto solo implemente la lógica funcional del archivo:

- `DataProvider`
- `HeaderBuilder`
- `BodyBuilder`
- `FooterBuilder`
- `ParentLifecycle`
- `OutputRegistrar`

El caso concreto montado actualmente sobre este framework es:

- `internal/services/test/bulkexportV2`
- `internal/services/generar_archivo_banco_galicia`

## Infraestructura compartida

Los adapters técnicos reutilizables viven ahora dentro de `exportmanager`.

Implementaciones:

- [redis_cache.go](file:///private/var/www/go-fiber-core/internal/services/exportmanager/redis_cache.go)
- [s3_store.go](file:///private/var/www/go-fiber-core/internal/services/exportmanager/s3_store.go)

Eso evita duplicar en cada proceso:

- `infrastructure_redis.go`
- `infrastructure_s3.go`

## Principios del flujo

Este proceso es:

- **async**
- **secuencial por lote**
- **aislado por corrida**

Eso significa:

- el request HTTP no genera todo el archivo inline
- el procesamiento de lotes se ejecuta por cola
- los lotes no se disparan todos en paralelo
- el siguiente lote se ejecuta cuando termina el anterior
- cada corrida tiene su propio `key_redis`

## Input mínimo

El input mínimo de negocio para arrancar una corrida es:

```json
{
  "id": 34
}
```

- `id` representa el identificador padre del proceso de negocio
- no es obligatorio enviar `key_redis`

En procesos basados en `bulk_jobs`, ese `id` corresponde al `bulk_job_id`.

Ejemplo:

```json
{
  "id": 2
}
```

Interpretación:

- `id = 2`
- `bulk_job_id = 2`

El framework genera automáticamente una clave única por corrida, por ejemplo:

```text
run-7bb145eb-a1c2-4373-adb5-ef49e3cee773
```

## Significado de `key_redis`

`key_redis` identifica una ejecución específica del proceso.

No representa el padre de negocio.

Su función es:

- aislar una corrida de otra
- evitar que variables temporales de distintos procesos se mezclen
- permitir compartir datos calculados entre `data`, `header`, `body`, `footer` y `end`

Ejemplo:

- el `header` calcula `total_amount`
- guarda ese valor bajo la corrida actual
- el `footer` lo vuelve a leer usando el mismo `key_redis`
- así evita consultar otra vez la base de datos

## Runtime Shared State

El framework expone un runtime context por corrida.

Contrato principal:

- [contracts.go](file:///private/var/www/go-fiber-core/internal/services/exportmanager/contracts.go)
- [runtime.go](file:///private/var/www/go-fiber-core/internal/services/exportmanager/runtime.go)

El runtime context entrega:

- `ExecutionContext.Input`
- `ExecutionContext.Summary`
- `ExecutionContext.Runtime`

`ExecutionContext.Runtime` expone:

- `Set(ctx, key, value)`
- `Get(ctx, key, dest)`
- `Delete(ctx, key)`

La composición real de keys runtime es:

```text
{key_redis}:runtime:{key}
```

Ejemplo:

```text
run-7bb145eb-a1c2-4373-adb5-ef49e3cee773:runtime:total_amount
```

## Contratos del framework

### `DataProvider`

Responsabilidad:

- consultar la data fuente
- aplicar filtros
- partir la información en lotes
- construir el `Summary`
- guardar variables runtime si hace falta

Contrato:

- `LoadBatches(ctx, execCtx, batchSize)`

### `HeaderBuilder`

Responsabilidad:

- construir una o varias líneas iniciales
- usar `id`, `key_redis`, `summary` y runtime compartido
- guardar variables en Redis si hace falta

Contrato:

- `BuildHeader(ctx, execCtx)`

### `BodyBuilder`

Responsabilidad:

- transformar cada registro en una o varias líneas del archivo

Contrato:

- `BuildBodyLines(ctx, execCtx, item)`

### `FooterBuilder`

Responsabilidad:

- construir una o varias líneas finales
- recuperar variables ya calculadas desde runtime

Contrato:

- `BuildFooter(ctx, execCtx)`

### `ParentLifecycle`

Responsabilidad:

- controlar el estado del registro padre

Métodos:

- `Start`
- `End`
- `Fail`

Uso esperado:

- `Start` cambia a `PROCESSING`
- `End` cambia a `PROCESSED`
- `Fail` cambia a `ERROR_PROCESS`

Esto permite que el framework no quede acoplado a `bulk_jobs`.

Pero cuando el proceso sí usa `bulk_jobs`, el scaffold puede dejar una implementación funcional usando:

- `bulk_jobs`
- `bulk_job_items`
- `bulk_job_outputs`

### `OutputRegistrar`

Responsabilidad:

- registrar el resultado final del archivo generado

Ejemplo actual:

- insertar en `bulk_job_outputs`

## Flujo del manager

Implementación principal:

- [manager.go](file:///private/var/www/go-fiber-core/internal/services/exportmanager/manager.go)

### 1. `Start`

Acciones:

- valida `id`
- genera `key_redis` si no vino
- crea el `ExecutionContext`
- invoca `ParentLifecycle.Start`
- invoca `DataProvider.LoadBatches`
- guarda batches y summary en Redis
- devuelve:
  - `key_redis`
  - `batches_list_key`
  - `parts_list_key`
  - `total_batches`

### 2. `ProcessBatch`

Acciones:

- carga `Summary` desde Redis
- carga el batch actual desde Redis
- si es el primer lote, construye `header`
- construye el `body`
- si es el último lote, construye `footer`
- genera la parte CSV y la sube a S3
- guarda la referencia de la parte en Redis
- devuelve:
  - `batch_index` siguiente
  - `is_last_batch`

### 3. `Finalize`

Acciones:

- toma todas las partes guardadas en Redis
- hace merge multipart en S3
- genera el archivo final
- elimina las partes temporales del `run-...` en S3
- limpia el estado temporal de la corrida en Redis
- registra el output final
- marca el proceso como `PROCESSED`

## Keys principales en Redis

El framework usa estas keys base por corrida:

- `{key_redis}:summary`
- `{key_redis}:batches`
- `{key_redis}:parts`
- `{key_redis}:batch:000000`
- `{key_redis}:batch:000001`
- `{key_redis}:runtime:{variable}`
- `{key_redis}:runtime_keys`

### Función de cada una

- `summary`
  - guarda `Summary` y metadata general de la corrida
- `batches`
  - lista de referencias a los lotes
- `parts`
  - lista de `partKey` generados en S3
- `batch:NNNNNN`
  - contenido del lote específico
- `runtime:{variable}`
  - valores compartidos entre servicios del mismo proceso
- `runtime_keys`
  - registro interno de variables runtime para poder limpiarlas al final de la corrida

## Caso concreto `bulkexportV2`

Carpeta:

- [bulkexportV2](file:///private/var/www/go-fiber-core/internal/services/test/bulkexportV2)

### Qué implementa hoy

- `BulkJobDataProvider`
  - lee `bulk_job_items`
  - calcula `total_records`
  - calcula `total_amount`
- `HardcodedHeaderBuilder`
  - genera el header fijo actual
- `JSONBodyBuilder`
  - transforma cada item JSON en línea CSV
- `EmptyFooterBuilder`
  - por ahora no genera líneas, pero ya puede leer runtime
- `BulkJobLifecycle`
  - valida `IMPORTED`
  - cambia a `PROCESSING`
  - cambia a `PROCESSED`
  - maneja `ERROR_PROCESS`
- `BulkJobOutputRegistrar`
  - inserta el resultado final en `bulk_job_outputs`

## Generador de scaffold

Comando:

- [main.go](file:///private/var/www/go-fiber-core/cmd/tools/export-manager-scaffold/main.go)

Target de `make`:

- [Makefile](file:///private/var/www/go-fiber-core/Makefile)

### Modo genérico

Si no se envía `bulk_job_id`, el scaffold:

- genera el módulo bajo `internal/services/<service-slug>`
- deja `DataProvider`, `ParentLifecycle` y `OutputRegistrar` con `TODOs`
- deja footer por defecto con la línea `footer`
- documenta cómo eliminar ese footer si no aplica

### Modo `bulk_jobs`

Si se envía:

```bash
make create-export-manager process_name="generar archivo banco galicia" file="exports/bank/galicia/manager-galicia" bulk_job_id=2
```

el scaffold queda funcional sobre `bulk_jobs`.

Eso significa:

- `DataProvider` consulta `bulk_job_items`
- `ParentLifecycle` valida y actualiza `bulk_jobs`
- `OutputRegistrar` inserta en `bulk_job_outputs`
- el request Bruno generado usa `id = 2`

### Inputs del generador

- `process_name`
- `file`
- `service_slug` opcional
- `batch_size` opcional, default `5000`
- `part_prefix` opcional
- `redis_ttl_hours` opcional, default `24`
- `bulk_job_id` opcional

### Derivaciones automáticas

- `label = process_name`
- `next_step = bulk/export/<service-slug>/finalize`
- execution keys:
  - `bulk/export/<service-slug>/start`
  - `bulk/export/<service-slug>/process_batch`
  - `bulk/export/<service-slug>/finalize`

## Atomicidad en Redis

El framework usa Redis de forma consistente, pero no como una transacción completa de punta a punta.

### Sí es atómico

- cada operación individual:
  - `SET`
  - `GET`
  - `RPUSH`
  - `DEL`

### No es transaccional completo

- la secuencia completa de inicialización, runtime y cleanup usa varias operaciones separadas
- hoy no se usa `MULTI/EXEC`
- hoy no se usan scripts Lua

Conclusión:

- atomicidad por comando: sí
- atomicidad total del flujo completo: no

## Seeder del proceso v2

Seeder:

- [bulk_export_generate_file_v2_seeder.go](file:///private/var/www/go-fiber-core/internal/database/seeders/bulk_export_generate_file_v2_seeder.go)

Steps configurados:

1. `bulk/export/v2/start`
2. `bulk/export/v2/process_batch`
3. `bulk/export/v2/finalize`

## Configuración del step de procesamiento

```json
{
  "execution_policy": {
    "mode": "ASYNC",
    "label": "generar archivo v2",
    "auto_invoke": {
      "enabled": true,
      "cursor_field": "batch_index",
      "stop_condition": "is_last_batch"
    },
    "next_step": "bulk/export/v2/finalize"
  }
}
```

### Significado

- `mode: ASYNC`
  - ejecuta el step por cola
- `auto_invoke.enabled: true`
  - reencola automáticamente el step de batch
- `cursor_field: batch_index`
  - indica cuál es el siguiente lote
- `stop_condition: is_last_batch`
  - indica cuándo el loop se detiene
- `next_step`
  - step que corre cuando ya no hay más lotes

## Configuración del archivo final

El step final usa:

```json
{
  "file": "exports/bank/colombia/manager-colombia"
}
```

Resultado esperado:

```text
exports/bank/colombia/manager-colombia-34.csv
```

## Cleanup de temporales

Cuando el proceso termina correctamente, el framework elimina los temporales de la corrida.

### En S3

- elimina cada parte parcial generada durante `process_batch`
- ejemplo:

```text
exports/bulk_jobs/v2/run-87f6a052-cc6b-4d4c-9fa6-8413ceda523d/part-000000.csv
exports/bulk_jobs/v2/run-87f6a052-cc6b-4d4c-9fa6-8413ceda523d/part-000001.csv
```

- el archivo final no se elimina

### En Redis

- elimina:
  - `{key_redis}:summary`
  - `{key_redis}:batches`
  - `{key_redis}:parts`
  - `{key_redis}:batch:*`
  - `{key_redis}:runtime:*`
  - `{key_redis}:runtime_keys`

### Resultado esperado

- el archivo final queda persistido
- los temporales de la corrida desaparecen
- no quedan restos del `run-...` para que una corrida nueva no se mezcle con una anterior

## Ejemplo práctico de sharing por `key_redis`

Caso:

1. `DataProvider` calcula `total_amount`
2. guarda `total_amount` en runtime
3. `HeaderBuilder` puede usar ese valor
4. `FooterBuilder` puede volver a usar ese mismo valor
5. no hace falta volver a consultar DB

Ejemplo conceptual:

```text
key_redis = run-7bb145eb-a1c2-4373-adb5-ef49e3cee773
runtime key = run-7bb145eb-a1c2-4373-adb5-ef49e3cee773:runtime:total_amount
```

## Request Bruno recomendado

El request mínimo para el flujo v2 puede usar solo `id`:

```json
{
  "process_type_id": 13,
  "sede_id": 0,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "id": 34
  }
}
```

El `key_redis` se genera automáticamente en `start`.

En un proceso basado en `bulk_jobs`, ese `id` es el `bulk_job_id`.
