# Instrucciones del Repositorio

## Documentacion por defecto

Cuando una solicitud implique documentar, redocumentar, reorganizar o ampliar documentacion del proyecto, la convencion por defecto es:https://docs.google.com/spreadsheets/d/1j6chmvLAwotcQnvTod6MZSZVi2nVoqlTLk3-vflN_RI/edit?gid=0#gid=0

- Crear o actualizar documentacion para humanos en `doc/info/`.
- Crear o actualizar documentacion normativa para IA y Spec-Driven Development en `doc/specs/`.
- Mantener `README.md` como portal principal y actualizar sus enlaces cuando cambie el mapa documental.
- Actualizar `doc/info/README.md` y `doc/specs/README.md` cuando se agreguen documentos nuevos o cambie la clasificacion.

## Regla de separacion

- `doc/info/` explica contexto, uso, operacion, troubleshooting y entendimiento humano.
- `doc/specs/` define contratos, invariantes, acceptance criteria y reglas verificables.
- No duplicar la misma informacion completa en ambos lados: `info` explica y `specs` norman.

## Regla de ubicacion

- No crear `.md` documentales fuera de `doc/info/`, `doc/specs/` o `README.md` raiz.
- Excepciones permitidas: templates Markdown funcionales, o READMEs tecnicos muy locales que no formen parte del mapa global.

## Regla de trazabilidad

Toda nueva documentacion debe:

- enlazar documentos relacionados cuando corresponda,
- evitar duplicacion con documentos existentes,
- respetar la clasificacion por dominio,
- dejar claro si el archivo pertenece a humanos (`info`) o IA/SDD (`specs`).

## Convenciones tecnicas de servicios

Todo servicio nuevo o refactorizado debe estructurarse con:

- interface segregation,
- constructor injection,
- method receivers,
- implementacion concreta no exportada cuando no exista una razon fuerte para exportarla.

### Patron esperado

1. Definir una interface pequena y enfocada en el contrato del servicio.
2. Implementar una struct concreta, preferentemente no exportada.
3. Exponer un constructor `New...` que reciba dependencias explicitas.
4. Implementar la interface mediante method receivers sobre la struct concreta.

## Convenciones de integraciones HTTP externas

Toda nueva integracion HTTP reutilizable debe seguir una estructura estandar comun.

### Fuente de configuracion

- La configuracion de APIs externas debe salir de `internal/appconfig/config.yml`.
- La fuente canonica debe ser la seccion `apis.xxx`.
- La resolucion para nuevos casos debe hacerse con `appConfig.APIConfig("xxx")`.
- Los adapters deben recibir `config.ApiConfig`; no deben hardcodear `url`, `token` ni leer variables de entorno por su cuenta.

### Cliente HTTP compartido

- Toda llamada HTTP externa reutilizable debe centralizarse en `internal/services/externalhttp/`.
- El servicio comun debe encargarse de construir el cliente HTTP a partir de `config.ApiConfig`.
- Los adapters deben delegar en `externalhttp.NewClientFromAPIConfig(...)`.
- El adapter debe describir la integracion; no debe duplicar logging transversal, manejo de `429` ni errores de red.

### Politica estricta

- No usar `resty.New()` directamente dentro de adapters nuevos.
- Si una integracion nueva necesita salirse del patron, debe dejar justificacion explicita en codigo y documentacion.
- Si el servicio externo usa otro esquema de autenticacion distinto de bearer token, se debe extender el servicio comun en vez de romper el patron en el adapter.

### Scaffold de adapters externos

- Para nuevos adapters HTTP externos, usar `make create-external-adapter adapter_name=customer_api config_key=customer_api`.
- Ese scaffold debe generar una base alineada con `config.ApiConfig`, `externalhttp` y la resolucion canonica desde `apis.xxx`.

## Convenciones de batchflow y versiones

Los procesos batch deben separar negocio de perfil tecnico.

### Modelo de versionado

- Debe existir un unico `process_type` por dominio de negocio.
- Las diferencias `sequential` o `fanout` deben vivir en `process_versions` y su configuracion, no en nombres distintos de `process_type`.
- La version base debe ser secuencial.
- La version companion fanout debe reutilizar las mismas execution keys del negocio.

### Scaffold y cleanup

- Para procesos batch nuevos, usar `make create-batch-process ...`.
- El scaffold batch debe generar:
  - servicio del proceso,
  - seeder base secuencial,
  - seeder companion `_fanout`.
- Para limpiar procesos scaffold, usar `make delete-process kind=batch-process service_slug=...`.
- El scaffold batch no debe crear carpetas Bruno especificas por proceso; debe reutilizar `bruno/legacy/process-lifecycle/test-batch-process`.

## Observabilidad operativa de batch e integraciones

Cuando se modifique el core batch o integraciones HTTP externas, debe preservarse la observabilidad estructurada existente.

## Convenciones de logger

Cuando una solicitud pida "logger" o un ajuste de logging, debe asumirse esta convención por defecto.

### Producción

- En producción, Lambda o EKS, el logger debe escribir a `stdout`.
- AWS debe capturar esos logs mediante CloudWatch.
- No diseñar el logger productivo con archivos locales como destino principal.

### Local

- En local, el logger debe organizarse por proceso específico.
- Preferir `logger.GetLogger("nombre_proceso")` para separar dominios.
- Si hace falta un archivo dedicado por depuración puntual, usar `logger.GetLoggerToFile("nombre_proceso", "...")`.
- Evitar un logger genérico único para todos los procesos cuando el flujo pertenece a un dominio identificable.

