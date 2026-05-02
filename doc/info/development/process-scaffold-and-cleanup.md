# Process Scaffold And Cleanup

## Objetivo

Documentar el flujo actual para:

- crear procesos nuevos basados en scaffold,
- probarlos desde Bruno sin crear carpetas específicas por proceso,
- y eliminarlos del código cuando ya no se necesiten.

Este documento describe el uso humano y operativo.
Las reglas verificables viven en `doc/specs/platform/process-scaffold-cleanup-spec.md`.

## Alcance

Aplica a dos familias de procesos scaffold:

- `exportmanager`
- `batch-process` basado en `batchflow`

## Comandos principales

### Listar scaffolds disponibles

```bash
make list-scaffolds
```

Uso típico:

- descubrir los scaffolds vigentes del repositorio,
- detectar generadores reusables cercanos como `create-step` y `create-command`,
- copiar el comando base correcto,
- identificar opciones importantes como `force=true`,
- identificar variantes técnicas relevantes del scaffold,
- identificar qué genera cada scaffold,
- y recordar el cleanup o comandos relacionados.

### Crear un export manager

```bash
make create-export-manager process_name="generar archivo x" service_slug="generar_archivo_x" file="exports/x/y"
```

Modo `bulk_jobs`:

```bash
make create-export-manager process_name="generar archivo x" service_slug="generar_archivo_x" file="exports/x/y" bulk_job_id=2
```

Si el scaffold ya existe y se quiere regenerar sobrescribiendo archivos generados:

```bash
make create-export-manager process_name="generar archivo x" service_slug="generar_archivo_x" file="exports/x/y" force=true
```

Uso típico:

- genera archivos del servicio,
- registra imports necesarios,
- cablea `runtimebootstrap` cuando aplica,
- registra el seeder,
- genera request Bruno dedicado para `run`,
- crea documentación humana base del export,
- deja `ParentLifecycle.Fail(...)` en el template base,
- registra preview y manager por `execution_key`,
- deja `BodyBuilder.renderItem(...)` como punto de extensión `item-oriented`,
- genera por defecto un layout funcional con `header`, `body`, `footer` y `layout_helpers.go`,
- ese layout base exporta CSV con `;`, columnas históricas y cálculo de `new_importe`,
- resuelve AWS/S3 dentro del wiring runtime del export sin depender de un `s3Client` compartido predeclarado,
- hace que preview y run reutilicen el mismo camino de render por registro,
- y deja el proceso listo para cancelación operativa y auto-cancel sin editar manualmente `sqs-consumer`.

Semántica del modo `bulk_jobs`:

- `input.id = bulk_job_id`
- `DataProvider` consulta `bulk_job_items`
- `ParentLifecycle` opera sobre `bulk_jobs`
- `OutputRegistrar` registra en `bulk_job_outputs`

Notas:

- `service_slug` es opcional
- si no se envía, el scaffold lo deriva automáticamente desde `process_name`
- el scaffold de export no implementa variante `batch-oriented`
- la personalización recomendada del archivo vive en `BodyBuilder.renderItem(...)` y `layout/layout_helpers.go`

### Crear un batch process

```bash
make create-batch-process process_name="procesar x" service_slug="procesar_x"
```

Modo `bulk_jobs`:

```bash
make create-batch-process process_name="procesar x" service_slug="procesar_x" mode=bulk_jobs
```

Estrategia `batch-oriented`:

```bash
make create-batch-process process_name="procesar x" service_slug="procesar_x" type_process=batch-oriented
```

Si el scaffold ya existe y se quiere regenerar sobrescribiendo archivos generados:

```bash
make create-batch-process process_name="procesar x" service_slug="procesar_x" force=true
```

Uso típico:

- genera el servicio batch,
- registra imports necesarios,
- genera el seeder base secuencial,
- genera el seeder adicional `_fanout`,
- cablea `runtimebootstrap`,
- deja `ParentLifecycle.Fail(...)` en el template base,
- registra el manager por `execution_key`,
- y deja el proceso listo para probarse con la carpeta genérica de Bruno.

Modos del scaffold:

