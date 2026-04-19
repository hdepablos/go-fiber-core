## Observabilidad: CloudWatch Logs

Se agregó soporte completo y modular para CloudWatch Logs con Terraform. Cada servicio (APIs, Lambdas, consumers, crons, workers) crea su propio Log Group con la siguiente convención:

- Nombre del Log Group: `/app/${project_name}/${service_name}`
- Retención: configurable por variable (`log_retention_in_days`, default: 7 días)
- LocalStack: compatible mediante el provider AWS cuando `environment = "local"`

### Logs de Aplicación (Zap)

Control de destino de salida mediante variable de entorno:

- `LOG_OUTPUT` controla dónde se escriben los logs de la aplicación:
  - `file`: solo archivos locales en `pkg/logs/` (rotación con Lumberjack)
  - `stdout`: solo consola (ideal para CloudWatch en Lambda)
  - `both`: archivos locales y consola a la vez
  - Sin definir: por defecto `file` en `APP_ENV=local`, `stdout` en otros entornos

Recomendaciones:

- Desarrollo local con `make watch`:
  - `APP_ENV=local`
  - `LOG_OUTPUT=file` (o dejarlo vacío para usar el default)
- Modo Lambda local con `make watch-lambda`:
  - `APP_ENV=local`
  - `LOG_OUTPUT=both` para mantener archivos locales y enviar a CloudWatch (LocalStack)

### Convención operativa por entorno

Para nuevas solicitudes de logger del proyecto, la convención canónica es:

- producción: escribir a `stdout` para que AWS capture los logs en CloudWatch,
- local: usar logger por proceso específico,
- local con archivo dedicado: usar un logger explícito orientado al proceso o dominio que se quiere depurar.

Esto significa:

- no se debe diseñar el logger de producción pensando en escribir archivos locales,
- no se debe usar un logger genérico sin nombre cuando el flujo pertenece a un proceso identificable,
- y en local se debe preferir `GetLogger("nombre_proceso")` o `GetLoggerToFile("nombre_proceso", "...")` según el caso.

### Producción en AWS

En producción, EKS o Lambda, la salida esperada es `stdout`.

AWS y/o la infraestructura contenedorizada son responsables de:

- capturar `stdout`,
- enrutarlo a CloudWatch Logs,
- aplicar retención,
- y permitir filtros o alarmas posteriores.

Patrón esperado:

- `APP_ENV != local`
- `LOG_OUTPUT=stdout`, o dejarlo vacío para usar el default no local

Esto aplica también a logs estructurados como:

- `redis_guard`
- `rate_limit_guard`
- logs de adapters externos
- logs batch por proceso

### Local por proceso específico

En local, la unidad mínima recomendada de logging es el proceso o caso de uso.

Ejemplos:

```go
log := logger.GetLogger("punitorios")
```

```go
log := logger.GetLogger("process-lifecycle")
```

```go
log := logger.GetLoggerToFile("imputations", "pkg/logs/imputations-debug.log")
```

Objetivo:

- separar debugging por proceso,
- evitar ruido cruzado entre dominios,
- y facilitar soporte puntual sin contaminar todos los logs locales.

### Regla práctica para nuevas implementaciones

Si te piden "logger" para una funcionalidad nueva:

1. En producción, asumir salida a AWS via `stdout`.
2. En local, usar logger por proceso específico.
3. Si se necesita archivo local dedicado, usar `GetLoggerToFile(...)`.
4. Mantener nombres de logger estables y alineados con el proceso, servicio o integración.

No conviene:

- crear un logger local genérico para todo el sistema,
- mezclar un mismo archivo para múltiples procesos pesados,
- o desviar producción a archivos locales como estrategia principal.

### Variables de Terraform

- `project_name`: nombre del proyecto (usado en nombres y tags)
- `environment`: `local | staging | prod`
- `log_retention_in_days`: días de retención (default: 7)
- `enable_cloudwatch_in_local`: si es `true`, se crean Log Groups en LocalStack; si es `false`, puedes optar por logs locales sin CloudWatch

Ubicación del módulo reusable: `terraform/modules/cloudwatch_logs`

### Integración con Lambdas

El módulo de Lambdas integra automáticamente CloudWatch Logs:

- IAM: adjunta `AWSLambdaBasicExecutionRole` (incluye permisos para logs)
- Outputs: expone el nombre/ARN del Log Group por servicio
- Ver ejemplo en: `terraform/modules/lambda_function/main.tf`

### Servicios NO-Lambda (ECS/containers/workers)

Para servicios no-Lambda se provee una policy mínima opcional:

