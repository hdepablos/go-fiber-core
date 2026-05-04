---
domain: process-lifecycle
summary: Contrato operativo para casos ejemplo reproducibles, su creación bajo demanda, seeders asociados y cleanup controlado del código de demo.
when_to_read:
  - cambios en casos ejemplo de process lifecycle
  - nuevos comandos create-example-case o delete-example-case
  - reorganización de servicios demo bajo internal/services/test
  - cambios en requests Bruno de ejemplos reproducibles
code_paths:
  - cmd/tools/example-case-manager/
  - internal/services/examplesregistry/
  - internal/services/test/
  - Makefile
  - bruno/legacy/process-lifecycle/example-cases/
related_info:
  - doc/info/process-lifecycle/example-cases.md
  - doc/info/platform/makefile-guide.md
related_specs:
  - doc/specs/platform/makefile-automation-spec.md
  - doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md
status: active
---

# Example Cases Spec

## Objetivo

Formalizar cómo se crean, activan, prueban y eliminan los casos ejemplo reproducibles del dominio `process lifecycle`.

## Alcance

Aplica a:

- `make list-example-cases`
- `make create-example-case`
- `make seed-example-case`
- `make recreate-example-case`
- `make delete-example-case`
- `cmd/tools/example-case-manager/`
- `internal/services/examplesregistry/`

## Reglas

### 1. Separación entre seed y código ejemplo

- `seed-one` sigue siendo la fuente de verdad para poblar datos y configuración en DB.
- La creación o eliminación de archivos de ejemplo no debe quedar implícita dentro de `seed-one`.
- Los comandos de example cases deben separar:
  - recreación de archivos;
  - ejecución de seeders;
  - cleanup del repo.

### 2. Activación explícita

- Los servicios demo bajo `internal/services/test/` no deben quedar importados permanentemente desde `cmd/api`, `cmd/sqs-consumer` o `cmd/cmd-cli`.
- La activación de esos servicios debe resolverse mediante `internal/services/examplesregistry/`.
- `examplesregistry` debe poder quedar vacío sin romper compilación.

### 3. Catálogo estable

- Cada caso debe tener un `case` estable y humano.
- Cada caso debe documentar explícitamente:
  - seeder asociado;
  - `process_type` sembrados;
  - servicios Go que recrea;
  - carpeta Bruno asociada;
  - comando de create;
  - comando de delete;
  - comando de recreate.

### 4. Casos soportados actualmente

- `process_lifecycle_manager`
- `test_process_scenarios`
- `process_lifecycle_auto_invoke`
- `multi_queue_batch_one_table_process_lifecycle`
- `multi_queue_batch_one_table_recreate_records`

### 5. Shared files entre casos

- El sistema de create/delete debe tolerar archivos compartidos entre casos.
- Si dos casos comparten los mismos archivos activos, por ejemplo `mqb1t`, `delete-example-case` no debe eliminarlos mientras otro caso activo todavía los necesite.

### 6. Bruno de ejemplos

- Cada caso debe recrear una carpeta dedicada bajo `bruno/legacy/process-lifecycle/example-cases/<case>/`.
- Los requests Bruno deben venir con body base listo para ejecutar.
- Los requests deben usar `auth: bearer` y `X-Client-Code: bruno`.

### 7. Cleanup

- `delete-example-case` solo debe actuar sobre archivos administrados por el sistema de example cases.
- No debe borrar código productivo ajeno.
- Debe poder ejecutarse para un caso puntual o para `all`.

## Acceptance Criteria

- El repositorio compila sin importar permanentemente servicios demo.
- Un usuario puede recrear un caso específico con un solo comando.
- Un usuario puede eliminar un caso específico con un solo comando.
- La documentación humana enumera los `process_type` reales que siembra cada caso.
- Los requests Bruno viejos de ejemplos quedan reemplazados por carpetas organizadas por caso.

## Trazabilidad

- `Makefile`
- `cmd/tools/example-case-manager/main.go`
- `internal/services/examplesregistry/`
- `doc/info/process-lifecycle/example-cases.md`
