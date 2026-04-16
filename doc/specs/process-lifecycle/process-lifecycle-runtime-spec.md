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

## Acceptance Criteria

- Las claves compartidas del runtime tienen una referencia documental estable.
- Los documentos humanos de process lifecycle quedan separados por rol: overview, runtime, escenarios, testing, SQL y ventajas.
- Las futuras implementaciones del motor deben referenciar esta spec antes de introducir nuevas invariantes.
