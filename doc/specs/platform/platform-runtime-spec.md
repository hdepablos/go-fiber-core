---
domain: platform
summary: Contrato operativo mínimo para desarrollo, configuración y despliegue en modos local, Lambda y EKS, variables de entorno y cambio de modo.
when_to_read:
  - cambios en infraestructura o entornos de despliegue
  - cambios en variables de entorno o configuración runtime
  - cambios en modos de ejecución lambda o eks
  - onboarding de DevOps o cambios en terraform o helm
code_paths:
  - terraform/
  - k8s/
  - docker-compose.yml
  - .env
related_info:
  - doc/info/development/development-workflow.md
  - doc/info/platform/devops-guide.md
  - doc/info/platform/manage-env-vars.md
  - doc/info/platform/eks-prerequisites.md
  - doc/info/platform/hybrid-deployment.md
related_specs:
  - doc/specs/architecture/service-runtime-bootstrap-spec.md
status: active
---

# Platform Runtime Spec

## Objetivo

Definir el contrato operativo minimo para desarrollo, configuracion y despliegue del proyecto en modos locales, Lambda y EKS.

## Alcance

Basado en:

- `doc/info/development/development-workflow.md`
- `doc/info/platform/devops-guide.md`
- `doc/info/platform/manage-env-vars.md`
- `doc/info/platform/eks-prerequisites.md`
- `doc/info/platform/eks-deployment-guide.md`
- `doc/info/platform/hybrid-deployment.md`
- `doc/info/platform/connect-s3.md`

## Reglas

### 1. Modos de ejecucion

- El proyecto debe soportar desarrollo local, simulacion local de infraestructura y despliegue remoto.
- Cada modo debe tener comandos y prerequisitos identificables.

### 2. Variables de entorno

- La gestion de variables debe tener una fuente de verdad clara por entorno.
- Las nuevas variables deben poder propagarse sin tocar multiples capas manualmente cuando exista mecanismo unificado.

### 3. Cambio de modo de despliegue

- Lambda y EKS son modos mutuamente excluyentes para el compute principal del mismo servicio.
- La capa de datos debe permanecer desacoplada del cambio de modo de compute.

### 4. Integraciones externas

- S3 y servicios AWS relacionados deben documentar credenciales, endpoint y comportamiento local vs remoto.

## Acceptance Criteria

- Existe un flujo documentado de setup para cada modo principal.
- El cambio de modo de despliegue esta gobernado por configuracion explicita.
- La documentacion humana y la spec usan la misma taxonomia de runtime/plataforma.
