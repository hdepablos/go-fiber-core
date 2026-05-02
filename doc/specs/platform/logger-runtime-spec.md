---
domain: platform
summary: Convención de runtime del logger por entorno, granularidad por proceso, tipos de log obligatorios y reglas de destino para producción y local.
when_to_read:
  - cambios en logger o logging estructurado
  - nuevas implementaciones que necesiten logging
  - cambios en tipos de log guard redis_guard rate_limit_guard execution_guard
  - observabilidad de batch o integraciones HTTP
code_paths:
  - internal/logger/
  - internal/logger/logger.go
  - internal/logger/guard_logs.go
related_info:
  - doc/info/operations/logs.md
related_specs:
  - doc/specs/process-lifecycle/batch-observability-spec.md
  - doc/specs/architecture/external-http-client-spec.md
status: active
---

# Logger Runtime Spec

## Objetivo

Formalizar la convención de runtime del logger para entornos locales y productivos.

## Alcance

Aplica a:

- `internal/logger/`,
- servicios y handlers que usen `logger.GetLogger(...)`,
- logging estructurado de batch, adapters externos y procesos operativos,
- y nuevas solicitudes del equipo relacionadas con "logger".

## Reglas de destino por entorno

- En producción, el logger debe escribir a `stdout`.
- En Lambda y EKS, `stdout` debe considerarse el destino canónico para ingestión por AWS/CloudWatch.
- En local, el logger puede escribir a archivo, a `stdout` o a ambos, según `LOG_OUTPUT`.
- Si `LOG_OUTPUT` no está definido:
  - `APP_ENV=local` debe privilegiar archivo local.
  - `APP_ENV!=local` debe privilegiar `stdout`.

## Reglas de granularidad

- En local, nuevas implementaciones deben usar logger por proceso específico.
- El nombre del logger debe representar el proceso, servicio, integración o caso de uso.
- No se debe introducir un logger genérico sin nombre de proceso cuando el flujo pertenece a un dominio identificable.

Ejemplos válidos:

- `logger.GetLogger("punitorios")`
- `logger.GetLogger("process-lifecycle")`
- `logger.GetLogger("external-http-client")`
- `logger.GetLoggerToFile("mi_proceso", "pkg/logs/mi_proceso-debug.log")`

## Reglas de archivos locales

- Los archivos locales son una herramienta de depuración local, no el destino principal de producción.
- Si se requiere un archivo dedicado, debe usarse `GetLoggerToFile(...)`.
- El nombre del archivo debe ser consistente con el proceso o caso de uso.
- No se deben mezclar múltiples procesos pesados en un mismo archivo de depuración salvo justificación explícita.

## Reglas para producción en AWS

- Producción no debe depender de archivos locales para observabilidad principal.
- Los logs estructurados deben ser visibles en CloudWatch a través de `stdout`.
- La retención, búsqueda y alarmado deben resolverse en la infraestructura AWS, no en archivos locales del contenedor o función.

## Invariantes

- La semántica de logging debe ser consistente entre servicios nuevos y existentes.
- La observabilidad estructurada (`redis_guard`, `rate_limit_guard`, etc.) debe seguir fluyendo a `stdout` en producción.
- La configuración de runtime del logger no debe romper la visibilidad en AWS.
- La depuración local debe poder hacerse por proceso específico sin alterar el comportamiento de producción.

## Acceptance Criteria

- La documentación humana explica que producción escribe a AWS vía `stdout`.
- La documentación humana explica que en local se usa logger por proceso específico.
- `AGENTS.md` refleja esta convención como regla del repositorio.
- Las nuevas implementaciones de logger siguen esta política salvo excepción documentada.

## Trazabilidad

- `internal/logger/logger.go`
- `internal/logger/guard_logs.go`
- `doc/info/operations/logs.md`
- `AGENTS.md`
