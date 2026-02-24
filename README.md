# Project go-fiber-core

One Paragraph of project description goes here

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests
```bash
make all
```

## Logs

La documentación completa sobre logs (CloudWatch, niveles de log, buenas prácticas y comandos útiles) se encuentra en:

- [doc/info/logs.md](doc/info/logs.md)

## Migraciones

La convención de nombres y el uso de `make create-migration` (incluyendo procesos como `create_process_lifecycle_manager`) se encuentra en:

- [doc/info/create-migrations.md](doc/info/create-migrations.md)

## Seeders

La documentación detallada sobre los seeders (incluyendo `catalog_items` y el uso de `seed --only`) se encuentra en:

- [doc/info/seeders-catalog-items.md](doc/info/seeders-catalog-items.md)

## Campos `operator_id` (equivalente a `created_by`)

En el proyecto se usa el nombre `operator_id` para representar al usuario que ejecuta una acción relevante de negocio, equivalente al concepto clásico de `created_by` o `updated_by`. Es un campo de auditoría funcional que apunta al usuario operador.

Tablas donde se utiliza actualmente:

- `users.operator_id`: último operador que activó, desactivó o actualizó al usuario.
- `menu_user.operator_id`: operador que creó o modificó la relación menú-usuario.
- `menu_role.operator_id`: operador que creó o modificó la relación menú-rol.
- `process_versions.operator_id`: operador que creó la versión de proceso.
- `process_version_history.promoted_by`: operador que promovió una versión de proceso entre estados (por ejemplo, a `PROD`).

En los modelos de Go se expone como:

- `OperatorID` (`*uint` o `*uint64`): columna `operator_id` en base de datos.
- `Operator` (`*User`): relación hacia la tabla `users` para cargar los datos del operador.

En los flujos HTTP, el `operator_id` normalmente se toma del usuario autenticado (ID del `User` extraído del JWT) y se guarda cuando se realizan acciones como activación/desactivación de usuarios, asignación de menús o promoción de escenarios de proceso.

## Ciclo de vida de procesos (Process Lifecycle Manager)

La base de datos expone las funciones PL/pgSQL:

- `promote_process_version(p_process_version_id BIGINT, p_operator_id BIGINT, p_comment VARCHAR)`
- `replicate_process_version(p_process_version_id BIGINT, p_operator_id BIGINT) RETURNS BIGINT`
- `resolve_process_version(p_process_type_id BIGINT, p_sede_id BIGINT, p_override_process_version_id BIGINT DEFAULT NULL) RETURNS TABLE (process_version_id BIGINT, process_steps JSONB)`
- `move_process_version_to_test(p_process_version_id BIGINT) RETURNS VOID`

La documentación funcional y de modelo de datos está en:

- [doc/info/process_lifecycle_manager.md](doc/info/process_lifecycle_manager.md)
La explicación funcional del flujo (sin detalles técnicos) está en:

- [doc/info/process_lifecycle_manager_flow.md](doc/info/process_lifecycle_manager_flow.md)

### Reglas y coherencia de datos

- Unicidad de `PROD` por `(process_type_id, sede_id)` con índice filtrado.
- Promoción a `PROD` permitida solo desde `TEST` o `HISTORY`.
- Al promover, la `PROD` anterior pasa a `HISTORY` sin crear registro de historial.
- Cada promoción genera **un único** registro en `process_version_history` (para la nueva `PROD`):
  - Incluye `process_version_id`, `process_type_id`, `promoted_from_status`, `promoted_at`, `promoted_by`, `comment`.
- `process_version_history` tiene FK compuesta a `process_versions(id, process_type_id)`; el par `(id, process_type_id)` está protegido por una `UNIQUE` en `process_versions`.
- `process_types.is_visible` permite ocultar tipos de proceso en el frontend.

Visibilidad de tipos de proceso en frontend:

- El campo `process_types.is_visible` controla si un tipo de proceso debe ser mostrado en el frontend (por ejemplo, en la datatable donde los operadores seleccionan procesos y crean escenarios). Las consultas que alimentan esa vista deberían filtrar por `is_visible = TRUE` y `archived_at IS NULL`.

### Endpoints HTTP

Todos los endpoints viven bajo `/api/v1/process-lifecycle` y usan el esquema de respuesta estándar (`status`, `message`, `data`, `errors`) definido en `internal/dtos/responses/response.go`, además del `GlobalErrorHandler` para mapear errores de dominio (`ErrNotFound`, `ErrInvalidArgument`, `ErrInternal`).

- `POST /api/v1/process-lifecycle/replicate`
  - Body:
    - `process_version_id` (int64, requerido, > 0)
    - `operator_id` (int64, requerido, > 0): id del usuario que crea la nueva versión (se replica el escenario y se registra este operador en `process_versions.operator_id`).
  - Acción: invoca `replicate_process_version` y devuelve el nuevo `process_version_id` en `data.new_process_version_id`.

