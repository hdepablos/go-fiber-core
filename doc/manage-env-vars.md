# Gestión de Variables de Entorno

Este documento explica cómo añadir y gestionar variables de entorno en el proyecto para asegurar que funcionen correctamente en todos los modos de despliegue (Local Go, Lambda, y EKS).

## Estrategia Unificada

Hemos centralizado la gestión de variables de entorno para que solo tengas que editarlas en dos lugares, independientemente de dónde despliegues.

### Pasos para añadir una nueva variable

Si necesitas añadir una nueva variable (ej: `NUEVA_API_KEY`), sigue estos pasos:

#### 1. Entorno Local (Desarrollo en Go)
Para que tu código funcione cuando ejecutas `make dev-local` o `go run`, añade la variable a tu archivo `.env` en la raíz del proyecto.

```bash
# Archivo: .env
NUEVA_API_KEY=valor_secreto_local
```

#### 2. Infraestructura (Lambda y EKS)
Para que la variable se inyecte automáticamente en AWS Lambda y en los Pods de Kubernetes (EKS/LocalStack), añádela al mapa `app_env_vars` en el archivo `terraform/local.tfvars`.

```hcl
# Archivo: terraform/local.tfvars

app_env_vars = {
  # ... variables existentes ...
  APP_ENV      = "local"
  PROJECT_SLUG = "go-fiber-core"
  
  # TU NUEVA VARIABLE
  NUEVA_API_KEY = "valor_secreto_infra"
}
```

**¡Y listo!** No necesitas editar `main.tf`, ni `variables.tf`, ni los charts de Helm.

---

## ¿Cómo funciona bajo el capó?

La magia ocurre en Terraform gracias a un mapa dinámico:

1.  **Definición:** En `terraform/variables.tf` existe una variable tipo mapa llamada `app_env_vars`.
2.  **Asignación:** En `terraform/local.tfvars` asignamos todos los valores a ese mapa.
3.  **Inyección Automática:**
    *   **Lambda:** El módulo de Lambda recibe `var.app_env_vars` y las pasa como variables de entorno nativas.
    *   **EKS (Helm):** El recurso `helm_release` fusiona `var.app_env_vars` con configuraciones específicas de EKS (como `host.docker.internal`) y las pasa al chart, que genera el ConfigMap/Deployment correspondiente.

## Producción

En un entorno de producción real, **NO** debes usar `local.tfvars` para secretos.
En su lugar, Terraform debería recibir estas variables desde:
*   Variables de entorno de CI/CD (GitHub Secrets).
*   AWS Secrets Manager (referenciados por ARN).
*   Parameter Store.

Sin embargo, la estructura de `app_env_vars` se mantiene, solo cambia el origen de los datos.
