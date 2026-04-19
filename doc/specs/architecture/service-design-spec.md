---
domain: architecture
summary: Convención de diseño para servicios, casos de uso y componentes equivalentes con interfaces pequeñas, constructor injection y receivers sobre implementación concreta.
when_to_read:
  - servicios nuevos
  - refactors estructurales de servicios
  - cambios en interfaces o constructor injection
code_paths:
  - internal/services/
  - internal/adapters/
related_info:
  - doc/info/development/service-design-conventions.md
  - doc/info/development/create-services-steps.md
related_specs:
  - doc/specs/architecture/core-architecture-spec.md
status: active
---

# Service Design Spec

## Objetivo

Formalizar la convención de diseño para servicios del proyecto, de modo que nuevos paquetes y refactors sigan una estructura consistente, testeable y desacoplada.

## Alcance

Aplica a servicios, casos de uso, coordinadores y componentes equivalentes que encapsulan lógica de aplicación o integración.

## Regla principal

Salvo excepción justificada, todo servicio nuevo o refactorizado debe implementarse con:

1. `interface segregation`
2. `constructor injection`
3. `method receivers`
4. implementación concreta no exportada

## Reglas

### 1. Interface segregation

- La interface debe ser pequeña y orientada al contrato consumido.
- No debe agrupar métodos sin cohesión funcional.
- Si dos consumidores necesitan contratos distintos, deben evaluarse interfaces separadas.

### 2. Constructor injection

- Las dependencias del servicio deben declararse en un constructor `New...`.
- No debe esconderse dependencia crítica en variables globales cuando pueda inyectarse.
- El constructor debe devolver la interface del servicio cuando eso mejore encapsulamiento.

### 3. Implementación concreta

- La struct concreta debe ser preferentemente no exportada.
- La implementación debe almacenar dependencias explícitas y solo el estado necesario.
- Los detalles internos del servicio no deben ser el contrato público por defecto.

### 4. Method receivers

- Los métodos del contrato deben implementarse como receivers sobre la struct concreta.
- La lógica del servicio no debe dispersarse en funciones globales cuando pertenece al comportamiento del servicio.

## Excepciones permitidas

Se permite apartarse del patrón solo cuando exista justificación clara, por ejemplo:

- restricciones de framework,
- adaptadores muy pequeños sin valor real en abstraer,
- compatibilidad con código legado donde el costo inmediato de migración sea alto.

En esos casos, la excepción debe seguir buscando:

- bajo acoplamiento,
- dependencias explícitas,
- responsabilidad acotada.

## Acceptance Criteria

- Un servicio nuevo expone un contrato claro y acotado.
- Las dependencias pueden identificarse desde el constructor.
- La implementación concreta no obliga a consumidores a acoplarse con detalles internos.
- La estructura facilita pruebas unitarias y refactors futuros.

## Trazabilidad

- `AGENTS.md`
- `doc/info/development/service-design-conventions.md`
- `doc/info/development/create-services-steps.md`
