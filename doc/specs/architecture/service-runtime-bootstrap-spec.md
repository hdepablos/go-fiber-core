# Service Runtime Bootstrap Spec

## Objetivo

Formalizar las reglas del runtime de servicios y del scaffold de export managers para evitar reintroducir patrones globales y mantener consistencia con las convenciones tecnicas del proyecto.

## Alcance

Aplica a:

- runtime de ejecucion de servicios
- bootstrap HTTP y SQS
- resolucion de dispatcher y providers de exportacion
- scaffold `cmd/tools/export-manager-scaffold`

## Regla principal

Las dependencias runtime de servicios no deben resolverse mediante estado global mutable cuando puedan inyectarse explicitamente por constructor o por `context.Context`.

## Reglas

### 1. Dispatcher runtime

- El dispatcher no debe exponerse como singleton global operativo.
- La ejecucion `ASYNC` debe resolver el dispatcher desde el contexto de ejecucion.
- Si el dispatcher requerido no esta disponible, el error debe ser explicito y temprano.

### 2. Providers de exportacion

- Los providers no deben depender de `DefaultProvider` o `SetDefaultProvider`.
- Cada provider runtime debe poder inyectarse en `context.Context`.
- La lectura del provider debe hacerse mediante una funcion explicita del paquete, por ejemplo `ProviderFromContext`.

### 3. Bootstrap compartido

- Los entrypoints que ejecutan servicios deben usar un bootstrap compartido cuando reutilicen el mismo conjunto de dependencias runtime.
- HTTP, SQS y otros ejecutores equivalentes deben inyectar el runtime antes de delegar al flujo de negocio.

### 4. Scaffold

- El scaffold de export managers no debe generar defaults globales ni service locators.
- El scaffold debe generar providers resolubles por contexto.
- El scaffold debe generar implementaciones concretas no exportadas cuando no haya una razon fuerte para exportarlas.
- Los constructores del scaffold deben devolver interfaces del framework `exportmanager` cuando exista un contrato claro.

### 5. Coherencia arquitectonica

- La resolucion runtime no debe contradecir las convenciones generales de servicios del repositorio.
- Las dependencias deben quedar trazables desde bootstrap hasta step o servicio consumidor.

## Invariantes

- El runtime productivo no depende de `DefaultDispatcher`.
- Los providers productivos no dependen de `DefaultProvider`.
- Los steps resuelven providers a partir del contexto de ejecucion.
- El scaffold no genera codigo nuevo con globals operativos de runtime.

## Acceptance Criteria

- Existe bootstrap runtime compartido para HTTP y SQS o equivalente.
- El executor asíncrono falla con error claro si no se inyectó dispatcher.
- Los providers de exportación pueden inyectarse y resolverse desde contexto.
- El scaffold genera código alineado con el patrón de contexto e implementaciones no exportadas.
- El proyecto compila sin reintroducir globals operativos en el runtime principal.

## Trazabilidad

- `AGENTS.md`
- `doc/info/development/service-runtime-and-scaffold.md`
- `doc/info/development/service-design-conventions.md`
- `internal/services/runtimectx/runtimectx.go`
- `internal/runtimebootstrap/bootstrap.go`
- `internal/services/serviceconfig/executor.go`
- `cmd/api/main.go`
- `cmd/sqs-consumer/main.go`
- `cmd/tools/export-manager-scaffold/main.go`