- `POST /api/v1/process-lifecycle/promote`
  - Body:
    - `process_version_id` (int64, requerido, > 0)
    - `comment` (string, requerido, máx. 300 chars)
    - `promoted_by` (int64, requerido, > 0): id del operador que ejecuta la promoción, se persiste en `process_version_history.promoted_by`.
  - Acción: invoca `promote_process_version` con el `promoted_by` indicado y registra el historial correspondiente (un solo registro por promoción).

- `POST /api/v1/process-lifecycle/resolve`
  - Body:
    - `process_type_id` (int64, requerido, > 0)
    - `sede_id` (int64, requerido)
    - `override_process_version_id` (int64, opcional, puede ser `null`)
    - `roadmap` (int, requerido): define qué segmento de pasos se resolverá.
  - Acción: invoca `resolve_process_version` y devuelve en `data`:
    - `process_version_id`: id de la versión efectiva resuelta.
    - `process_steps`: array de pasos asociados a esa versión, ordenados por `step_order`, con la forma:
      - `name`
      - `execution_key`
      - `config` (objeto JSON)
      - `step_order`

- `POST /api/v1/process-lifecycle/to-test`
  - Body:
    - `process_version_id` (int64, requerido, > 0)
  - Acción: invoca `move_process_version_to_test` para mover una versión desde `DRAFT` a `TEST`. Si la versión no existe o está archivada, se traduce a `ErrNotFound`. Si la versión no está en `DRAFT`, se traduce a `ErrInvalidArgument`.

- `POST /api/v1/process-lifecycle/run`
  - Endpoint genérico para ejecutar un proceso completo usando el motor de lifecycle (`RunResolvedProcess`).
  - Body (`RunProcessRequest`):
    - `process_type_id` (int64, requerido, > 0): identifica el tipo de proceso a ejecutar (ej. Loan risk).
    - `sede_id` (int64, requerido): sede desde la cual se resuelve la versión vigente.
    - `override_process_version_id` (int64, opcional, puede ser `null`):
      - `null` → usa la versión `PROD` vigente según `process_type_id` + `sede_id` (con fallback a versión global).
      - `!= null` → fuerza la ejecución de esa versión específica, respetando las reglas de `resolve_process_version`.
    - `roadmap` (int, requerido): define el segmento de pasos a ejecutar.
    - `input` (objeto JSON, requerido): bolsa de datos de negocio (`ServiceContext.Input`) que verán todos los servicios.
      - El handler garantiza que `input["sede_id"]` exista (se copia desde `sede_id` si no viene).
  - Acción:
    - Resuelve la versión efectiva (`resolve_process_version`).
    - Construye el registro de servicios a partir de los steps (`execution_key`, `step_order`, `config`, `required_keys`).
    - Ejecuta todos los servicios en orden con el `input` dado (`RunResolvedProcess`).
  - Respuesta exitosa (HTTP 200, `status = "success"`):
    - `data.process_version_id`: id de la versión ejecutada.
    - `data.input`: JSON de entrada (incluyendo `sede_id`).
    - `data.results`: mapa de resultados por servicio (`execution_key` → `StepResult`).
    - `data.execute_ordered`: arreglo ordenado por `step_order` con la forma:
      - `service_path` (execution_key, ej. `loanrisk/NewAgeService`)
      - `step_order` (int).
  - Respuesta con error de ejecución (HTTP != 200, `status = "error"`):
    - El endpoint **no pierde contexto de ejecución**:
      - `data.process_version_id`: puede venir con el id resuelto antes del fallo.
      - `data.input`: JSON de entrada.
      - `data.results`: resultados parciales de servicios ejecutados hasta el momento del error.
      - `data.execute_ordered`: orden de steps ejecutados/parcialmente ejecutados.
      - `data.error`: detalle del fallo:
        - `code`: uno de:
          - `PROCESS_VERSION_NOT_FOUND` → cuando `resolve_process_version` no encuentra versión activa (mapeado a `domain.ErrNotFound` → HTTP 404).
          - `MISSING_REQUIRED_KEY` → falta una key obligatoria en `input` (por ejemplo, `salary`, `min_salary`, `is_renovation`; mapeado a `domain.ErrMissingRequiredKey` → HTTP 422).
          - `VALUE_OUT_OF_RANGE` → un valor está fuera de rango permitido (por ejemplo, salario menor al mínimo configurado; mapeado a `domain.ErrValueOutOfRange` → HTTP 422).
          - `INVALID_ARGUMENT` → otros errores de validación de entrada que no encajan en los dos casos anteriores (mapeados a `domain.ErrInvalidArgument` → HTTP 422).
          - `CRITICAL_ERROR` → errores críticos de negocio o técnicos que deben detener la cadena (por ejemplo, fallos en servicios marcados como críticos que no dependen solo de validaciones de input) → HTTP 500.
          - `INTERNAL_ERROR` → cualquier otro fallo inesperado (incluye errores como “servicio no encontrado en el registro” si falta configuración de servicios) → HTTP 500.
        - `message`: mensaje técnico completo, útil para debugging y trazas.