- Acciones: `logs:CreateLogStream`, `logs:PutLogEvents`
- Activar con: `create_writer_policy = true` en el módulo `cloudwatch_logs`
- Adjuntar la policy ARN al rol del servicio

### Comandos útiles (Makefile)

Ver y seguir logs rápidamente desde la terminal:

- Listar grupos del proyecto:
  ```bash
  make logs-groups
  ```
- Tail de un servicio (ej: api), desde la última hora:
  ```bash
  make logs-tail service=api since=1h
  ```

Ambos comandos respetan los endpoints de LocalStack o AWS real según la variable `APP_ENV` y las opciones `AWS_PROFILE_NAME`/`LOCALSTACK_ENDPOINT_BASE`.

### Outputs

Luego de aplicar Terraform:
```bash
terraform output log_groups
```
Retorna un mapa con los nombres de los Log Groups por servicio para fácil referencia.

### Buenas prácticas

- Producción: usa `log_retention_in_days` bajo para reducir costos (p. ej. 7 a 14 días).
- Depuración puntual: activa `enable_cloudwatch_in_local` en local y `DB_LOG_LEVEL=warn` con `DB_SLOW_THRESHOLD_MS` para identificar consultas lentas.
- Seguridad: evita `info` en producción para no exponer datos sensibles en los logs.

## Filtros recomendados para batch fanout

El motor batch usa dos familias de logs estructurados para observabilidad operativa:

- `log_type=redis_guard`
- `log_type=rate_limit_guard`

Filtros sugeridos en CloudWatch Logs Insights o filtros de patrones:

```text
{ $.log_type = "redis_guard" }
```

```text
{ $.log_type = "rate_limit_guard" }
```

```text
{ $.log_type = "rate_limit_guard" && $.event_type = "external_http_429" }
```

```text
{ $.log_type = "rate_limit_guard" && $.scope = "internal" }
```

```text
{ $.log_type = "rate_limit_guard" && $.event_type = "external_dependency_timeout" }
```

Esto permite separar rápidamente:

- errores Redis del núcleo batch,
- rate limit interno del core,
- respuestas `429` de APIs internas o externas,
- y timeouts de dependencias HTTP externas.

## Configuración de Logs de Base de Datos

El sistema permite controlar el nivel de detalle de los logs de base de datos (GORM) mediante variables de entorno, sin necesidad de recompilar.

### Variables de Entorno

| Variable | Descripción | Valores Posibles | Default |
|----------|-------------|------------------|---------|
| `DB_LOG_LEVEL` | Controla qué se imprime en la consola. | `silent`, `error`, `warn`, `info` | Depende de `APP_ENV` (info en local, silent en prod) |
| `DB_SLOW_THRESHOLD_MS` | Define el umbral para considerar una query como "lenta". | Número entero (milisegundos) | `1000` (1 segundo) |
| `DB_SLOW_SQL_ENABLED` | Activa o desactiva por completo el tracking de slow SQL. | `true/1/yes` o `false/0/no` | Desactivado si no se define |

### Niveles de Log (`DB_LOG_LEVEL`)

1.  **`silent`**
    *   **Comportamiento:** No imprime nada en la consola.
    *   **Uso:** Recomendado para producción si se busca minimizar ruido y costos de ingestión de logs.

2.  **`error`**
    *   **Comportamiento:** Solo imprime errores de ejecución SQL.
    *   **Uso:** Recomendado estándar para producción. Permite ver fallos sin exponer datos sensibles de queries exitosas.

3.  **`warn`**
    *   **Comportamiento:** Imprime errores **Y** consultas lentas.
    *   **Uso:** Ideal para optimización en producción. Permite detectar cuellos de botella que superen el `DB_SLOW_THRESHOLD_MS`.

4.  **`info`**
    *   **Comportamiento:** Imprime **TODAS** las consultas SQL ejecutadas, sus parámetros y tiempo de ejecución.
    *   **Uso:** Desarrollo local. **NO** usar en producción (impacto en rendimiento y seguridad).

### Ejemplo de Configuración para Debugging de Performance

Para detectar queries que tardan más de 500ms en producción sin llenar los logs con todo el tráfico:

```bash
DB_LOG_LEVEL=warn
DB_SLOW_THRESHOLD_MS=500
```

En local, además puedes definir un archivo dedicado para registrar las consultas lentas:

```bash
DB_LOG_LEVEL=warn
DB_SLOW_THRESHOLD_MS=500
DB_SLOW_LOG_FILE=pkg/logs/db-slow.log
```

En producción (CloudWatch), las consultas que superen el umbral aparecerán como `SLOW SQL` en los logs de GORM, por lo que puedes crear un filtro en CloudWatch Logs que busque esa cadena para identificarlas rápidamente.
