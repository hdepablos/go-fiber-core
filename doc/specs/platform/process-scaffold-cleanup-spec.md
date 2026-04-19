---
domain: platform
summary: Contrato de scaffold y cleanup para procesos exportmanager y batch-process, incluyendo convenciones Bruno y catálogos de scaffolds.
when_to_read:
  - cambios en scaffolds de procesos
  - cambios en delete-process
  - cambios en list-scaffolds
  - cambios en convenciones Bruno batch
code_paths:
  - cmd/tools/export-manager-scaffold/
  - cmd/tools/batch-process-scaffold/
  - cmd/tools/process-cleanup/
  - Makefile
  - bruno/legacy/process-lifecycle/test-batch-process/
related_info:
  - doc/info/development/process-scaffold-and-cleanup.md
  - doc/info/platform/makefile-guide.md
related_specs:
  - doc/specs/platform/makefile-automation-spec.md
status: active
---

# Process Scaffold Cleanup Spec

## Objetivo

Formalizar el comportamiento esperado de los comandos de scaffold y cleanup de procesos generados en el repositorio.

## Alcance

Aplica a:

- `make create-export-manager`
- `make create-batch-process`
- `make list-scaffolds`
- `make delete-process`

Aplica tanto a procesos `exportmanager` como a procesos `batch-process`.

## Contratos de entrada

### `create-export-manager`

Debe aceptar al menos:

- `process_name`
- `file`

Opcionales típicos:

- `service_slug`
- `batch_size`
- `part_prefix`
- `redis_ttl_hours`
- `bulk_job_id`

### `create-batch-process`

Debe aceptar al menos:

- `process_name`

Opcionales típicos:

- `service_slug`
- `batch_size`
- `concurrent_batches`
- `parallel_shards`
- `redis_ttl_hours`
- `bulk_job_id`
- `force`

### `delete-process`

Debe aceptar:

- `kind in {batch-process, export}`
- `service_slug`

Opcional:

- `dry_run`

### `list-scaffolds`

No requiere parámetros.
Debe poder ejecutarse sin contexto adicional.

## Reglas de scaffold

### Catálogo de scaffolds

- Debe existir un comando de descubrimiento humano para scaffolds vigentes.
- La referencia actual es `make list-scaffolds`.
- Ese catálogo debe incluir al menos:
  - nombre del scaffold,
  - comando base de creación,
  - opciones importantes como `force=true` cuando existan,
  - variantes técnicas relevantes del scaffold cuando existan,
  - resumen de artefactos generados,
  - cleanup o comandos relacionados cuando aplique.
- Si se agrega un scaffold nuevo o un generador reusable del mismo nivel operativo, el catálogo debe actualizarse.
- La cobertura actual del catálogo debe contemplar al menos:
  - `service-step`
  - `batch-process`
  - `export-manager`
  - `external-api-config`
  - `external-adapter`
  - `external-integration`
  - `cli-command`

### Export manager

El scaffold de export debe generar:

- archivos del servicio export,
- archivo de seeder,
- wiring de imports requerido,
- request Bruno dedicado de `run`,
- documentación humana base del export.

### Batch process

El scaffold de batch debe generar:

- archivos del servicio batch,
- archivo de seeder base secuencial,
- archivo de seeder fanout,
- wiring de imports requerido,
- wiring en `runtimebootstrap`.

El scaffold de batch no debe requerir una carpeta Bruno específica por proceso.
El scaffold debe usar un único `process_type` por negocio y separar el modo técnico a nivel de `process_version`.
- La versión base debe usar como `execution_policy.label` el nombre del proceso de negocio sin sufijo técnico.
- La versión fanout puede usar un label técnico explícito, por ejemplo `<process_name> fanout`.

## Reglas de Bruno para batch

- La carpeta canónica de pruebas batch debe ser `bruno/legacy/process-lifecycle/test-batch-process/`.
- Los requests de esta carpeta deben ser genéricos y parametrizados por variables.
- El cambio de proceso debe resolverse ajustando:
  - `process_type_id`
  - `override_process_version_id`
  - `sede_id`
  - `bulk_job_id`
  - `key_redis`

## Variantes mínimas requeridas en Bruno batch

