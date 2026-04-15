# Export Pipelines Spec

## Objetivo

Definir contratos y criterios de aceptacion para pipelines de exportacion basados en lotes, runtime compartido y builders de archivo.

## Alcance

Aplica a:

- `doc/info/exports/bulk-export-generate-file-v1-async.md`
- `doc/info/exports/exportmanager-bulkexport-v2.md`
- `doc/info/exports/exportmanager-generar-archivo-banco-galicia.md`
- `doc/info/platform/connect-s3.md`

## Reglas

### 1. Pipeline por lotes

- Todo pipeline debe definir inicio, procesamiento por lote y finalizacion o equivalente.
- El runtime compartido debe exponer claves suficientes para coordinar Redis, S3 y estado del proceso.

### 2. Builders de salida

- Header, body y footer deben tener contratos separables.
- El formato final debe ser reproducible a partir del input y de las reglas de layout.

### 3. Persistencia y coordinacion

- Las keys de Redis deben tener un proposito explicito.
- Los artefactos parciales y finales en S3 deben seguir una nomenclatura estable.

### 4. Filtros y datos

- Los filtros deben normalizarse antes de tocar repositorios o providers.
- Los pipelines no deben duplicar helpers transversales que ya existan como shared utilities.

## Acceptance Criteria

- Cada export tiene un documento humano orientado a operacion y un documento spec orientado a contrato.
- Las implementaciones concretas pueden variar, pero respetan la misma forma general del pipeline.
- La coordinacion entre Redis, almacenamiento y layout queda documentada en una sola fuente normativa.
