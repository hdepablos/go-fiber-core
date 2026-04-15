# generar archivo banco galicia

## Objetivo

Scaffold base generado automaticamente para montar un flujo sobre `exportmanager`.

## Modo

- `bulk_jobs funcional (bulk_job_id=2)`

Esto significa que el scaffold de este módulo ya salió con implementación base sobre:

- `bulk_jobs`
- `bulk_job_items`
- `bulk_job_outputs`

## Execution Keys

- `bulk/export/generar_archivo_banco_galicia/start`
- `bulk/export/generar_archivo_banco_galicia/process_batch`
- `bulk/export/generar_archivo_banco_galicia/finalize`

## Config base

- batch_size: 5000
- part_prefix: `exports/bulk_jobs/generar_archivo_banco_galicia`
- redis_ttl_hours: 24
- file: `exports/bank/galicia/manager-galicia`

## Variables que recibe el developer

Todos los servicios funcionales reciben:

- `id`: id del padre de negocio
- `key_redis`: key unica de la corrida

En este caso concreto:

- `id = bulk_job_id`
- ejemplo actual:
  - `bulk_job_id = 2`

## Runtime en Redis

El runtime permite compartir datos entre data/header/body/footer/end.

Ejemplo:

- guardar `total_amount` en header
- leer `total_amount` en footer

La composición real de la key runtime es:

```text
{key_redis}:runtime:{variable}
```

Ejemplo:

```text
run-xxxx:runtime:total_amount
```

## Infraestructura compartida

Este módulo ya no define adapters locales de Redis/S3.

Usa las implementaciones compartidas del framework:

- [redis_cache.go](file:///private/var/www/go-fiber-core/internal/services/exportmanager/redis_cache.go)
- [s3_store.go](file:///private/var/www/go-fiber-core/internal/services/exportmanager/s3_store.go)

Eso evita duplicación entre procesos nuevos generados por el scaffold.

## Qué quedó funcional

- `DataProvider`
  - consulta `bulk_job_items`
  - por defecto no aplica filtros adicionales
  - puede incorporar filtros desde el `config` del step
  - puede incorporar filtros desde `input.filters`
  - cuando una misma key existe en ambos orígenes, `input.filters` sobrescribe al `config`
  - arma batches
  - calcula `total_records`
  - calcula `total_amount`
- `ParentLifecycle`
  - valida `IMPORTED`
  - cambia `bulk_jobs.status_code` a `PROCESSING`
  - cambia `bulk_jobs.status_code` a `PROCESSED`
  - maneja `ERROR_PROCESS`
- `OutputRegistrar`
  - registra el archivo final en `bulk_job_outputs`

## Qué sigue personalizando el developer

- `HeaderBuilder`
- `BodyBuilder`
- `FooterBuilder`

El footer sale por defecto con una línea:

```text
footer
```

Si no se quiere usar footer, el comentario del scaffold indica reemplazarlo por:

```go
return []string{}, nil
```

## Config base del proceso

- batch_size: `5000`
- part_prefix: `exports/bulk_jobs/generar_archivo_banco_galicia`
- redis_ttl_hours: `24`
- file: `exports/bank/galicia/manager-galicia`

## Lógica de filtros

Regla actual del `DataProvider`:

1. Por defecto no aplica ningún filtro extra sobre `bulk_job_items`
2. Si el `config` del step `start` trae `filters`, esos filtros se aplican
3. Si el request trae `input.filters`, también se aplican
4. Si una misma key viene en `config` y en `input`, prevalece `input`

Campos soportados actualmente:

- `bulk_job_items.status_code` o `status_code`
- `bulk_job_items.reference_key` o `reference_key`
- `bulk_job_items.row_number` o `row_number`
- `bulk_job_items.id` o `id`
- `bulk_job_items.bulk_job_id` o `bulk_job_id`

Ejemplo de `config` en el step `start`:

```json
{
  "batch_size": 5000,
  "part_prefix": "exports/bulk_jobs/generar_archivo_banco_galicia",
  "redis_ttl_hours": 24,
  "filters": {
    "status_code": "IMPORTED"
  }
}
```

Ejemplo de request con filtros por `input`:

```json
{
  "input": {
    "id": 2,
    "filters": {
      "reference_key": "ABC123"
    }
  }
}
```

Ejemplo combinado:

```json
{
  "config.filters": {
    "status_code": "IMPORTED",
    "reference_key": "ABC123"
  },
  "input.filters": {
    "status_code": "ERROR_PROCESS",
    "row_number": 10
  }
}
```

Resultado aplicado:

```json
{
  "status_code": "ERROR_PROCESS",
  "reference_key": "ABC123",
  "row_number": 10
}
```

## Request base

El request generado para este caso usa:

```json
{
  "input": {
    "id": 2
  }
}
```

Interpretación:

- `id = 2`
- `bulk_job_id = 2`
- sin `filters`, no se agrega ningún filtro adicional en `bulk_job_items`

## Preview Local

El preview reutiliza los mismos `DataProvider`, `HeaderBuilder`, `BodyBuilder` y `FooterBuilder` del proceso principal.

Endpoint:

```text
POST /api/v1/process-lifecycle/export-preview
```

Body recomendado:

```json
{
  "process_type_id": 17,
  "sede_id": 0,
  "override_process_version_id": 0,
  "roadmap": 0,
  "mode": "all",
  "limit": 20,
  "offset": 0,
  "input": {
    "id": 2,
    "key_redis": "preview-galicia-001",
    "filters": [
      {
        "field": "status_code",
        "operator": "eq",
        "value": "ERROR_PROCESS"
      }
    ]
  }
}
```

Semántica de resolución:

- `override_process_version_id = 0`: usa la versión que el lifecycle resuelva como productiva
- `override_process_version_id > 0`: fuerza esa versión específica aunque no esté en `PROD`
- `roadmap`: se resuelve igual que en `run`

Los requests Bruno quedaron en:

- request canónico: `bruno/api/v1/process-lifecycle/post-export-preview.bru`
- variantes históricas: `bruno/legacy/process-lifecycle/test-export`

## Nota sobre atomicidad en Redis

Redis se usa con operaciones atómicas individuales, pero no con una transacción completa de todo el flujo.

Eso significa:

- `SET`, `GET`, `RPUSH`, `DEL`: atómicos por comando
- el flujo total de runtime/state/cleanup: no transaccional completo

## Siguientes pasos

1. Ajustar DataProvider si necesitas filtros adicionales
2. Personalizar `HeaderBuilder`, `BodyBuilder` y `FooterBuilder`
3. Ejecutar el seeder y probar con Bruno
