# Ventajas de la arquitectura Process Lifecycle (y por qué escala bien)

Este documento resume las ventajas de trabajar con la estructura del **Process Lifecycle Manager** y el **motor de ejecución de servicios** del proyecto.

## 1) Cambios de flujo sin redeploy (orquestación dinámica)

El flujo de negocio (qué pasos se ejecutan y en qué orden) vive en base de datos (`process_steps`), no “hardcodeado” en el backend.

- Permite ajustar un proceso (agregar/quitar pasos, reordenar, cambiar parámetros) sin desplegar código.
- Reduce el “time-to-change” para negocio/QA cuando el cambio es de configuración.
- Mantiene el código estable: el motor ejecuta y los servicios son piezas reutilizables.

Relacionado: `execution_key`, `step_order` y construcción del registro desde BD.

## 2) Versionado real y gobernanza del cambio

El modelo `DRAFT → TEST → PROD → HISTORY` habilita un ciclo de vida controlado:

- Evita “editar producción” directamente.
- Permite iterar en borrador, validar en pruebas y promover de forma consistente.
- Conserva histórico: puedes reconstruir qué versión estuvo activa y cuándo.
- Obliga a registrar “quién/cuándo/por qué” en promociones (comentario obligatorio).

Relacionado: `promote_process_version`, `replicate_process_version`, `resolve_process_version` y `process_version_history`.

## 3) Resolución por sede + fallback global (multi-tenant práctico)

El mismo tipo de proceso puede variar por `sede_id`, con una regla simple:

- Si existe versión `PROD` para la sede, se usa.
- Si no existe, se usa la versión global (`sede_id = NULL`).
- Si no hay global, el error es explícito (“falta configuración”).

Ventajas:

- Permite customizaciones locales sin duplicar todo el sistema.
- Mantiene una base común global y solo “sobrescribes” donde hace falta.

Relacionado: jerarquía de resolución y matriz de comportamiento.

## 4) Paralelismo por diseño (performance sin complejidad accidental)

`step_order` define una semántica clara:

- Steps con el mismo `step_order` se ejecutan en paralelo.
- Se espera a que termine el grupo para avanzar al siguiente orden.

Ventajas:

- Acelera procesos con pasos independientes.
- Permite aprovechar CPU sin inventar orquestadores externos.

Relacionado: `ExecuteServicesInOrder` y agrupación por `Order`.

## 5) Contratos claros: Input compartido + Results por servicio

El motor trabaja con dos estructuras centrales:

- `ServiceContext.Input`: bolsa de datos global del proceso.
- `ServiceContext.Results`: resultados indexados por `execution_key`.

Ventajas:

- Facilita composición: un paso puede enriquecer el input para el siguiente.
- Mejora trazabilidad: cada servicio puede dejar `StepResult` con `Data` y snapshot de input.
- Simplifica consumo: `GetAll()` devuelve un mapa plano de salidas relevantes.

Relacionado: `ServiceContext`, `StepResult`, `GetAll()`.

## 6) Validación declarativa de precondiciones (`required_keys`)

Cada step puede declarar qué claves necesita antes de ejecutar.

Ventajas:

- Menos “if nil” repetidos en cada servicio.
- Errores consistentes (mismo tipo de error y mensaje).
- Integración más segura: si el input está incompleto, se detecta temprano y de forma uniforme.

Relacionado: validación de `required_keys` en el executor.

## 7) Manejo de errores configurable (resiliencia controlada)

El motor combina:

- Tipo de error de dominio (`ErrCritical`, `ErrTolerable`, etc.).
- Política por step (`error_tolerance`: `critical | tolerable | inherit`).

Ventajas:

- Permite que el proceso continúe cuando el fallo es aceptable (por negocio).
- Mantiene “fail fast” cuando el error es crítico.
- Reduce acoplamiento: la política se define por config, no por código.

Relacionado: reglas de `error_tolerance` y catálogo de errores.

## 8) Timeouts por step (protección contra pasos lentos)

Cada step puede definir `timeout_ms`.

Ventajas:

- Evita que un paso bloquee todo el proceso indefinidamente.
- Protege la API/runtime de degradaciones por llamadas lentas.
- Hace el rendimiento “configurable” sin tocar el servicio.

Relacionado: timeouts por step.

## 9) Sync y Async con el mismo modelo mental (un solo motor)

El motor soporta escenarios:

- Secuenciales SYNC.
- Paralelos ASYNC con “batching”.
- Flujos híbridos SYNC → ASYNC → SYNC.

Ventajas:

- La misma estructura sirve para validaciones rápidas y procesos pesados en background.
- Permite evolucionar un proceso (de sync a async) sin reescribirlo completo.

Relacionado: escenarios (`execution_policy`) y guía de pruebas.

## 10) Observabilidad y modo test (medir sin ensuciar producción)

Cuando se ejecuta con override (modo test), se habilitan métricas detalladas.

Ventajas:

- Diagnóstico y tuning de performance sin overhead permanente.
- Facilita QA y debugging: puedes comparar duraciones por step y costos de DB.

Relacionado: “Modo Test y Métricas de Rendimiento”.

