# Go Fiber Core

Proyecto backend en Go con arquitectura modular, soporte para multiples modos de despliegue y un motor de procesos orientado a configuracion.

## Mapa Documental

La documentacion se divide en dos capas complementarias:

- **Humanos**: [doc/info/README.md](doc/info/README.md)
- **IA / Spec-Driven Development**: [doc/specs/README.md](doc/specs/README.md)

## Rutas Recomendadas

### Plataforma y despliegue

- [Flujo de desarrollo](doc/info/development/development-workflow.md)
- [Guia DevOps](doc/info/platform/devops-guide.md)
- [Variables de entorno](doc/info/platform/manage-env-vars.md)
- [Guia del Makefile](doc/info/platform/makefile-guide.md)
- [Prerequisitos EKS](doc/info/platform/eks-prerequisites.md)
- [Despliegue EKS](doc/info/platform/eks-deployment-guide.md)
- [Despliegue hibrido Lambda vs EKS](doc/info/platform/hybrid-deployment.md)

### Arquitectura

- [Autenticacion modular](doc/info/architecture/authentication-providers.md)
- [Configuracion y secretos](doc/info/architecture/configuration-secrets.md)
- [Redis locking strategy](doc/info/architecture/redis-locking-strategy.md)
- [Evolucion del motor de procesos](doc/info/architecture/process-architecture-evolution.md)
- [Convenciones de diseño de servicios](doc/info/development/service-design-conventions.md)
- [Runtime y scaffold de servicios](doc/info/development/service-runtime-and-scaffold.md)
- [Estándar HTTP externo](doc/info/development/external-http-service-standard.md)
- [Scaffold y cleanup de procesos](doc/info/development/process-scaffold-and-cleanup.md)

### API

- [Guia de endpoints HTTP](doc/info/api/http-endpoints-guide.md)

### Process Lifecycle

- [Overview del motor](doc/info/process-lifecycle/motor-overview.md)
- [Runtime](doc/info/process-lifecycle/runtime.md)
- [Uso](doc/info/process-lifecycle/usage-guide.md)
- [Escenarios](doc/info/process-lifecycle/scenarios.md)
- [Testing](doc/info/process-lifecycle/testing-guide.md)
- [Batch Preview](doc/info/process-lifecycle/batch-preview-guide.md)
- [Batch Fanout](doc/info/process-lifecycle/batch-fanout-guide.md)
- [Capacidad y Stress Batch](doc/info/process-lifecycle/batch-capacity-and-stress-guide.md)
- [Riesgos Fanout](doc/info/process-lifecycle/batch-fanout-risks.md)

### Exports

- [Bulk export v1 async](doc/info/exports/bulk-export-generate-file-v1-async.md)
- [ExportManager v2](doc/info/exports/exportmanager-bulkexport-v2.md)
- [Generar archivo Banco Galicia](doc/info/exports/exportmanager-generar-archivo-banco-galicia.md)

### Datos

- [Modelo de datos y relaciones](doc/info/data/database-model-relations.md)

## Regla Editorial

- Toda documentacion para personas debe vivir en `doc/info/`.
- Toda documentacion normativa para IA y SDD debe vivir en `doc/specs/`.
- `README.md` funciona como portal de entrada y no como deposito de detalle.
- La convención por defecto para futuras solicitudes de documentación queda fijada en [documentation-defaults-spec.md](doc/specs/documentation-defaults-spec.md) y en `AGENTS.md`.

## Git Hooks

Este proyecto incluye un hook `post-commit` para simular despliegues locales segun `DEPLOY_MODE`.

```bash
./tools/install-hooks.sh
```
