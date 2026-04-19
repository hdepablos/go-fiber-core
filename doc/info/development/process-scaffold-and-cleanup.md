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
make create-export-manager process_name="generar archivo x" file="exports/x/y"
```

Si el scaffold ya existe y se quiere regenerar sobrescribiendo archivos generados:

```bash
make create-export-manager process_name="generar archivo x" file="exports/x/y" force=true
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
- y deja el proceso listo para cancelación operativa y auto-cancel sin editar manualmente `sqs-consumer`.

### Crear un batch process

```bash
make create-batch-process process_name="procesar x" service_slug="procesar_x"
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
- `steps.go`
- `data_provider.go`
- `layout.go`
- `lifecycle.go`
- seeder del proceso
- request Bruno `RunProc -> <service_slug>.bru`
- documentación humana base del export
- wiring en `runtimebootstrap` cuando el proceso corre en runtime compartido
- registro de preview y manager por `execution_key`

### Batch process

El scaffold de batch crea normalmente:

- `provider.go`
- `steps.go`
- `data_provider.go`
- `processor.go`
- `lifecycle.go`
- seeder base del proceso: `batch_process_<service_slug>`
- seeder fanout del proceso: `batch_process_<service_slug>_fanout`
- registro del manager por `execution_key`

No crea una carpeta Bruno nueva por proceso.

Variantes operativas relevantes:

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

- un solo `process_type` por negocio, por ejemplo `imputations`
- una versión base secuencial
- una versión adicional fanout

Ejemplo:

- `process_type`: `imputations`
- seeder base: `batch_process_imputations`
- seeder fanout: `batch_process_imputations_fanout`
- label base recomendado: `imputations`
- label fanout recomendado: `imputations fanout`

La idea es no mezclar:

- nombre del proceso de negocio
- nombre del perfil técnico de ejecución

## Qué elimina `delete-process`

### Para `batch-process`

El cleanup elimina, cuando existen:

- carpeta `internal/services/<service_slug>/`
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

- carpeta `internal/services/<service_slug>/`
- archivo de seeder del proceso
- import en `cmd/api/main.go`
- import en `cmd/sqs-consumer/main.go`
- wiring en `internal/runtimebootstrap/bootstrap.go`
- registro del seeder en `internal/database/seeders/seed_service.go`
- documentación `doc/info/exportmanager_<service_slug>.md`
- request Bruno dedicado del export

## Flujo recomendado de trabajo

### Caso 1: batch process nuevo

1. crear el scaffold con `make create-batch-process`
   Ejemplos:
   - `make create-batch-process process_name="imputations" service_slug="imputations"`
   - `make create-batch-process process_name="imputations" service_slug="imputations" pacing=true pacing_messages=100 pacing_interval=2`
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
