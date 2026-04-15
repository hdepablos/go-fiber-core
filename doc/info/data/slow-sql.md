Detección y Observabilidad de Consultas Lentas (Slow SQL)
========================================================

Este documento describe cómo detectar, registrar y monitorear consultas lentas tanto en local como en producción (CloudWatch), para los dos drivers utilizados en el proyecto: GORM y PGX.

Variables de entorno
--------------------

- DB_LOG_LEVEL: silent | error | warn | info
  - Recomendado: warn en producción, warn o info en local.
- DB_SLOW_THRESHOLD_MS: Umbral en milisegundos para considerar una consulta como lenta.
  - Recomendado: 500
- DB_SLOW_SQL_ENABLED: Flag para activar/desactivar el tracking de slow SQL.
  - Valores: true/1/yes para activar, false/0/no para desactivar. Default: desactivado (si no se define).
- DB_SLOW_LOG_FILE (solo local): Ruta de archivo para duplicar logs de consultas lentas.
  - Ejemplo: pkg/logs/db-slow.log

Comportamiento
--------------

- Local (APP_ENV=local)
  - Con DB_LOG_LEVEL=warn y DB_SLOW_THRESHOLD_MS=500, solo se registran errores y queries que superen 500ms.
  - Si además defines DB_SLOW_LOG_FILE, las consultas lentas se duplican en ese archivo dedicado (útil para revisar únicamente las lentas).
- Producción
  - Con `DB_SLOW_SQL_ENABLED=true`, `DB_LOG_LEVEL=warn` y `DB_SLOW_THRESHOLD_MS=500`, se registran en STDOUT únicamente las “SLOW SQL” y errores, que CloudWatch indexa.

GORM
----

- Implementación en: internal/database/connections/gorm/gorm_connect.go
- Características:
  - Umbral configurable con DB_SLOW_THRESHOLD_MS.
  - Nivel configurable con DB_LOG_LEVEL.
  - Duplicación en archivo local si DB_SLOW_LOG_FILE está definido y APP_ENV=local.

PGX
---

- Implementación en: internal/database/connections/pgx/pgx_connect.go
- Características:
  - Usa pgx/v5/tracelog para interceptar eventos y medir su duración.
  - Umbral configurable con DB_SLOW_THRESHOLD_MS.
  - Nivel configurable con DB_LOG_LEVEL.
  - Duplicación en archivo local si DB_SLOW_LOG_FILE está definido y APP_ENV=local.
  - Para entradas “lentas”, escribe líneas con el prefijo SLOW SQL e incluye duration y sql.

CloudWatch: Filtro, Métrica y Alarma
------------------------------------

Se agregaron filtros y alarmas de ejemplo en Terraform que detectan el texto SLOW SQL en los log groups de cada Lambda y generan una métrica SlowSQLCount (namespace DB/SlowQueries).

Archivo: terraform/main.tf

Variables:
- slow_sql_alarm_enabled (bool, default: true)
- slow_sql_alarm_threshold (número, default: 1)
- slow_sql_alarm_period (segundos, default: 300)
- slow_sql_metric_namespace (string, default: DB/SlowQueries)

Esto permite:
- Crear un Metric Filter que incrementa SlowSQLCount cada vez que aparece SLOW SQL.
- Crear una Metric Alarm por servicio si la suma en el período supera el umbral.

Ejemplos de configuración
-------------------------

Local (make watch / make watch-lambda)

```env
APP_ENV=local
DB_LOG_LEVEL=warn
DB_SLOW_THRESHOLD_MS=500
DB_SLOW_LOG_FILE=pkg/logs/db-slow.log
```

Producción (Lambdas)

```env
DB_LOG_LEVEL=warn
DB_SLOW_THRESHOLD_MS=500
```

En CloudWatch Logs, filtra por SLOW SQL o revisa la métrica DB/SlowQueries -> SlowSQLCount.

Targets útiles del Makefile
---------------------------

Para facilitar el análisis de slow SQL, existen dos comandos en el Makefile:

- Tail local del archivo de slow SQL:

```bash
make logs-tail-slow-sql
```

- Tail en CloudWatch filtrando solo las entradas con SLOW SQL:

```bash
make logs-tail-slow-sql-cloudwatch service=api since=1h
```

- service: nombre lógico del servicio (por ejemplo api, sqs-consumer).
- since: ventana de tiempo opcional (por defecto 1h).
