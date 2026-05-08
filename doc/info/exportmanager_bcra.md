# generar archivo BCRA

## Objetivo

Scaffold base generado automaticamente para montar un flujo sobre `exportmanager`.

## Modo

- `bulk_jobs funcional (bulk_job_id=1)`
- `item-oriented` por contrato: cada registro del batch se transforma desde `BodyBuilder.renderItem(...)`

## Execution Keys

- `bulk/export/bcra/start`
- `bulk/export/bcra/process_batch`
- `bulk/export/bcra/finalize`

## Config base

- batch_size: 5000
- part_prefix: `exports/bulk_jobs/bcra`
- redis_ttl_hours: 24
- file: `exports/bcra`

## Variables que recibe el developer

Todos los servicios funcionales reciben:

- `id`: id del padre de negocio
- `key_redis`: key unica de la corrida

## Runtime en Redis

El runtime permite compartir datos entre data/header/body/footer/end.

Ejemplo:

- guardar `total_amount` en header
- leer `total_amount` en footer

## Layout por defecto

- `header`, `body` y `footer` salen funcionales desde el scaffold
- usa CSV con separador `;`
- incluye columnas históricas y cálculo de `new_importe`
- concentra helpers compartidos en `layout/layout_helpers.go`

## BodyBuilder

- `BuildBodyLines(...)` mantiene el contrato del framework
- `renderItem(...)` es el punto de extension recomendado para el developer
- el preview reutiliza este mismo camino de render por item

## Siguientes pasos

1. Ajustar DataProvider si necesitas filtros adicionales
2. Ajustar `layout/header_builder.go`, `layout/body_builder.go`, `layout/footer_builder.go` y `layout/layout_helpers.go` si el formato final difiere del default
3. Ejecutar el seeder y probar con Bruno
