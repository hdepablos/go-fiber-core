---
domain: documentation
summary: Estructura documental oficial del repositorio, reglas de ubicación, clasificación, no duplicación y vinculación entre doc/info/ y doc/specs/.
when_to_read:
  - cambios en estructura de carpetas documentales
  - creación de nuevos dominios documentales
  - reorganización de índices o mapa documental
  - cuando haya duda sobre dónde ubicar un documento nuevo
code_paths:
  - doc/info/
  - doc/specs/
  - README.md
related_info:
  - doc/info/README.md
related_specs:
  - doc/specs/documentation-defaults-spec.md
status: active
---

# Documentation Governance Spec

## Objetivo

Definir la estructura documental oficial del repositorio para evitar dispersion, duplicacion y ambiguedad entre documentacion humana y documentacion para IA.

## Alcance

Aplica a todo documento Markdown documental del repositorio, con estas excepciones:

- `README.md` en raiz: portal de entrada del proyecto.
- Archivos Markdown funcionales que no son documentacion, por ejemplo templates de email.
- READMEs locales de modulos solo cuando describen un subsistema aislado y no deben formar parte del mapa global.

## Reglas Estructurales

### 1. Ubicacion

- La documentacion para personas debe vivir en `doc/info/`.
- La documentacion normativa para IA y SDD debe vivir en `doc/specs/`.
- No deben existir `.md` documentales sueltos en `doc/` fuera de `info` o `specs`.

### 2. Clasificacion

Los documentos de `doc/info/` se organizan por dominio:

- `architecture`
- `development`
- `platform`
- `process-lifecycle`
- `exports`
- `data`
- `operations`

Los documentos de `doc/specs/` se organizan por capability o dominio normativo.

### 3. No duplicacion

- Cada archivo debe tener un proposito unico y explicito.
- Un overview no debe repetir un procedimiento completo.
- Un procedimiento no debe repetir un contrato normativo que ya existe en `doc/specs`.
- Si un tema necesita dos documentos, deben diferenciarse por rol: overview, how-to, runtime, spec, testing, troubleshooting.

### 4. Vinculacion

- `README.md` debe enlazar a `doc/info/README.md` y `doc/specs/README.md`.
- `doc/info/README.md` debe funcionar como indice humano.
- `doc/specs/README.md` debe funcionar como indice normativo para IA.
- La convención por defecto para futuras solicitudes de documentación debe quedar documentada en `AGENTS.md` y en `doc/specs/documentation-defaults-spec.md`.
- Toda spec debe enlazar al menos una fuente humana o tecnica asociada.

## Acceptance Criteria

- No quedan `.md` documentales fuera de `doc/info/`, `doc/specs/` o `README.md` raiz.
- El repositorio tiene un indice claro para humanos y otro para IA.
- Cada documento puede clasificarse en un unico dominio principal.
- Los enlaces del `README.md` principal apuntan a la estructura vigente.