### Manejo de errores

Las excepciones levantadas por las funciones SQL se traducen a errores de dominio:

- `Process version not found or archived`
- `Process type does not exist or is archived`
- `No active version found`
  - → `domain.ErrNotFound` → HTTP 404 + payload estándar.

- `Override version invalid`
- `Cannot promote version without steps`
- `Promotion comment exceeds 300 characters`
  - → `domain.ErrInvalidArgument` → HTTP 400 + payload estándar.

Los errores inesperados se traducen a `domain.ErrInternal` → HTTP 500 con mensaje genérico.

En el caso particular del endpoint `POST /api/v1/process-lifecycle/run`, además de este mapeo:

- El handler devuelve un payload rico con:
  - `process_version_id`, `input`, `results`, `execute_ordered` y un bloque `error` con `code` + `message`.
- El código HTTP refleja el tipo de error:
  - 404 cuando el proceso/versión no existe (`PROCESS_VERSION_NOT_FOUND`).
  - 422 cuando la entrada es inválida:
    - `MISSING_REQUIRED_KEY` (falta de keys como `salary`, `min_salary`, `salary_checked`, `is_renovation`, etc.).
    - `VALUE_OUT_OF_RANGE` (por ejemplo, salario menor al mínimo permitido por configuración).
    - `INVALID_ARGUMENT` (otros problemas de datos de entrada).
  - 500 en errores críticos o internos (`CRITICAL_ERROR`, `INTERNAL_ERROR`).

### Requests en Bruno

En la carpeta `bruno/process-lifecycle` hay requests de ejemplo que utilizan estos endpoints:

- `replicate-scenario.bru` → `POST /api/v1/process-lifecycle/replicate`
- `promote-scenario.bru` → `POST /api/v1/process-lifecycle/promote`
- `resolve-scenario.bru` → `POST /api/v1/process-lifecycle/resolve`
- `move-to-test-scenario.bru` → `POST /api/v1/process-lifecycle/to-test`
- `run-process.bru` → `POST /api/v1/process-lifecycle/run`
  - Body de ejemplo (Loan risk lifecycle):
    ```json
    {
      "process_type_id": 2,
      "sede_id": 1,
      "override_process_version_id": null,
      "input": {
        "age": 45,
        "salary": 2500000
      }
    }
    ```
  - Body de ejemplo (otro proceso):
    ```json
    {
      "process_type_id": 1,
      "sede_id": 1,
      "override_process_version_id": 123,
      "input": {
        "customer_id": 999,
        "operation_id": "ABC-123"
      }
    }
    ```

### Registro de servicios configurables (Loan Risk y otros)

El motor de lifecycle ejecuta servicios a partir de los `execution_key` definidos en `process_steps.execution_key`. Para que estos servicios estén disponibles en tiempo de ejecución:

- Cada servicio debe registrar su constructor en el mapa global:
  - Ejemplo (`internal/services/loanrisk/age.go`):
    ```go
    func init() {
      serviceconfig.Register("loanrisk/NewAgeService", NewAgeService)
    }
    ```
- Es **obligatorio** que el binario que ejecuta el motor (`cmd/api`, `cmd/cmd-cli`, workers, etc.) importe el paquete de servicios con **importación en blanco**, para que los `init()` se ejecuten y el registro se llene:
  - Ejemplo en `cmd/api/main.go`:
    ```go
    import (
      // ...
      fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"

      // Importación en blanco para asegurar el registro de servicios Loan Risk
      _ "go-fiber-core/internal/services/loanrisk"
    )
    ```

Si se omite esta importación:

- El motor no encuentra las factories (`GetServiceFactory`) para claves como `loanrisk/NewAgeService`.
- La ejecución de `ExecuteServicesInOrder` falla con un error interno:
  - `error al obtener la fábrica para loanrisk/NewAgeService: servicio no encontrado en el registro: loanrisk/NewAgeService`
- En el contexto del endpoint `/run`:
  - El HTTP será 500.
  - `data.error` vendrá con:
    - `code = "INTERNAL_ERROR"`.
    - `message` con el detalle del problema de registro.

Todos heredan la configuración de autenticación y base URL (`{{urlBase}}`) definida en `bruno/environments`.

## Infraestructura y Optimización

El proyecto utiliza configuraciones avanzadas para maximizar el rendimiento y minimizar costos en AWS Lambda:

*   **Arquitectura:** ARM64 (Graviton2)
*   **Memoria:** 1769 MB (1 vCPU completo)
*   **Logs:** Métricas de CPU y Goroutines integradas.

Para más detalles, consulta la documentación completa en: [doc/info/lambda-optimization.md](doc/info/lambda-optimization.md).

## Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