### Regla práctica

- Si piden logger para producción: asumir AWS via `stdout`.
- Si piden logger para local: asumir logger por proceso específico.
- Si hace falta depuración puntual: usar archivo local específico por proceso, no un archivo global para todo.

### Tipos de logs obligatorios

- `log_type=redis_guard` para errores relevantes de Redis en el core batch.
- `log_type=rate_limit_guard` para rate limit interno del core y `429` externos.

### Reglas de emision

- Los errores Redis del motor batch no deben pasar silenciosamente.
- Un `429` de dependencia externa no debe pasar silenciosamente.
- El logging transversal debe vivir en el core compartido correspondiente, no repetirse por cada adapter o flujo.
- Si una solicitud implica capacidad, estres, Redis o fanout, revisar y mantener alineados los documentos operativos y normativos de observabilidad batch.

### Ejemplo de referencia

```go
type PaymentService interface {
    Process(ctx context.Context, amount float64) error
    Refund(ctx context.Context, id string) error
}

type paymentService struct {
    db     *sql.DB
    logger *slog.Logger
}

func NewPaymentService(db *sql.DB, logger *slog.Logger) PaymentService {
    return &paymentService{
        db:     db,
        logger: logger,
    }
}

func (s *paymentService) Process(ctx context.Context, amount float64) error {
    return nil
}

func (s *paymentService) Refund(ctx context.Context, id string) error {
    return nil
}
```

## Documentacion de base de datos

La base de datos debe mantener documentacion `info + specs` suficiente para:

- entender entidades, tablas y relaciones,
- identificar claves primarias, foraneas, pivotes e invariantes,
- mapear modelos GORM con migraciones SQL,
- permitir que en futuras solicitudes se pueda generar SQL a partir de la documentacion base sin relevar todo desde cero.

Cuando cambien tablas, relaciones, indices, enums o reglas de integridad, deben actualizarse:

- la documentacion humana de base de datos en `doc/info/`,
- la documentacion normativa de base de datos en `doc/specs/`,
- y sus enlaces en los indices documentales si corresponde.

## Documentacion del Makefile

El `Makefile` debe tener documentacion `info + specs` cuando no exista una cobertura canonica suficiente.

La documentacion del `Makefile` debe dejar claro:

- dominios de comandos,
- prerequisitos,
- efectos colaterales,
- comandos destructivos o sensibles,
- flujos principales de desarrollo, despliegue, datos y soporte.

## Convenciones de endpoints y Bruno

Todo endpoint nuevo o modificado debe evaluarse junto con su documentacion HTTP y su request correspondiente en Bruno.

### Regla general

- Todo endpoint nuevo debe tener documentacion humana en `doc/info/` con ejemplo de request si usa body.
- Si el endpoint define un contrato reutilizable o relevante para automatizacion, debe reflejarse tambien en `doc/specs/`.
- Todo endpoint nuevo o modificado debe tener request Bruno canónico cuando forme parte del API operable.

### Regla de organizacion en Bruno

- La coleccion canónica debe vivir en `bruno/api/`.
- Los endpoints bajo `/api/v1/...` deben organizarse en `bruno/api/v1/...` siguiendo la URL real.
- Endpoints fuera de `/api/v1`, como `/` o `/oauth/...`, deben agruparse por path real.
- Requests historicos, variantes de prueba o casos exploratorios deben preservarse en `bruno/legacy/` y no mezclarse con la colección principal.

### Regla de auth y headers

- Todo endpoint protegido debe tener `auth: bearer` y reutilizar `{{access_token}}`.
- El login o endpoints equivalentes deben actualizar `access_token` y, cuando aplique, `refresh_token`.
- Los requests operativos de Bruno deben usar por defecto `X-Client-Code: bruno`, salvo que exista un motivo explícito para otro valor.

### Regla de request bodies

- Todo endpoint `POST .../paginated` debe partir de una estructura base consistente compatible con `PaginationRequest`.
- Los endpoints multipart deben dejar claro el `content-type`, el nombre del campo archivo y las variables necesarias.
- Los ejemplos de body en documentación y Bruno deben mantenerse alineados con los DTOs reales del código.

## Referencias

- `doc/info/README.md`
- `doc/specs/README.md`
- `doc/specs/documentation-governance-spec.md`
- `doc/specs/documentation-defaults-spec.md`
- `doc/info/development/service-design-conventions.md`
- `doc/info/development/external-http-service-standard.md`
- `doc/info/development/service-runtime-and-scaffold.md`
- `doc/info/development/process-scaffold-and-cleanup.md`
- `doc/info/platform/makefile-guide.md`
- `doc/info/process-lifecycle/batch-fanout-guide.md`
- `doc/info/process-lifecycle/batch-capacity-and-stress-guide.md`
- `doc/specs/architecture/service-design-spec.md`
- `doc/specs/architecture/external-http-client-spec.md`
- `doc/specs/platform/logger-runtime-spec.md`
- `doc/specs/architecture/service-runtime-bootstrap-spec.md`
- `doc/specs/platform/process-scaffold-cleanup-spec.md`
- `doc/specs/process-lifecycle/batch-observability-spec.md`
- `doc/info/api/http-endpoints-guide.md`
- `doc/specs/api/http-endpoints-spec.md`