La colección genérica `test-batch-process` debe exponer ejemplos de:

- `prepare`
- `all`
- `batch_index`
- `item_ids`
- `row_numbers`
- `item_ids + apply_changes`
- `item_ids + apply_changes + dispatch_pacing`
- `run` filtrado por ids
- `run` filtrado por row_numbers

## Reglas de cleanup

### Batch process

El cleanup debe remover, cuando existan:

- carpeta del servicio
- archivo de seeder
- archivo de seeder fanout
- import en `cmd/api/main.go`
- import en `cmd/sqs-consumer/main.go`
- wiring en `internal/runtimebootstrap/bootstrap.go`
- registro del seeder en `internal/database/seeders/seed_service.go`
- request `RunProc -> <service_slug>.bru`
- carpeta Bruno específica vieja del proceso, si existiera

### Export

El cleanup debe remover, cuando existan:

- carpeta del servicio
- archivo de seeder
- import en `cmd/api/main.go`
- import en `cmd/sqs-consumer/main.go`
- registro del seeder en `internal/database/seeders/seed_service.go`
- documentación humana base del export
- request Bruno dedicado del proceso

## Invariantes

- `delete-process dry_run=true` no debe modificar archivos ni borrar paths.
- `delete-process` debe tolerar paths inexistentes sin fallar por ello.
- El cleanup no debe borrar documentación o wiring de otros procesos.
- El scaffold batch no debe generar nuevas carpetas Bruno específicas por proceso.
- El scaffold batch debe producir un seeder base `sequential` y un seeder adicional `_fanout`.
- Los seeders base y fanout deben apuntar al mismo `process_type`.
- `force=true` debe permitir sobrescribir el scaffold existente.
- El scaffold batch debe poder generar `dispatch_pacing` cuando se solicite explícitamente con parámetros del comando.
- Si el scaffold batch genera `dispatch_pacing`, debe reflejarlo en el `config` del step `process_batch` sin requerir edición manual mínima para el caso base.
- Debe existir una herramienta operativa genérica para clonar una `process_version` existente.
- Debe existir una herramienta operativa para clonar una `process_version` existente y agregar `dispatch_pacing` sin regenerar el servicio.
- La herramienta genérica de clonado debe poder, opcionalmente, agregar `dispatch_pacing` cuando se solicite explícitamente.
- Esa herramienta debe preservar la semántica técnica de la versión origen, por ejemplo `sequential` o `fanout`, y modificar solo el `process_batch`.
- En el catálogo humano, esas operaciones de versionado deben poder aparecer como hijas del dominio `batch-process`.
- La documentación humana y normativa debe reflejar el modelo Bruno genérico.
- `make list-scaffolds` debe mantenerse alineado con los scaffolds vigentes del repositorio.

## Limitaciones explícitas

- `delete-process` actúa sobre código y archivos del repositorio, no sobre datos ya sembrados en base.
- Si un proceso fue modificado manualmente fuera del patrón scaffold, el cleanup puede requerir revisión manual posterior.

## Errores esperados

- `kind inválido`
- `service_slug inválido`
- errores de lectura o escritura de archivos del repositorio
- errores de permisos del filesystem

## Acceptance Criteria

- El equipo puede crear un batch process nuevo sin que se genere una carpeta Bruno específica.
- El equipo puede sembrar una versión secuencial y otra fanout del mismo proceso de negocio.
- El equipo puede probar cualquier batch process ajustando variables en `test-batch-process`.
- El equipo puede descubrir los scaffolds vigentes mediante un comando explícito.
- El equipo puede eliminar un proceso scaffold con un único comando `make delete-process`.
- El modo `dry_run` permite inspeccionar el alcance del cleanup antes de ejecutarlo.
- La documentación del Makefile y del flujo batch refleja este comportamiento.

## Trazabilidad

- `Makefile`
- `cmd/tools/export-manager-scaffold/main.go`
- `cmd/tools/batch-process-scaffold/main.go`
- `cmd/tools/process-cleanup/main.go`
- `doc/info/development/process-scaffold-and-cleanup.md`
- `doc/info/platform/makefile-guide.md`
- `doc/info/process-lifecycle/batch-preview-guide.md`
