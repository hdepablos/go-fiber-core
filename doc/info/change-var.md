Cambiar Variables de Entorno en Producción (añadir, modificar, eliminar)
========================================================================

Este procedimiento describe tres caminos para gestionar variables de entorno de Lambdas en producción. Aplica a **cualquier variable** (p. ej. `DB_SLOW_SQL_ENABLED`, `DB_LOG_LEVEL`, `FEATURE_FLAG_X`), tanto para **añadir**, **modificar** como **eliminar**.

Contexto (ejemplo con DB_SLOW_SQL_ENABLED)
-----------------------------------------

- Si `DB_SLOW_SQL_ENABLED` NO está definida, el tracking de slow SQL queda desactivado (default actual).
- Para activarlo de forma segura en producción, usa además:
  - `DB_LOG_LEVEL=warn`
  - `DB_SLOW_THRESHOLD_MS=500` (o el umbral que definas)
- Ver detalles en: [slow-sql.md](file:///private/var/www/go-fiber-core/doc/info/slow-sql.md) y [logs.md](file:///private/var/www/go-fiber-core/doc/info/logs.md).

Opción 1 — Terraform (global, recomendado)
-----------------------------------------

Impacta a todas las Lambdas administradas por IaC en el entorno. Mantiene el cambio versionado.

1) Abre el archivo de variables del entorno de producción (ej.: `terraform/prod.tfvars`) y en el mapa `lambda_env_vars` **agrega/modifica/elimina** claves:

```
DB_SLOW_SQL_ENABLED = "true"     # Activar
DB_LOG_LEVEL        = "warn"
DB_SLOW_THRESHOLD_MS = "500"
```

Para desactivar posteriormente (modificar):

```
DB_SLOW_SQL_ENABLED = "false"
```

Para **eliminar** una variable, quita la clave del mapa `lambda_env_vars` (no la dejes con cadena vacía; eliminar la clave la remueve del runtime).

2) Aplica cambios en AWS:

```
cd terraform
terraform init
terraform plan -var-file=prod.tfvars
terraform apply -var-file=prod.tfvars
```

3) Verifica durante la ventana de observación:

- Tail por servicio con filtro de “SLOW SQL”:

```
make logs-tail-slow-sql-cloudwatch service=api since=1h
```

- Métricas/Alarmas: en CloudWatch (namespace `DB/SlowQueries`, métrica `SlowSQLCount`).

4) Al finalizar la observación, vuelve a `DB_SLOW_SQL_ENABLED="false"` y aplica nuevamente.

Opción 2 — AWS Console (rápido, por función)
-------------------------------------------

Útil para un ajuste puntual sin esperar al pipeline, pero debes repetirlo por cada función.

1) Ve a AWS Console → Lambda → (función objetivo, p. ej. API).
2) En “Configuration” → “Environment variables” → “Edit”:
   - **Añadir**: crea una nueva fila con `NOMBRE=valor`.
   - **Modificar**: cambia el valor existente.
   - **Eliminar**: borra la fila de la variable.
   - Ejemplo (activar slow SQL): `DB_SLOW_SQL_ENABLED=true`, `DB_LOG_LEVEL=warn`, `DB_SLOW_THRESHOLD_MS=500`
   - Ejemplo (desactivar): `DB_SLOW_SQL_ENABLED=false`
3) Guarda. Lambda aplicará el reinicio del runtime con las nuevas variables.
4) Repite en las otras funciones (sqs-consumer, crons, etc.) que quieras ajustar.

Opción 3 — AWS CLI (por función, reproducible)
----------------------------------------------

Permite automatizar cambios sin tocar Terraform ni la consola. Repite por cada función objetivo.

1) Exporta las variables actuales a un archivo JSON:

```
aws lambda get-function-configuration \
  --function-name <NOMBRE_FUNCION> \
  --query 'Environment' \
  --output json > env.json
```

2) Edita `env.json` y **agrega/modifica/elimina** claves dentro de `Variables`:

```
{
  "Variables": {
    "...": "...",
    "DB_SLOW_SQL_ENABLED": "true",   // o "false"
    "DB_LOG_LEVEL": "warn",
    "DB_SLOW_THRESHOLD_MS": "500"
  }
}
```

Para **eliminar** una variable, quítala del objeto `Variables` antes de actualizar.

3) Sube la configuración a la función:

```
aws lambda update-function-configuration \
  --function-name <NOMBRE_FUNCION> \
  --environment file://env.json
```

4) Verifica en CloudWatch Logs y, al finalizar la ventana, repite el proceso con `DB_SLOW_SQL_ENABLED="false"`.

Notas y Buenas Prácticas
------------------------

- Mantén el umbral (`DB_SLOW_THRESHOLD_MS`) en 500ms o más en producción para limitar ruido y costos.
- Usa `DB_LOG_LEVEL=warn` para ver errores + “SLOW SQL” únicamente.
- En periodos largos, prefiere la Opción 1 (Terraform) para mantener trazabilidad y consistencia en todas las Lambdas.
