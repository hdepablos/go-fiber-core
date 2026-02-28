
## Despliegue e Infraestructura (EKS/Hybrid)

Documentación relacionada con el despliegue en Kubernetes (EKS), uso de LocalStack y la estrategia híbrida (Lambda + EKS).

- [📍 GUÍA PRINCIPAL: Flujo de Desarrollo (3 Niveles)](doc/development-workflow.md)
- [Guía de Despliegue en EKS (Local con OrbStack)](doc/eks-migration/01-deployment-guide.md)
- [Estrategia de Despliegue Híbrido: Lambda vs EKS](doc/eks-migration/02-hybrid-deployment.md)
- [📦 Gestión de Variables de Entorno (Unificada)](doc/manage-env-vars.md)

## Arquitectura Modular

Documentación sobre los patrones de diseño implementados para soportar múltiples entornos y proveedores.

- [🔐 Autenticación Modular (Local vs Cognito)](doc/architecture/authentication-providers.md)
- [🔑 Gestión de Secretos (Env vs AWS Secrets Manager)](doc/architecture/configuration-secrets.md)

## Git Hooks

This project includes a `post-commit` hook that automatically deploys changes to your local environment (Lambda or EKS) based on the `DEPLOY_MODE` variable in `.env`.

To install the hooks manually (if not already installed):
```bash
./tools/install-hooks.sh
```