- `generic`: modo por defecto; deja `DataProvider`, `Processor`, `ParentLifecycle` y `Finalizer` comentados para adaptar otra tabla padre/hija.
- `bulk_jobs`: genera el scaffold funcional base sobre `bulk_jobs` y `bulk_job_items` con la misma lógica operativa inicial usada en `punitorios`.

Estrategias del processor:

- `item-oriented`: default; el developer implementa `processItemOriented(...)` y luego persiste el resultado del lote en bloque.
- `batch-oriented`: el developer implementa `processBatchOriented(...)` recibiendo el lote completo para agrupar ids y preparar updates masivos.

Semántica `bulk_jobs` del modo funcional:

- `input.id = bulk_job_id`
- `DataProvider` consulta `bulk_job_items`
- `ParentLifecycle` opera sobre `bulk_jobs`
- `Finalize` calcula progreso y pendientes desde `bulk_job_items`
- `ProcessBatch` actualiza `status_code`, `last_detail_message` y mensajes operativos sobre `bulk_job_items`

Puntos del flujo que el scaffold deja documentados en código:

- fuente de datos: `data/provider.go` en `LoadBatches()`
- cambio de status del padre al iniciar: `lifecycle/parent.go` en `Start()`
- recepción del lote a procesar: `steps/process_batch.go` en `Execute()`
- procesamiento lote a lote: `processor/processor.go` en `ProcessBatch()`
- extensión item-oriented: `processor/processor.go` en `processItemOriented(...)`
- extensión batch-oriented: `processor/processor.go` en `processBatchOriented(...)`
- persistencia del status del detalle: `processor/processor.go` en `updateBatchItemStatuses()`
- persistencia de mensajes detallados por item: `processor/processor.go` dentro de `ProcessBatch()`
- cálculo del cierre del proceso: `lifecycle/finalizer.go` en `Finalize()`
- cambio de status final del padre: `lifecycle/parent.go` en `End()`
- cambio de status por error: `lifecycle/parent.go` en `Fail()`

Lectura recomendada del flujo cuando se genera un batch process:

1. `data/provider.go`
   Define la fuente de datos, el filtro, el corte en lotes y el `Summary`.
2. `lifecycle/parent.go`
   Controla el status del padre al iniciar, finalizar o fallar.
3. `steps/process_batch.go`
   Entrega el lote actual al manager para que invoque el processor.
4. `processor/processor.go`
   Procesa el lote, delega a `processItemOriented(...)` o `processBatchOriented(...)`, actualiza el detalle y persiste mensajes.
5. `lifecycle/finalizer.go`
   Resume el resultado global y prepara la metadata final que consume `End()`.

### Eliminar un proceso scaffold

```bash
make delete-process kind=batch-process service_slug=punitorios
```

También soporta export:

```bash
make delete-process kind=export service_slug=generar_archivo_banco_galicia
```

Modo simulación:

```bash
make delete-process kind=batch-process service_slug=punitorios dry_run=true
```

## Filosofía actual de Bruno

Para `batch-process` ya no se crean carpetas específicas por proceso.
La convención operativa actual es usar una sola carpeta genérica:

- `bruno/legacy/process-lifecycle/test-batch-process/`

Eso permite cambiar de proceso solo ajustando variables del request:

- `process_type_id_batch_current`
- `process_version_id_batch_current`
- `sede_id_batch_current`
- `bulk_job_id_batch_current`
- `redis_key_batch_current`

## Requests disponibles en `test-batch-process`

La carpeta genérica debe cubrir todas las variantes necesarias del flujo batch:

- `preview - prepare`
- `preview - all`
- `preview - batch - batch_index`
- `preview - batch - item_ids`
- `preview - batch - row_numbers`
- `preview - batch - item_ids - apply_changes`
- `run - item_ids`
- `run - row_numbers`

La idea es que un developer no necesite una carpeta Bruno nueva por cada proceso batch.

## Qué crea cada scaffold

### Export manager

El scaffold de export crea normalmente:

- `provider.go`
- carpeta del servicio bajo `internal/services/exports/<service_slug>/`
- `runtime/provider_context.go`
- `data/provider.go`
- `layout/header_builder.go`
- `layout/body_builder.go`
- `layout/footer_builder.go`
- `layout/layout_helpers.go`
- `lifecycle/parent.go`
- `lifecycle/output_registrar.go`
- `steps/start.go`
- `steps/process_batch.go`
- `steps/finalize.go`
- `steps/input.go`
- `steps/failure.go`
- seeder del proceso
- request Bruno `RunProc -> <service_slug>.bru`
- documentación humana base del export
- wiring en `runtimebootstrap` cuando el proceso corre en runtime compartido
- registro de preview y manager por `execution_key`

### Batch process

El scaffold de batch crea normalmente:

- `provider.go`
- carpeta del servicio bajo `internal/services/batchprocess/<service_slug>/`
- `runtime/provider_context.go`
- `data/provider.go`
- `processor/processor.go`
- `lifecycle/parent.go`
- `lifecycle/finalizer.go`
- `steps/start.go`
- `steps/dispatch_shards.go`
- `steps/process_batch.go`
- `steps/finalize.go`
- `steps/input.go`
- `steps/failure.go`
- `steps/helpers.go`
- seeder base del proceso: `batch_process_<service_slug>`
- seeder fanout del proceso: `batch_process_<service_slug>_fanout`
- registro del manager por `execution_key`

No crea una carpeta Bruno nueva por proceso.

Variantes operativas relevantes:

- `generic`: modo default para adaptar a otra tabla padre/hija.
- `bulk_jobs`: modo funcional tipo `punitorios` sobre `bulk_jobs/bulk_job_items`.
- `sequential`: base generada automáticamente.
- `fanout`: companion `_fanout` generado automáticamente.
- `dispatch_pacing`: variante técnica opcional generable desde el scaffold con `pacing=true`.
- `clone-process-version`: operación hija de `batch-process` para clonar una `process_version` existente y opcionalmente agregar `dispatch_pacing`.
- `add-process-pacing`: operación hija de `batch-process`, atajo de `clone-process-version` con pacing activado.

## Catálogo operativo recomendado

Cuando no recuerdes el nombre exacto del scaffold o quieras confirmar su alcance, usar:

```bash
make list-scaffolds
```

Si la necesidad es encontrar utilidades operativas más amplias del repositorio y no solo scaffolds, usar:

```bash
make list-tools
```

Este comando debe resumir:

- el nombre del scaffold,
- el comando de creación,
- las opciones importantes como `force=true`,
- los modos operativos relevantes como `bulk_jobs` cuando estén contemplados,
- las variantes técnicas relevantes del scaffold,
- los artefactos principales que genera,
- el cleanup asociado cuando aplique.

Cobertura mínima esperada hoy:

- `service-step`
- `batch-process`
- `export-manager`
- `external-api-config`
- `external-adapter`
- `external-integration`
- `cli-command`

## Modelo recomendado para versiones batch

 La convención actual para un batch nuevo es:

- un solo `process_type` por negocio, por ejemplo `mi_proceso`
- una versión base secuencial
- una versión adicional fanout

Ejemplo:

- `process_type`: `mi_proceso`
- seeder base: `batch_process_mi_proceso`
- seeder fanout: `batch_process_mi_proceso_fanout`
- label base recomendado: `mi_proceso`
- label fanout recomendado: `mi_proceso fanout`

La idea es no mezclar:

- nombre del proceso de negocio
- nombre del perfil técnico de ejecución

## Qué elimina `delete-process`

### Para `batch-process`

El cleanup elimina, cuando existen:

- carpeta `internal/services/batchprocess/<service_slug>/`
- carpeta legacy `internal/services/<service_slug>/`
- archivo de seeder del proceso
- archivo de seeder fanout del proceso
- import en `cmd/api/main.go`
- import en `cmd/sqs-consumer/main.go`
- wiring en `internal/runtimebootstrap/bootstrap.go`
- registro del seeder en `internal/database/seeders/seed_service.go`
- request `RunProc -> <service_slug>.bru`
- carpetas específicas viejas de Bruno si existieran

### Para `export`

