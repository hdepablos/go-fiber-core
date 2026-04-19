---
domain: architecture
summary: Estandar compartido para integraciones HTTP externas, resolución desde apis.xxx, observabilidad transversal y adapters reutilizables sobre externalhttp.
when_to_read:
  - nuevas integraciones HTTP
  - cambios en externalhttp
  - cambios en adapters externos
  - cambios en observabilidad de 429 o timeouts
code_paths:
  - internal/services/externalhttp/
  - internal/adapters/
  - internal/appconfig/config.yml
  - cmd/tools/external-api-config-scaffold/
  - cmd/tools/external-api-adapter-scaffold/
related_info:
  - doc/info/development/external-http-service-standard.md
related_specs:
  - doc/specs/platform/logger-runtime-spec.md
status: active
---

# External HTTP Client Spec

## Objetivo

Formalizar la estructura estándar para solicitudes HTTP hacia dependencias externas o internas no propias.

## Alcance

Aplica a:

- adapters HTTP reutilizables,
- servicios que encapsulen integraciones externas,
- data providers batch que llamen APIs remotas,
- y cualquier componente que deba hacer observabilidad transversal de `429` o errores de red.

La configuración de cada integración debe provenir de `internal/appconfig/config.yml` en la sección `apis`.

El contrato mínimo de `apis.xxx` debe soportar:

- `url`
- `token`
- `timeout_seconds`

Campos opcionales de extensión:

- `headers`
- `auth_type`

## Contrato principal

La ejecución HTTP compartida debe centralizarse en un servicio común.

El servicio común debe:

- recibir dependencias por constructor,
- aceptar un contrato de request explícito,
- ejecutar la llamada,
- registrar observabilidad transversal,
- y devolver la respuesta HTTP tal cual.

## Reglas de diseño

- Debe existir una interface pequeña y enfocada para el cliente HTTP compartido.
- La implementación concreta debe quedar detrás de constructor.
- Los adapters no deben duplicar lógica transversal de logging de `429` o errores de red.
- Los adapters nuevos no deben construir clientes con `resty.New()` directamente.
- El adapter solo debe aportar `source`, `method`, `endpoint`, `headers`, `query params` y `body`.
- Los adapters nuevos deben recibir `config.ApiConfig` y delegar la construcción del cliente a `externalhttp`.
- Debe existir una forma operativa de agregar entradas `apis.xxx` al `config.yml`.

## Resolución de configuración

- La vía canónica para nuevas integraciones debe ser `appConfig.APIConfig("<config_key>")`.
- La `<config_key>` debe corresponder a una entrada existente en `apis.xxx`.
- El servicio común debe poder construir el cliente HTTP a partir de `config.ApiConfig`.
- `timeout_seconds` debe controlar el timeout del cliente HTTP compartido.

## Reglas de respuesta

- Si la llamada responde `2xx`, el servicio debe retornar `*resty.Response` y `error=nil`.
- Si la llamada responde no `2xx`, el servicio debe retornar `*resty.Response` tal cual y `error=nil`, salvo que exista error de transporte.
- Si existe error de red, timeout o cancelación, el servicio debe retornar `error` original.

## Reglas de observabilidad

### Error de transporte

Ante error de red, timeout o cancelación:

- debe emitirse `log_type=rate_limit_guard`,
- `event_type=external_dependency_error`,
- `scope=external`.

### Timeout externo

Ante timeout del cliente HTTP compartido:

- debe emitirse `log_type=rate_limit_guard`,
- `event_type=external_dependency_timeout`,
- `scope=external`.

### HTTP 429

Ante respuesta `429`:

- debe emitirse `log_type=rate_limit_guard`,
- `event_type=external_http_429`,
- `scope=external`.

Campos recomendados:

- `source`
- `method`
- `endpoint`
- `status_code`
- `retry_after`

## Hook opcional de notificación

El servicio puede recibir un `Notifier` opcional para reacción transversal.

El notifier:

- no debe alterar la semántica del request principal,
- no debe impedir devolver la respuesta original,
- y sus fallos deben quedar aislados del flujo principal.

## Invariantes

- El logging transversal debe vivir en el servicio común, no en cada adapter.
- El adapter no debe transformar silenciosamente `429` en error técnico genérico.
- La respuesta HTTP original debe preservarse para el caller.
- La estructura debe ser reutilizable por nuevas integraciones sin copiar lógica.
- La guía humana debe prohibir explícitamente el uso directo de `resty.New()` en adapters nuevos.

## Acceptance Criteria

- Existe un servicio HTTP compartido para integraciones externas.
- Existe una forma canónica de resolver configuración desde `apis.xxx`.
- Existe una forma operativa de crear `apis.xxx` sin editar manualmente el YAML completo.
- Existe una forma operativa de crear `config + adapter` en una sola ejecución.
- Los adapters reutilizables existentes pueden apoyarse en ese servicio.
- Un `429` queda registrado de forma consistente.
- Un error de transporte queda registrado de forma consistente.
- Un timeout externo queda registrado de forma consistente.
- La documentación humana instruye al equipo a usar este patrón.

## Trazabilidad

- `internal/services/externalhttp/service.go`
- `internal/services/externalhttp/factory.go`
- `internal/services/externalhttp/service_test.go`
- `internal/adapters/backoffice_adapter.go`
- `internal/adapters/discord_adapter.go`
- `cmd/tools/external-api-config-scaffold/main.go`
- `cmd/tools/external-api-adapter-scaffold/main.go`
- `internal/logger/guard_logs.go`
- `doc/info/development/external-http-service-standard.md`
