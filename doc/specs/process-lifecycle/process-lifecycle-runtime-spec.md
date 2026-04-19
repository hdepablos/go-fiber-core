---
domain: process-lifecycle
summary: Contrato del runtime del motor process lifecycle, su contexto compartido, resolución de versiones y ejecución general.
when_to_read:
  - cambios en el runtime del motor
  - cambios en execution keys o resolucion de versiones
  - cambios en service context o manager
  - cancelacion manual de corridas
  - auto-cancel por errores repetidos
  - cambios estructurales en process lifecycle
code_paths:
  - internal/services/processlifecycle/
  - internal/services/serviceconfig/
  - internal/handlers/process_lifecycle_handler.go
  - internal/services/batchflow/
related_info:
  - doc/info/process-lifecycle/motor-overview.md
  - doc/info/process-lifecycle/runtime.md
  - doc/info/process-lifecycle/manager-flow.md
related_specs:
  - doc/specs/process-lifecycle/batch-preview-spec.md
  - doc/specs/process-lifecycle/batch-fanout-spec.md
  - doc/specs/process-lifecycle/batch-observability-spec.md
status: active
---

# Process Lifecycle Runtime Spec

## Objetivo

Formalizar el comportamiento esperado del motor `process lifecycle`, sus contextos compartidos y su modelo de ejecucion.

## Alcance

Sintetiza el dominio descrito en:

- `doc/info/process-lifecycle/motor-overview.md`
- `doc/info/process-lifecycle/manager.md`
- `doc/info/process-lifecycle/manager-flow.md`
- `doc/info/process-lifecycle/runtime.md`
- `doc/info/process-lifecycle/resolution-and-history.md`
- `doc/info/process-lifecycle/usage-guide.md`
- `doc/info/process-lifecycle/scenarios.md`
- `doc/info/process-lifecycle/testing-guide.md`
- `doc/info/process-lifecycle/batch-preview-guide.md`
- `doc/info/process-lifecycle/batch-fanout-guide.md`
- `doc/info/process-lifecycle/batch-fanout-risks.md`
- `doc/info/process-lifecycle/sql-cheatsheet.md`
- `doc/info/process-lifecycle/advantages.md`

## Reglas

### 1. Contexto compartido

- Debe existir un contexto de servicio compartido entre pasos.
- Los pasos leen claves de entrada y publican resultados sin acoplamiento directo entre implementaciones.

### 2. Resolucion de version

- La version efectiva del proceso debe resolverse de forma deterministica.
- El sistema debe soportar historial, escenarios y variantes controladas.

### 3. Ejecucion

- El motor debe soportar escenarios secuenciales y asincronos dentro del mismo modelo mental.
- La validacion de precondiciones debe ser declarativa siempre que sea posible.
- Los errores deben poder clasificarse entre regla de negocio, configuracion y fallo tecnico.

### 4. Observabilidad y pruebas

- Los escenarios de prueba deben derivarse de contratos del runtime y de las keys compartidas.
- Debe existir trazabilidad entre escenarios, configuracion y resultado esperado.

### 5. Cancelacion operativa de corridas

- El runtime debe soportar cancelacion operativa de corridas asincronas cuando la ejecucion use un `run_key` o `key_redis` compartido.
- La fuente de verdad de cancelacion debe ser distribuida y compatible con Lambda y EKS; no debe depender de memoria local del proceso.
- Debe existir al menos una forma manual de cancelar una corrida desde la API y una forma operativa equivalente desde comandos.
- Si una corrida fue cancelada, el runtime no debe seguir re-encolando pasos `auto_invoke` ni disparando `next_step` o `finalize` como si siguiera activa.
- La deteccion de cancelacion debe poder ocurrir en workers asincronos y ser consistente aunque existan multiples pods, lambdas o reintentos de SQS.

### 6. Auto-cancel por errores repetidos

- El runtime batch debe poder registrar errores repetidos por fingerprint de error dentro de la misma corrida.
- Cuando el contador supere el umbral configurado, la corrida debe poder marcarse como cancelada automaticamente.
- Cuando el auto-cancel afecte un proceso batch con `ParentLifecycle`, tambien debe ejecutarse `lifecycle.Fail(...)` para que el padre de negocio refleje el estado correspondiente sin depender de un mensaje posterior.
- Los procesos batch que quieran participar de ese auto-cancel inmediato deben registrar su `Manager` en un registry central resuelto por `execution_key`; no debe requerirse editar el consumer por cada proceso nuevo.
- El auto-cancel debe registrar motivo, fingerprint, threshold y contexto suficiente para observabilidad operativa.
- El auto-cancel no debe depender de diferencias de bootstrap entre Lambda y EKS.

## Acceptance Criteria

- Las claves compartidas del runtime tienen una referencia documental estable.
- Los documentos humanos de process lifecycle quedan separados por rol: overview, runtime, escenarios, testing, SQL y ventajas.
- Las futuras implementaciones del motor deben referenciar esta spec antes de introducir nuevas invariantes.
- Una corrida asincrona puede cancelarse manualmente sin seguir generando nuevos mensajes `auto_invoke`.
- El runtime puede auto-cancelar una corrida por exceso de errores repetidos del mismo fingerprint.
