---
domain: data
summary: Contrato documental mínimo del esquema relacional para generar SQL, joins y reportes confiables a partir de migraciones, modelos y documentación base.
when_to_read:
  - cambios en migraciones SQL
  - cambios en tablas o columnas
  - cambios en relaciones o integridad
  - solicitudes de reportes o consultas SQL
code_paths:
  - internal/database/migrations/postgres/
  - internal/models/
related_info:
  - doc/info/data/database-model-relations.md
  - doc/info/data/create-migrations.md
related_specs:
  - doc/specs/documentation-governance-spec.md
status: active
---

# Database Schema Query Spec

## Objetivo

Formalizar la documentación mínima de base de datos necesaria para que futuros asistentes o contributors puedan generar SQL confiable a partir de la documentación del repositorio.

## Alcance

Aplica al modelo relacional documentado mediante:

- modelos GORM en `internal/models/`
- migraciones SQL en `internal/database/migrations/postgres/`
- documentación humana en `doc/info/data/database-model-relations.md`

## Regla principal

Toda evolución del esquema que afecte tablas, relaciones o reglas de integridad debe dejar trazabilidad suficiente para responder futuras solicitudes de SQL sin reexplorar todo el código.

## Contrato documental mínimo

Para cada dominio de datos importante debe quedar documentado:

1. Nombre de tablas implicadas.
2. Clave primaria.
3. Claves foráneas y cardinalidad.
4. Tablas pivote o tablas enriquecidas de relación.
5. Campos de estado, versión, auditoría o soft delete.
6. Enums, restricciones o invariantes semánticas relevantes.

## Dominios mínimos cubiertos

### 1. Acceso y autorización

Debe quedar claro:

- cómo se relacionan `users`, `roles` y `menus`,
- qué pivotes existen,
- qué relaciones son jerárquicas,
- qué tablas soportan sesión, refresh token y auditoría de autenticación.

### 2. Process Lifecycle

Debe quedar claro:

- cómo se relacionan `process_types`, `process_versions`, `process_steps` y `process_version_history`,
- qué partes del modelo representan definición, versión vigente e historial,
- qué consultas deben considerar versionado y trazabilidad.

### 3. Bulk Jobs

Debe quedar claro:

- relación entre `bulk_jobs`, `bulk_job_items`, `bulk_job_outputs`, `bulk_job_item_messages` y `bulk_job_configs`,
- significado de estados y artefactos,
- rutas de join para análisis operativo y exportación.

## Reglas para generación futura de SQL

Cuando se pida SQL a partir de esta documentación, la respuesta debe:

- partir de la tabla principal pedida por el usuario,
- respetar claves foráneas y cardinalidades documentadas,
- considerar `deleted_at` cuando aplique soft delete,
- considerar enums o estados como parte del contrato,
- explicitar joins cuando el dominio tenga pivotes o relaciones jerárquicas,
- distinguir entre consulta operacional, auditoría o analítica si el contexto lo requiere.

## Reglas de mantenimiento

- Si una migración agrega o cambia una relación, debe actualizarse la documentación `info + specs`.
- Si un modelo GORM expresa relaciones no evidentes en SQL, eso también debe quedar documentado.
- Si aparece un nuevo dominio de datos relevante, debe añadirse al mapa documental e índice correspondiente.

## Acceptance Criteria

- Un asistente puede responder consultas SQL razonables usando la documentación sin inspección exhaustiva adicional.
- La documentación permite identificar relaciones principales y rutas de join por dominio.
- Los cambios de esquema no quedan solo en migraciones o modelos; también quedan reflejados en documentación humana y normativa.

## Trazabilidad

- `AGENTS.md`
- `doc/info/data/database-model-relations.md`
- `internal/models/`
- `internal/database/migrations/postgres/`
