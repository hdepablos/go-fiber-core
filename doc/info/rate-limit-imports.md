# Rate Limit por Grupo (Global vs Imports) + Convención X-Client-Code

## Objetivo

Evitar errores HTTP 429 (Too Many Requests) manteniendo:

- Un rate limit global “justo” para proteger toda la API.
- Un rate limit más alto y específico para endpoints de importación (`/api/v1/imports/...`).
- Cuotas separadas por tipo de cliente usando `X-Client-Code`.

## Rate Limit por Grupo de Rutas

Se aplican 2 límites con una sola estrategia:

- **Global**: aplica a todas las rutas (incluye login, endpoints normales, etc.).
- **Imports**: aplica solo a rutas que empiezan con:
  - `/api/v1/imports/`

Internamente, los contadores se separan por “scope” en Redis:

- `...:rate-limit:global:<X-Client-Code>`
- `...:rate-limit:imports:<X-Client-Code>`

De esta forma, un upload masivo no “consume” el cupo global del resto de la aplicación.

## Variables de Entorno

- `RATE_LIMIT_GLOBAL_PER_MINUTE=100`
- `RATE_LIMIT_IMPORTS_PER_MINUTE=5000`

Estas variables viven en:

- [.env-example](file:///private/var/www/go-fiber-core/.env-example)
- [.env](file:///private/var/www/go-fiber-core/.env)

## Convención `X-Client-Code` (Clientes Reales)

La cabecera `X-Client-Code` identifica el “cliente”/consumidor (no el usuario) para separar cupos.

- **Quasar (login)**: `quasar-login`
- **Quasar (usuarios logueados)**: `quasar`
- **Bruno**: `bruno`
- **Cron/importador**: `cron`
- **Bash (script local)**: `bash` (si querés separar pruebas manuales del cron real)

Nota: si todos usan el mismo `X-Client-Code`, todos comparten el mismo cupo.

## Cómo Evitar 429 en Imports (Archivos Grandes)

La cantidad de requests de un import por chunks se puede estimar y controlar.

### Cálculo de Requests

Si el CSV tiene:

- `total_lines` = total de líneas del archivo (incluye header)
- `batch` = cantidad de filas de datos por chunk

Entonces:

- `data_rows = total_lines - 1`
- `chunks = ceil(data_rows / batch)`
- Requests aproximados del flujo:
  - `1` login
  - `chunks` uploads
  - `1` logout

Total aproximado:

- `requests_total ≈ chunks + 2`

En imports, lo relevante es `chunks` (porque es lo que más crece).

### Regla de Oro (Tiempo mínimo)

Si tu límite es `RATE_LIMIT_IMPORTS_PER_MINUTE = L`, el tiempo mínimo teórico para completar `chunks` requests sin 429 es:

- `min_seconds ≈ (chunks / L) * 60`

Ejemplo:

- `chunks = 120`
- `L = 100/min`
- `min_seconds ≈ 72s`

Si intentás hacerlo más rápido, vas a chocar con 429.

### Estrategia Recomendada: “Throttle + Retry”

La mejor práctica es:

1) **Throttle** (enviar más lento), por ejemplo dormir 0.2–1s entre requests.
2) **Retry ante 429** esperando la ventana del rate limit y reintentando.

Esto evita que el proceso se corte a mitad por un pico de tráfico y permite que continúe.

## Script Bash (upload_imports.sh)

El script de carga en lotes:

- Divide el CSV por `batch` manteniendo el header.
- Hace login, toma el token y sube cada chunk.
- Usa `X-Client-Code` para aislar el cupo del import.
- Si recibe 429, espera y reintenta (para continuar el proceso).

Archivo:

- [upload_imports.sh](file:///private/var/www/go-fiber-core/tests/upload_imports.sh)

