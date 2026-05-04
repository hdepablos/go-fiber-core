# ExportManager Batch Guide

## Objetivo

`exportmanager` es la capa reusable del proyecto para procesos batch de generación de archivos.

El framework resuelve la parte técnica:

- control del flujo;
- batching;
- persistencia temporal en Redis;
- generación de partes en S3;
- merge final;
- registro del output;
- cancelación operativa y auto-cancel.

El caso concreto solo implementa la lógica funcional del archivo.

## Componentes que implementa el proceso concreto

- `DataProvider`
- `HeaderBuilder`
- `BodyBuilder`
- `FooterBuilder`
- `ParentLifecycle`
- `OutputRegistrar`

## Contratos del framework

Rutas principales:

- `internal/services/exportmanager/contracts.go`
- `internal/services/exportmanager/manager.go`
- `internal/services/exportmanager/runtime.go`
- `internal/services/exportmanager/redis_cache.go`
- `internal/services/exportmanager/s3_store.go`

### `DataProvider`

Responsabilidad:

- consultar la fuente de datos;
- aplicar filtros;
- partir la información en lotes;
- construir el `Summary`;
- guardar variables runtime cuando haga falta.

Contrato:

- `LoadBatches(ctx, execCtx, batchSize)`

### `HeaderBuilder`

Responsabilidad:

- construir las líneas iniciales del archivo;
- usar `id`, `key_redis`, `summary` y runtime compartido;
- guardar variables runtime si hace falta.

Contrato:

- `BuildHeader(ctx, execCtx)`

### `BodyBuilder`

Responsabilidad:

- transformar cada item en una o varias líneas;
- centralizar la lógica item-oriented del export;
- reutilizar el mismo camino en preview y run.

Contrato:

- `BuildBodyLines(ctx, execCtx, item)`

### `FooterBuilder`

Responsabilidad:

- construir las líneas finales;
- recuperar variables ya calculadas desde runtime.

Contrato:

- `BuildFooter(ctx, execCtx)`

### `ParentLifecycle`

Responsabilidad:

- controlar el estado del registro padre;
- dejar el padre consistente en `Start`, `End` y `Fail`.

### `OutputRegistrar`

Responsabilidad:

- registrar el archivo final generado en la tabla o repositorio correspondiente.

## Flujo del manager

### 1. `Start`

- valida el `id`;
- genera `key_redis` si no vino;
- crea `ExecutionContext`;
- ejecuta `ParentLifecycle.Start`;
- llama `DataProvider.LoadBatches`;
- guarda batches y summary en Redis;
- devuelve `key_redis`, `batches_list_key`, `parts_list_key` y `total_batches`.

### 2. `ProcessBatch`

- carga `Summary` desde Redis;
- carga el batch actual;
- construye `header` si corresponde;
- construye `body`;
- construye `footer` si es el último lote;
- genera la parte temporal y la sube a S3;
- registra la parte en Redis;
- devuelve `batch_index` siguiente e `is_last_batch`.

### 3. `Finalize`

- toma todas las partes registradas;
- hace merge final en S3;
- limpia temporales;
- registra el output final;
- ejecuta `ParentLifecycle.End`.

## Keys principales en Redis

Por corrida, el framework usa conceptualmente:

- `{key_redis}:summary`
- `{key_redis}:batches`
- `{key_redis}:parts`
- `{key_redis}:batch:000000`
- `{key_redis}:batch:000001`
- `{key_redis}:runtime:{variable}`
- `{key_redis}:runtime_keys`

### Qué hace cada una

- `summary`: guarda metadata general de la corrida;
- `batches`: lista de referencias a lotes;
- `parts`: lista de partes generadas en S3;
- `batch:NNNNNN`: contenido de un lote específico;
- `runtime:{variable}`: datos compartidos entre pasos de la misma corrida;
- `runtime_keys`: inventario interno para cleanup.

## Cancelación operativa y auto-cancel

Los exports batch montados sobre `exportmanager` deben:

- registrar preview con `exportmanager.RegisterPreviewProvider(...)`;
- registrar el manager con `exportmanager.RegisterManagedExportManager(...)`;
- registrar sus `execution_key` reales;
- implementar `ParentLifecycle.Fail(...)` si el padre necesita reflejar `ERROR_PROCESS`.

Eso permite:

- cancelación manual;
- auto-cancel por errores repetidos;
- y consistencia del estado del padre sin tocar el consumer por cada export nuevo.

## Caso concreto de referencia

Actualmente el caso de referencia montado sobre `exportmanager` es:

- `internal/services/exports/bcra`

Este caso sirve como ejemplo práctico de:

- `DataProvider`;
- builders de layout;
- lifecycle del padre;
- output registrar;
- wiring completo del manager.

## Scaffold recomendado

Comando:

```bash
make create-export-manager process_name="generar archivo x" service_slug="generar_archivo_x" file="exports/x/y"
```

Modo `bulk_jobs`:

```bash
make create-export-manager process_name="generar archivo x" service_slug="generar_archivo_x" file="exports/x/y" bulk_job_id=2
```

En `bulk_jobs`:

- `input.id = bulk_job_id`;
- `DataProvider` consulta `bulk_job_items`;
- `ParentLifecycle` opera sobre `bulk_jobs`;
- `OutputRegistrar` registra en `bulk_job_outputs`.

## Notas de diseño

- `exportmanager` describe el framework reusable, no una versión histórica del proceso.
- Los pipelines legacy que no usan `exportmanager.Manager` deben documentarse explícitamente como excepción hasta migrarse o eliminarse.
- La documentación de cada export concreto debe vivir en su propio documento humano y reutilizar esta guía como base común.
