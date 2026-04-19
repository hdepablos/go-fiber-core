# External HTTP Service Standard

## Objetivo

Definir una estructura estándar para todas las solicitudes HTTP hacia APIs externas o internas no controladas por este servicio.

La regla buscada es simple:

- los adapters describen la integración,
- el servicio común ejecuta la llamada,
- el servicio común registra logs y avisos transversales,
- y la respuesta HTTP se devuelve tal cual al caller.

## Decisión de arquitectura

Las llamadas HTTP externas deben centralizarse en:

- `internal/services/externalhttp/`

La configuración de cada integración debe salir de:

- `internal/appconfig/config.yml`
- sección `apis.xxx`

Ejemplo actual:

```yaml
apis:
  customer_api:
    url: ${CUSTOMER_API_URL}
    token: ${CUSTOMER_API_TOKEN}
    timeout_seconds: 10
```

El servicio estándar:

- recibe un `resty.Client` por constructor,
- recibe un contrato `Request`,
- ejecuta la llamada,
- registra errores de transporte,
- registra `429`,
- puede disparar una notificación central mediante un hook,
- y devuelve `*resty.Response` sin transformar el payload.

## Qué debe hacer un adapter

Un adapter debe:

- recibir `config.ApiConfig` del caller,
- asumir que esa configuración proviene de `appConfig.APIConfig("xxx")`,
- definir `source`,
- definir `method`,
- definir `endpoint`,
- definir headers o body propios,
- y delegar la ejecución al servicio `externalhttp`.

Un adapter no debe:

- crear `resty.New()` directamente,
- duplicar logs de `429`,
- duplicar logs de errores de red,
- reimplementar retry genérico transversal,
- ni crear su propia estrategia de observabilidad por fuera del servicio común.

## Resolución canónica de configuración

La forma recomendada para obtener la configuración es:

```go
apiCfg, err := appConfig.APIConfig("backoffice")
if err != nil {
    return err
}

adapter := adapters.NewBackofficeAdapter(apiCfg)
```

`AppConfig` mantiene además campos tipados existentes, pero la vía canónica para nuevas integraciones debe ser `APIConfig("xxx")`.

## Campos soportados hoy

El contrato actual de `apis.xxx` soporta:

- `url`
- `token`
- `timeout_seconds`
- `headers`

Comportamiento:

- `url` define el `BaseURL`,
- `token` se usa como bearer token por defecto,
- `timeout_seconds` controla el timeout del cliente HTTP común,
- `headers` permite headers base compartidos para toda la integración.

## Comportamiento esperado

### Si la llamada responde `200-299`

- se retorna la respuesta tal cual,
- sin modificar cuerpo ni código,
- y sin forzar error.

### Si la llamada responde distinto de `200-299`

- se retorna la respuesta tal cual,
- el caller decide la semántica de negocio,
- y el servicio común puede registrar avisos/notificaciones según el tipo de evento.

### Si la llamada responde `429`

- el servicio común registra `log_type=rate_limit_guard`,
- `event_type=external_http_429`,
- `scope=external`,
- y conserva `retry_after` cuando exista.

### Si hay error de red, timeout o cancelación

- el servicio común registra `log_type=rate_limit_guard`,
- `event_type=external_dependency_error`,
- `scope=external`,
- y retorna el error original.

### Si se alcanza `timeout_seconds`

- el servicio común registra `log_type=rate_limit_guard`,
- `event_type=external_dependency_timeout`,
- `scope=external`,
- y devuelve el error original para que el caller decida la semántica.

## Hook de notificación

El servicio estándar contempla un `Notifier` opcional.

Esto permite que en el futuro se agregue en un solo punto:

- notificación a Discord,
- alertas a SNS,
- integración con incident management,
- o cualquier otra reacción transversal.

El adapter no debe conocer esa lógica.

## Patrón recomendado

