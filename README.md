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

## Ciclo de vida de procesos (Process Lifecycle Manager)

La base de datos expone las funciones PL/pgSQL:

- `promote_process_version(p_process_version_id BIGINT, p_operator_id BIGINT, p_comment VARCHAR)`
- `replicate_process_version(p_process_version_id BIGINT) RETURNS BIGINT`
- `resolve_process_version(p_process_type_id BIGINT, p_sede_id BIGINT, p_override_process_version_id BIGINT DEFAULT NULL) RETURNS BIGINT`

La documentación funcional y de modelo de datos está en:

- [doc/info/process_lifecycle_manager.md](doc/info/process_lifecycle_manager.md)

### Endpoints HTTP

Todos los endpoints viven bajo `/api/v1/process-lifecycle` y usan el esquema de respuesta estándar (`status`, `message`, `data`, `errors`) definido en `internal/dtos/responses/response.go`, además del `GlobalErrorHandler` para mapear errores de dominio (`ErrNotFound`, `ErrInvalidArgument`, `ErrInternal`).

- `POST /api/v1/process-lifecycle/replicate`
  - Body:
    - `process_version_id` (int64, requerido, > 0)
  - Acción: invoca `replicate_process_version` y devuelve el nuevo `process_version_id` en `data.new_process_version_id`.

- `POST /api/v1/process-lifecycle/promote`
  - Body:
    - `process_version_id` (int64, requerido, > 0)
    - `comment` (string, requerido, máx. 300 chars)
  - Acción: invoca `promote_process_version` usando como `operator_id` el usuario autenticado (ID extraído del token).

- `POST /api/v1/process-lifecycle/resolve`
  - Body:
    - `process_type_id` (int64, requerido, > 0)
    - `sede_id` (int64, requerido)
    - `override_process_version_id` (int64, opcional, puede ser `null`)
  - Acción: invoca `resolve_process_version` y devuelve en `data`:
    - `process_version_id`: id de la versión efectiva resuelta.
    - `process_steps`: array de pasos asociados a esa versión, ordenados por `step_order`, con la forma:
      - `name`
      - `execution_key`
      - `config` (objeto JSON)
      - `step_order`

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

### Requests en Bruno

En la carpeta `bruno/process-lifecycle` hay requests de ejemplo que utilizan estos endpoints:

- `replicate-scenario.bru` → `POST /api/v1/process-lifecycle/replicate`
- `promote-scenario.bru` → `POST /api/v1/process-lifecycle/promote`
- `resolve-scenario.bru` → `POST /api/v1/process-lifecycle/resolve`

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
