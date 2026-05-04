---
domain: exports
summary: Contratos y acceptance criteria para pipelines de exportación por lotes con exportmanager, redis, S3, cancelación operativa y auto-cancel.
when_to_read:
  - exports batch nuevos sobre exportmanager
  - cambios en exportmanager.Manager
  - cambios en registro de exports por execution_key
  - cambios en auto-cancel de exports
  - cambios en generación de archivos por lote
code_paths:
  - internal/services/exportmanager/
  - internal/services/exports/bcra/
related_info:
  - doc/info/exports/exportmanager-guide.md
  - doc/info/exports/exportmanager-generar-archivo-banco-galicia.md
related_specs:
  - doc/specs/process-lifecycle/process-lifecycle-runtime-spec.md
status: active
---

# Export Pipelines Spec

## Objetivo

Definir contratos y criterios de aceptacion para pipelines de exportacion basados en lotes, runtime compartido y builders de archivo.

## Alcance

Aplica a:

- `doc/info/exports/exportmanager-guide.md`
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

### 4. Cancelacion operativa y auto-cancel

- Los exports batch basados en `exportmanager.Manager` deben poder participar en cancelacion operativa y auto-cancel por errores repetidos cuando el flujo se ejecute por `execution_key`.
- Si un export batch define `ParentLifecycle.Fail`, el auto-cancel debe poder invocarlo para dejar el padre en estado consistente sin esperar otro mensaje.
- Para eso, el provider del export debe registrar su `Manager` en un registry central resuelto por `execution_key`.
- No debe requerirse editar el consumer por cada export batch nuevo.
- Los pipelines legacy que no usan `exportmanager.Manager` deben quedar documentados explícitamente como excepción hasta ser migrados.

### 5. Filtros y datos

- Los filtros deben normalizarse antes de tocar repositorios o providers.
- Los pipelines no deben duplicar helpers transversales que ya existan como shared utilities.

## Acceptance Criteria

- Cada export tiene un documento humano orientado a operacion y un documento spec orientado a contrato.
- Las implementaciones concretas pueden variar, pero respetan la misma forma general del pipeline.
- La coordinacion entre Redis, almacenamiento y layout queda documentada en una sola fuente normativa.
- Un export batch nuevo sobre `exportmanager` puede enchufarse al auto-cancel registrando su manager por `execution_key`.