El cleanup elimina, cuando existen:

- carpeta `internal/services/exports/<service_slug>/`
- archivo de seeder del proceso
- import en `cmd/api/main.go`
- import en `cmd/sqs-consumer/main.go`
- wiring en `internal/runtimebootstrap/bootstrap.go`
- registro del seeder en `internal/database/seeders/seed_service.go`
- documentación `doc/info/exportmanager_<service_slug>.md`
- request Bruno dedicado del export
- normalización final de `bootstrap.go` para no dejar wiring huérfano si se elimina el último export

## Flujo recomendado de trabajo

### Caso 1: batch process nuevo

1. crear el scaffold con `make create-batch-process`
   Ejemplos:
   - `make create-batch-process process_name="mi proceso" service_slug="mi_proceso"`
   - `make create-batch-process process_name="mi proceso" service_slug="mi_proceso" mode=bulk_jobs`
   - `make create-batch-process process_name="mi proceso" service_slug="mi_proceso" pacing=true pacing_messages=100 pacing_interval=2`
2. si hace falta confirmar parámetros o cleanup, correr `make list-scaffolds`
2. ejecutar el seeder base secuencial
3. ejecutar el seeder fanout si se quiere comparar rendimiento
3. abrir `test-batch-process`
4. ajustar `process_type_id`, `process_version_id`, `sede_id`, `bulk_job_id`
5. probar `prepare`
6. probar `all` o `batch`
7. probar `apply_changes=true` sobre pocos registros
8. si cierra, probar `run` filtrado

### Caso 2: export nuevo

1. crear el scaffold con `make create-export-manager`
2. si hace falta confirmar el comando o el cleanup, correr `make list-scaffolds`
2. ajustar `layout`, provider y lifecycle
3. ejecutar el seeder
4. usar el request Bruno dedicado del proceso

### Caso 3: eliminación de un proceso

1. correr `make delete-process ... dry_run=true`
2. revisar el alcance
3. ejecutar el comando real
4. revisar si quedan referencias de negocio o datos seed ya aplicados en base

## Limitaciones conocidas

- `delete-process` limpia código y archivos del repositorio, no borra datos ya seedados en la base.
- Los requests de `test-batch-process` son genéricos; dependen de que el developer actualice las variables correctas.
- Si un proceso tiene wiring manual fuera del patrón scaffold, el cleanup puede no capturarlo y requerir revisión manual.
- `list-scaffolds` es un catálogo operativo, no una fuente dinámica; debe mantenerse sincronizado cuando aparezcan scaffolds nuevos.
- `list-scaffolds` también debe mantenerse sincronizado cuando cambien opciones críticas como `force=true` o aparezcan variantes técnicas relevantes como `dispatch_pacing`, `pacing_messages` o `pacing_interval`.

## Troubleshooting

### El proceso batch compila pero no aparece en preview

Revisar:

- imports en `cmd/api/main.go`
- imports en `cmd/sqs-consumer/main.go`
- wiring en `internal/runtimebootstrap/bootstrap.go`
- registro del seeder
- `process_type_id` y `override_process_version_id` usados en Bruno

### `create-batch-process force=true` no sobrescribe

Revisar que el comando Makefile ejecute el flag booleano como `-with-bruno=false`.
El scaffold batch usa flags booleanos de Go, por lo que la forma `-with-bruno false` puede cortar el parseo antes de llegar a `-force`.

### El request Bruno apunta a otro proceso

Revisar las variables del request en `test-batch-process`.
La carpeta es genérica y no representa un proceso fijo.

### El cleanup no borra algo esperado

Usar primero `dry_run=true` y luego revisar si el proceso tenía personalizaciones fuera del patrón scaffold.

## Trazabilidad

- `Makefile`
- `cmd/tools/export-manager-scaffold/main.go`
- `cmd/tools/batch-process-scaffold/main.go`
- `cmd/tools/process-cleanup/main.go`
- `doc/info/platform/makefile-guide.md`
- `doc/info/process-lifecycle/batch-preview-guide.md`
- `doc/specs/platform/process-scaffold-cleanup-spec.md`
