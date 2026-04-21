---
domain: documentation
summary: Comportamiento por defecto para documentar en el repositorio, separación info/specs, convenciones de clasificación por dominio y reglas de vinculación.
when_to_read:
  - ante cualquier solicitud de documentar, redocumentar o reorganizar
  - cuando haya duda sobre si crear info, spec o ambos
  - onboarding de nuevos colaboradores o agentes
code_paths:
  - doc/info/
  - doc/specs/
  - AGENTS.md
related_info:
  - doc/info/README.md
related_specs:
  - doc/specs/documentation-governance-spec.md
status: active
---

# Documentation Defaults Spec

## Objetivo

Definir el comportamiento por defecto que debe seguir cualquier asistente o contributor cuando se le pida documentar algo en este repositorio, para que no sea necesario volver a aclarar la separacion entre `info` y `specs`.

## Alcance

Aplica a solicitudes como:

- "documenta esto"
- "actualiza la documentacion"
- "crea docs del modulo"
- "explica el flujo"
- "ordena la documentacion"

## Regla Principal

Si no se indica lo contrario, toda nueva documentacion del proyecto debe producir dos artefactos complementarios:

1. Un documento en `doc/info/` orientado a personas.
2. Un documento en `doc/specs/` orientado a IA y Spec-Driven Development.

## Rol de Cada Capa

### `doc/info/`

Debe incluir, segun corresponda:

- contexto funcional,
- explicacion de negocio o tecnica,
- pasos operativos,
- ejemplos de uso,
- troubleshooting,
- mapa de decisiones humanas.

### `doc/specs/`

Debe incluir, segun corresponda:

- objetivo,
- alcance,
- contratos de entrada y salida,
- invariantes,
- errores esperados,
- acceptance criteria,
- trazabilidad a codigo, pruebas y documentos humanos.

## Reglas de Implementacion

### 1. Vinculacion

- Si se crea un documento nuevo, deben actualizarse los indices `doc/info/README.md` y `doc/specs/README.md` cuando corresponda.
- Si el cambio afecta la navegacion principal, debe actualizarse `README.md`.

### 2. No duplicacion

- `doc/info/` no debe copiar textualmente una spec completa.
- `doc/specs/` no debe convertirse en una guia operativa extensa.
- Si un tema ya existe, debe ampliarse o referenciarse antes de crear otro archivo paralelo.

### 3. Clasificacion

Los documentos deben ubicarse por dominio y no por capricho de nombre. Los dominios preferidos son:

- `architecture`
- `development`
- `platform`
- `process-lifecycle`
- `exports`
- `data`
- `operations`
- otros dominios solo si el repositorio realmente los necesita

### 4. Excepciones

No es obligatorio crear ambos artefactos solo cuando:

- el archivo es un template funcional y no documentacion,
- el cambio es minimo y solo corrige enlaces o typos,
- ya existe una de las dos capas y solo hace falta completar la otra parcialmente,
- el usuario pide explicitamente solo una capa.

En esos casos debe preservarse la consistencia con la regla general.

## Acceptance Criteria

- Ante una solicitud generica de documentacion, el resultado por defecto incluye `info` y `specs`.
- El contributor no necesita preguntar nuevamente si debe separar documentacion humana y normativa, salvo que el usuario quiera una excepcion.
- Toda nueva documentacion queda enlazada dentro del mapa documental vigente.

## Trazabilidad

- `AGENTS.md`
- `doc/specs/documentation-governance-spec.md`
- `doc/info/README.md`
- `doc/specs/README.md`