```go
type CustomerAdapter interface {
    Send(ctx context.Context, payload any) (*resty.Response, error)
}

type customerAdapter struct {
    httpClient externalhttp.Client
}

func NewCustomerAdapter(client *resty.Client) CustomerAdapter {
    return &customerAdapter{
        httpClient: externalhttp.NewClient(client, nil),
    }
}

func (a *customerAdapter) Send(ctx context.Context, payload any) (*resty.Response, error) {
    return a.httpClient.Do(ctx, externalhttp.Request{
        Source:   "customer_api",
        Method:   http.MethodPost,
        Endpoint: "/v1/customers",
        Headers: map[string]string{
            "Content-Type": "application/json",
        },
        Body: payload,
    })
}
```

## Regla para el equipo

Toda nueva integración HTTP debe usar esta estructura estándar salvo excepción técnica explícita.

Toda nueva integración HTTP debe leer su configuración desde `apis.xxx` en `config.yml`.

No se permite crear `resty.New()` dentro de adapters nuevos.

Si alguien necesita salirse del patrón, debe dejar justificación clara en código y documentación.

## Scaffold recomendado

Para crear un adapter base alineado con esta convención:

```bash
make create-external-integration api_key=customer_api
```

o si quieres separar pasos:

```bash
make create-external-api-config api_key=customer_api
make create-external-adapter adapter_name=customer_api config_key=customer_api
```

Eso genera un adapter con:

- constructor basado en `config.ApiConfig`,
- uso de `externalhttp.NewClientFromAPIConfig(...)`,
- y método `Do(...)` como punto inicial para especializar la integración.

El comando `create-external-api-config` agrega en `config.yml`:

```yaml
apis:
  customer_api:
    url: ${CUSTOMER_API_URL}
    token: ${CUSTOMER_API_TOKEN}
    timeout_seconds: 10
```

El comando `create-external-integration` ejecuta ambos pasos y debe considerarse el flujo operativo por defecto para integraciones nuevas.

## Guía de evolución futura

Si en el futuro quieres ampliar `apis.xxx`, estos son los puntos a tocar:

### `auth_type`

Casos típicos:

- `bearer`
- `basic`
- `api_key_header`
- `none`

Cambios recomendados:

- agregar el campo a `config.ApiConfig`,
- extender `externalhttp/factory.go` para construir autenticación según `auth_type`,
- mantener `bearer` como default backward compatible,
- actualizar la spec y esta guía.

### `timeout_seconds`

Ya está implementado.

Puntos tocados:

- `internal/dtos/config/config.go`
- `internal/services/externalhttp/factory.go`
- `internal/services/externalhttp/service.go`

### `headers`

Ya está contemplado en el contrato y el factory los aplica como headers base.

Uso recomendado:

- headers estáticos de integración,
- por ejemplo `X-Api-Version`, `X-Tenant`, `Accept`.

No conviene usarlo para:

- headers dinámicos por request,
- tokens efímeros por usuario,
- o correlación específica del request; eso sigue viviendo en `externalhttp.Request`.

## Casos ya alineados

- `internal/adapters/backoffice_adapter.go`
- `internal/adapters/discord_adapter.go`

## Casos a migrar gradualmente

Cualquier llamada HTTP externa que todavía use:

- `resty.New()` directo en el adapter,
- `http.Client.Do(...)` para integraciones reutilizables,
- o lógica manual de logging de `429`,

debe migrarse al servicio estándar cuando se toque esa integración.

## Trazabilidad

- `internal/services/externalhttp/service.go`
- `internal/services/externalhttp/factory.go`
- `internal/services/externalhttp/service_test.go`
- `internal/adapters/backoffice_adapter.go`
- `internal/adapters/discord_adapter.go`
- `cmd/tools/external-api-config-scaffold/main.go`
- `cmd/tools/external-api-adapter-scaffold/main.go`
- `internal/logger/guard_logs.go`
- `doc/specs/architecture/external-http-client-spec.md`
