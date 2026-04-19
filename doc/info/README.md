# Documentacion para Humanos

`doc/info/` contiene documentacion operacional y de entendimiento para personas.
La estructura se clasifica por dominio para evitar duplicacion y para que cada archivo tenga un rol unico.

## Como usar esta carpeta

- Lee esta carpeta si necesitas contexto funcional, operativo o arquitectonico.
- Busca primero por dominio, no por tipo de archivo.
- Si un tema requiere reglas verificables o contratos para automatizacion, busca su contraparte en `doc/specs/`.
- La convención por defecto para nuevas solicitudes de documentación está formalizada en `../specs/documentation-defaults-spec.md`.

## Arquitectura

- [authentication-providers.md](architecture/authentication-providers.md): estrategia de autenticacion y providers disponibles.
- [configuration-secrets.md](architecture/configuration-secrets.md): origen, resolucion y uso de secretos.
- [redis-locking-strategy.md](architecture/redis-locking-strategy.md): consistencia de cache y bloqueo en Redis.
- [process-architecture-evolution.md](architecture/process-architecture-evolution.md): analisis y lineas de evolucion del motor.

## API

- [http-endpoints-guide.md](api/http-endpoints-guide.md): mapa de endpoints HTTP con ejemplos de request y organizacion Bruno por URL.

## Development

- [development-workflow.md](development/development-workflow.md): niveles de desarrollo y flujo de trabajo local.
- [create-services-steps.md](development/create-services-steps.md): guia de implementacion de servicios/pasos.
- [service-design-conventions.md](development/service-design-conventions.md): patron recomendado para diseñar servicios y casos de uso.
- [service-runtime-and-scaffold.md](development/service-runtime-and-scaffold.md): runtime por contexto y scaffold de export managers sin globals.
- [external-http-service-standard.md](development/external-http-service-standard.md): estándar obligatorio para adapters y requests HTTP externos reutilizables.
- [process-scaffold-and-cleanup.md](development/process-scaffold-and-cleanup.md): crear y eliminar procesos scaffold (batch/export) y convención Bruno genérica.
- [change-var.md](development/change-var.md): cambios de configuracion y notas de soporte relacionadas.

## Platform

- [devops-guide.md](platform/devops-guide.md): operacion de entornos y despliegues.
- [manage-env-vars.md](platform/manage-env-vars.md): convenio unificado de variables de entorno.
- [makefile-guide.md](platform/makefile-guide.md): mapa operativo del `Makefile` por dominios y riesgos.
- [eks-prerequisites.md](platform/eks-prerequisites.md): prerequisitos y setup local para EKS.
- [eks-deployment-guide.md](platform/eks-deployment-guide.md): despliegue y troubleshooting en EKS.
- [hybrid-deployment.md](platform/hybrid-deployment.md): estrategia Lambda vs EKS.
- [connect-s3.md](platform/connect-s3.md): conexion y configuracion S3.

## Process Lifecycle

- [motor-overview.md](process-lifecycle/motor-overview.md): vista general del motor.
- [manager.md](process-lifecycle/manager.md): descripcion del manager.
- [manager-flow.md](process-lifecycle/manager-flow.md): flujo funcional del manager.
- [runtime.md](process-lifecycle/runtime.md): contrato runtime y `ServiceContext` desde perspectiva humana.
- [resolution-and-history.md](process-lifecycle/resolution-and-history.md): versionado, resolucion e historial.
- [usage-guide.md](process-lifecycle/usage-guide.md): como usar el modulo.
- [scenarios.md](process-lifecycle/scenarios.md): escenarios de ejecucion.
- [testing-guide.md](process-lifecycle/testing-guide.md): pruebas funcionales y requests.
- [batch-preview-guide.md](process-lifecycle/batch-preview-guide.md): uso de batch preview, apply_changes local y run filtrado.
- [batch-fanout-guide.md](process-lifecycle/batch-fanout-guide.md): fan-out batch para Lambda/EKS y configuración de steps.
- [dispatch-pacing-guide.md](process-lifecycle/dispatch-pacing-guide.md): dosificación por tandas usando `process_batch` + `auto_invoke` con delay.
- [batch-capacity-and-stress-guide.md](process-lifecycle/batch-capacity-and-stress-guide.md): checklist de capacidad, stress test y filtros de logs para Redis/rate limit.
- [batch-fanout-risks.md](process-lifecycle/batch-fanout-risks.md): riesgos, anti-patrones y alertas del fan-out.
- [sql-cheatsheet.md](process-lifecycle/sql-cheatsheet.md): apoyo SQL para analisis y operacion.
- [advantages.md](process-lifecycle/advantages.md): motivacion y ventajas del enfoque.

## Exports

- [bulk-export-generate-file-v1-async.md](exports/bulk-export-generate-file-v1-async.md): pipeline v1 asincrono.
- [exportmanager-bulkexport-v2.md](exports/exportmanager-bulkexport-v2.md): framework y caso v2.
- [exportmanager-generar-archivo-banco-galicia.md](exports/exportmanager-generar-archivo-banco-galicia.md): implementacion Banco Galicia.

## Data

- [database-model-relations.md](data/database-model-relations.md): modelo relacional, dominios y rutas de join principales.
- [create-migrations.md](data/create-migrations.md): proceso de migraciones.
- [seeders-catalog-items.md](data/seeders-catalog-items.md): seeders de catalogos.
- [locking-select.md](data/locking-select.md): patrones de locking/select.
- [slow-sql.md](data/slow-sql.md): diagnostico de consultas lentas.

## Operations

- [ci-cd-checklist.md](operations/ci-cd-checklist.md): checklist operativo de CI/CD.
- [lambda-optimization.md](operations/lambda-optimization.md): ajustes y buenas practicas Lambda.
- [logs.md](operations/logs.md): uso de logs y trazabilidad.
- [rate-limit-imports.md](operations/rate-limit-imports.md): convenciones y limites para imports.

## Regla de No Duplicacion

- Cada archivo debe resolver una sola necesidad principal.
- Si dos documentos hablan del mismo tema, uno debe ser overview y el otro procedimiento o contrato.
- No dupliques comportamiento normativo en `doc/info`; llevalo a `doc/specs` y referencia esa spec.
