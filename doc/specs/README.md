# Specs para IA y Spec-Driven Development

`doc/specs/` contiene documentos normativos para asistentes, automatizaciones y flujos de Spec-Driven Development.
Estas specs no reemplazan la documentacion humana: la complementan.

## Principios

- `doc/info/` explica el contexto para personas.
- `doc/specs/` define contratos, invariantes y criterios de aceptacion.
- Cada spec debe apuntar a codigo, pruebas o documentos humanos relacionados.
- Ninguna spec debe duplicar una guia operativa completa; debe formalizar reglas verificables.

## Specs Disponibles

### Gobierno documental

- [documentation-governance-spec.md](documentation-governance-spec.md)
- [documentation-defaults-spec.md](documentation-defaults-spec.md)

### Shared

- [shared-utils-spec.md](shared/shared-utils-spec.md)

### Architecture

- [core-architecture-spec.md](architecture/core-architecture-spec.md)
- [service-design-spec.md](architecture/service-design-spec.md)
- [service-runtime-bootstrap-spec.md](architecture/service-runtime-bootstrap-spec.md)

### API

- [http-endpoints-spec.md](api/http-endpoints-spec.md)

### Platform

- [platform-runtime-spec.md](platform/platform-runtime-spec.md)
- [makefile-automation-spec.md](platform/makefile-automation-spec.md)

### Process Lifecycle

- [process-lifecycle-runtime-spec.md](process-lifecycle/process-lifecycle-runtime-spec.md)

### Exports

- [export-pipelines-spec.md](exports/export-pipelines-spec.md)

### Data

- [database-schema-query-spec.md](data/database-schema-query-spec.md)

## Plantilla Recomendada

Toda nueva spec deberia incluir:

1. Objetivo
2. Alcance
3. Contratos de entrada y salida
4. Invariantes
5. Errores esperados
6. Acceptance criteria
7. Trazabilidad a `doc/info`, codigo y pruebas
