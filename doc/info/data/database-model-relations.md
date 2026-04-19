# Modelo de Datos y Relaciones

Este documento resume las entidades principales del proyecto, sus relaciones y la lógica estructural necesaria para entender el esquema sin relevar el código completo cada vez.
Su objetivo es servir como base humana para análisis funcional, debugging y futuras consultas SQL.

## Fuentes de verdad

- Modelos GORM en `internal/models/`
- Migraciones SQL en `internal/database/migrations/postgres/`
- Specs normativas en `doc/specs/data/database-schema-query-spec.md`

## Dominios principales

### 1. Acceso y autorización

Tablas y modelos clave:

- `users`
- `roles`
- `role_user`
- `menus`
- `menu_user`
- `menu_role`
- `sessions`
- `refresh_tokens`
- `authentication_logs`

Relaciones relevantes:

- `users` ↔ `roles`: relación many-to-many a través de `role_user`
- `users` ↔ `menus`: relación many-to-many a través de `menu_user`
- `roles` ↔ `menus`: relación many-to-many a través de `menu_role`
- `users.operator_id` → `users.id`: relación jerárquica de usuario operador
- `sessions.user_id` → `users.id`
- `refresh_tokens.user_id` → `users.id`
- `authentication_logs.user_id` → `users.id` cuando el evento está asociado a un usuario concreto
- `menus.parent_id` → `menus.id`: jerarquía de menús

Notas:

- `menu_user` y `menu_role` funcionan como pivotes enriquecidos, no solo como tablas puente.
- Varias tablas usan `deleted_at`, por lo que hay soft delete en parte del dominio de acceso.

### 2. Process Lifecycle

Tablas confirmadas por migración:

- `process_types`
- `process_versions`
- `process_steps`
- `process_version_history`

Relaciones principales:

- `process_versions.process_type_id` → `process_types.id`
- `process_steps.process_version_id` → `process_versions.id`
- `process_version_history` mantiene trazabilidad histórica y referencia versiones por clave compuesta

Notas:

- Este dominio modela definición de procesos, versiones publicables y pasos configurables.
- La lógica de resolución de versión y ejecución debe complementarse con la documentación de `process-lifecycle`.

### 3. Bulk Jobs y exportaciones

Tablas confirmadas por modelos y migración:

- `bulk_jobs`
- `bulk_job_outputs`
- `bulk_job_items`
- `bulk_job_item_messages`
- `bulk_job_configs`

Relaciones principales:

- `bulk_job_outputs.bulk_job_id` → `bulk_jobs.id`
- `bulk_job_items.bulk_job_id` → `bulk_jobs.id`
- `bulk_job_item_messages.bulk_job_item_id` → `bulk_job_items.id`
- `bulk_job_configs.operator_id` enlaza con el operador lógico que parametriza la configuración

Notas:

- `bulk_jobs` concentra metadatos del proceso masivo, estado, referencia y operador.
- `bulk_job_items` representa unidades de trabajo por fila o ítem.
- `bulk_job_outputs` representa artefactos o salidas generadas por el proceso.
- `bulk_job_item_messages` conserva mensajes de detalle por ítem.
- Existen enums SQL como `bulk_job_status` y `log_severity`.
- Los contadores operativos de avance deben derivarse de `bulk_job_items` y no de columnas calculadas persistidas en `bulk_jobs`.
- Para listados operativos, `bulk_job_items.status_code = 'IMPORTED'` se considera pendiente; `PROCESSED`, `ERROR_PROCESS` y `PROCESSED_WITH_DETAILS` se reportan por separado.

### 4. Catálogos y maestros

Tablas y modelos visibles:

- `catalogs`
- `banks`

Notas:

- Son dominios de apoyo para lookup, configuración y reglas de negocio.
- `banks` tiene timestamps indexados y soft delete.

## Relación conceptual rápida

```text
users --< role_user >-- roles --< menu_role >-- menus
  |                         |
  |--< menu_user >----------|
  |--< sessions
  |--< refresh_tokens
  |--< authentication_logs
  |
  '-- operator_id --> users

process_types --< process_versions --< process_steps
                       |
                       '--< process_version_history

bulk_jobs --< bulk_job_items --< bulk_job_item_messages
    |
    '--< bulk_job_outputs
```

## Claves e invariantes importantes

- Las tablas pivot de permisos deben tratarse como parte del modelo de autorización, no como detalle incidental.
- Los dominios `process_lifecycle` y `bulk_jobs` tienen valor histórico y operativo, por lo que no deben consultarse solo por ID sin considerar estado y versionado.
- Las relaciones autorreferenciales como `users.operator_id` y `menus.parent_id` requieren cuidado en joins recursivos o jerárquicos.
- Los enums del esquema son parte del contrato y deben reflejarse al generar consultas o filtros.

## Cómo usar este documento para pedir SQL

Cuando necesites una consulta futura, conviene indicar:

- dominio (`auth`, `process_lifecycle`, `bulk_jobs`, etc.),
- tabla principal,
- relaciones necesarias,
- filtros por estado, fecha, operador o usuario,
- columnas esperadas,
- si el resultado es operacional, analítico o de auditoría.

Ejemplo:

> "Genera una SQL para listar `bulk_jobs` con sus `bulk_job_items` en error, filtrando por `operator_id`, rango de fechas y `status_code`."

Con eso, más la spec asociada, ya existe suficiente base documental para derivar una consulta consistente.
