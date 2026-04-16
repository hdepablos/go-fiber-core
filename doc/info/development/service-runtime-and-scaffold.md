# Runtime de Servicios y Scaffold sin Globals

Este documento explica el cambio reciente en el runtime de servicios y en el scaffold de export managers para eliminar dependencias globales y alinear el proyecto con las convenciones tecnicas de servicios.

## Objetivo

Antes, parte del runtime resolvia dependencias con estado global compartido:

- `DefaultDispatcher`
- `DefaultProvider`
- `SetDefaultProvider`

Ese enfoque permitia arrancar rapido, pero hacia menos explicito el flujo de dependencias y dificultaba pruebas, refactors y ejecuciones concurrentes.

El objetivo del cambio fue pasar a una resolucion explicita por contexto de ejecucion.

## Que cambio

### 1. Runtime por `context.Context`

El runtime ahora puede transportar dependencias operativas mediante contexto:

- dispatcher
- providers de exportacion
- otros valores runtime nombrados

Piezas principales:

- [runtimectx.go](file:///private/var/www/go-fiber-core/internal/services/runtimectx/runtimectx.go)
- [bootstrap.go](file:///private/var/www/go-fiber-core/internal/runtimebootstrap/bootstrap.go)

### 2. Bootstrap compartido

Se agrego un bootstrap comun para construir dependencias runtime una sola vez y luego inyectarlas en:

- requests HTTP
- ejecuciones desde SQS
- flujos futuros que necesiten el mismo runtime

Esto evita que cada entrypoint vuelva a construir su propia logica ad hoc.

### 3. Executor sin dispatcher global

El executor de `serviceconfig` ya no usa un dispatcher global.

Ahora:

- busca el dispatcher en el contexto,
- falla explicitamente si no esta disponible,
- y mantiene el flujo de `ASYNC` sin depender de singletons.

Archivo relacionado:

- [executor.go](file:///private/var/www/go-fiber-core/internal/services/serviceconfig/executor.go)

### 4. Providers de exportacion sin `DefaultProvider`

Los providers de exportacion dejaron de resolver dependencias con defaults globales.

Ahora cada provider expone:

- `WithProvider(ctx, prov)`
- `ProviderFromContext(ctx)`

Eso aplica a:

- [provider.go](file:///private/var/www/go-fiber-core/internal/services/test/bulkexportv1/provider.go)
- [provider.go](file:///private/var/www/go-fiber-core/internal/services/test/bulkexportV2/provider.go)
- [provider.go](file:///private/var/www/go-fiber-core/internal/services/generar_archivo_banco_galicia/provider.go)

### 5. Step services consumen dependencias del contexto

Los steps que antes hacian lookup de defaults globales ahora resuelven el provider desde `ServiceContext.Ctx`.

Eso hace que:

- la ejecucion sea mas predecible,
- la dependencia quede visible,
- y los errores de bootstrap sean mas faciles de diagnosticar.

## Flujo actual

### HTTP

1. El entrypoint crea el servidor y sus dependencias.
2. El bootstrap runtime construye dispatcher y providers.
3. Un middleware inyecta esas dependencias en `UserContext`.
4. Los handlers y servicios ejecutan con ese contexto enriquecido.

Archivo relacionado:

- [main.go](file:///private/var/www/go-fiber-core/cmd/api/main.go)

### SQS

1. El consumer inicializa contenedor y dependencias.
2. El bootstrap runtime crea el paquete de dependencias compartidas.
3. Cada mensaje se ejecuta con un contexto enriquecido.
4. Los steps y servicios resuelven providers y dispatcher desde ese contexto.

Archivo relacionado:

- [main.go](file:///private/var/www/go-fiber-core/cmd/sqs-consumer/main.go)

## Impacto en el scaffold

El scaffold de export managers y el scaffold de batch processes fueron actualizados para que nuevo codigo no reintroduzca el patron viejo.

Archivo relacionado:

- [main.go](file:///private/var/www/go-fiber-core/cmd/tools/export-manager-scaffold/main.go)
- [main.go](file:///private/var/www/go-fiber-core/cmd/tools/batch-process-scaffold/main.go)
- [main.go](file:///private/var/www/go-fiber-core/cmd/tools/process-cleanup/main.go)

Ahora el scaffold genera:

- provider con resolucion por contexto
- `dataProvider` no exportado
- `headerBuilder`, `bodyBuilder`, `footerBuilder` no exportados
- `parentLifecycle` y `outputRegistrar` no exportados
- constructores que devuelven interfaces del framework `exportmanager`

Para `batch-process`, el scaffold tambien:

- registra el provider en `runtimebootstrap`,
- deja el proceso preparado para usar la carpeta Bruno genérica `test-batch-process`,
- y evita crear una carpeta Bruno específica por proceso.

Adicionalmente existe un comando de cleanup para revertir el scaffold de procesos siguiendo el mismo patrón.

## Beneficios practicos

- menos estado global compartido
- mejor testabilidad
- menor acoplamiento entre runtime y servicios
- mejor trazabilidad de dependencias
- menor riesgo de contaminacion entre ejecuciones
- scaffold mas alineado con la arquitectura real

## Relacion con otras piezas

Este cambio complementa:

- `AGENTS.md`
- `doc/info/development/service-design-conventions.md`
- `doc/specs/architecture/service-design-spec.md`
- `doc/info/process-lifecycle/runtime.md`
- `doc/info/exports/exportmanager-bulkexport-v2.md`
- `doc/info/exports/exportmanager-generar-archivo-banco-galicia.md`

Si en el futuro se agrega otro runtime especial o un nuevo scaffold, debe seguir la misma idea: dependencias explicitas, sin globals de negocio.
