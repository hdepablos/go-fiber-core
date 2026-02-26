# Estrategia de Despliegue Híbrido: Lambda vs EKS

Este documento describe la arquitectura de despliegue híbrido implementada en el proyecto `GoFiberCore`. Esta estrategia permite desplegar la aplicación tanto en **AWS Lambda** (modelo Serverless clásico) como en **Amazon EKS** (modelo Kubernetes moderno) utilizando la misma base de código y configuración de Terraform.

---

## 1. Concepto y Motivación

El objetivo es tener flexibilidad total en la infraestructura:
*   **Modo Lambda**: Ideal para entornos de bajo tráfico, costos reducidos en idle, y gestión mínima. Es el modelo "clásico" del proyecto.
*   **Modo EKS**: Ideal para cargas de trabajo constantes, control total sobre el runtime (contenedores de larga duración), y aprovechamiento del ecosistema de Kubernetes (KEDA, Service Mesh, etc.).

La implementación utiliza **Terraform** como orquestador principal. Una única variable (`deploy_mode`) decide qué recursos se crean y cuáles se destruyen.

---

## 2. Configuración de Terraform

### Variable `deploy_mode`

Se ha introducido una nueva variable en `terraform/variables.tf`:

```hcl
variable "deploy_mode" {
  description = "Modo de despliegue: 'lambda' (por defecto) o 'eks'"
  type        = string
  default     = "lambda"
}
```

### Lógica Condicional (`count`)

En `terraform/main.tf`, los recursos se crean condicionalmente basándose en esta variable:

*   **Módulos Lambda**: Tienen `count = var.deploy_mode == "lambda" ? 1 : 0`. Si el modo es `eks`, estos recursos se omiten.
*   **Release de Helm (EKS)**: Tiene `count = var.deploy_mode == "eks" ? 1 : 0`. Si el modo es `lambda`, este recurso se omite.

Esto garantiza que **nunca** tengas duplicidad de recursos de cómputo (Lambda + Pods) consumiendo la misma cola SQS o exponiendo la misma API, evitando condiciones de carrera y costos innecesarios.

---

## 3. Estructura de Archivos

Se han añadido/modificado los siguientes archivos clave:

*   `terraform/charts/gofiber-app/`: **Nuevo**. Contiene un Helm Chart genérico diseñado para desplegar cualquier servicio de este proyecto (API, Workers) en Kubernetes. Incluye plantillas para `Deployment`, `Service` y `ScaledObject` (KEDA).
*   `terraform/provider.tf`: **Modificado**. Ahora incluye los proveedores `helm` y `kubernetes`. En entorno local (`environment = "local"`), se configuran automáticamente para usar el contexto de **OrbStack**.
*   `terraform/main.tf`: **Modificado**. Orquesta la lógica híbrida.

---

## 4. Comandos de Despliegue (Makefile)

Para facilitar el cambio entre modos, se han añadido comandos específicos en el `Makefile` raíz.

### 4.1. Desplegar en Modo EKS (Kubernetes)

Este comando configura Terraform con `deploy_mode=eks`. Desplegará los Charts de Helm en tu cluster local (OrbStack) y eliminará las funciones Lambda si existían.

```bash
make terraform-deploy-eks
```

**Pre-requisitos:**
*   Tener OrbStack corriendo con Kubernetes habilitado.
*   Haber construido la imagen Docker localmente (puedes usar `make watch-eks` o `docker build`). El chart está configurado con `pullPolicy: Never` para usar imágenes locales.

### 4.2. Desplegar en Modo Lambda (Serverless)

Este comando configura Terraform con `deploy_mode=lambda`. Desplegará las funciones en LocalStack y eliminará los releases de Helm del cluster si existían.

```bash
make terraform-deploy-lambda
```

**Pre-requisitos:**
*   Tener LocalStack corriendo (`docker-compose -f docker-compose.localstack.yml up -d`).

---

## 5. Verificación

### En Modo EKS
Verifica que los pods estén corriendo en tu cluster:
```bash
kubectl get pods
# Deberías ver algo como: sqs-consumer-gofiber-app-xxxx-yyyy
```

Verifica los logs:
```bash
kubectl logs -l app.kubernetes.io/name=gofiber-app -f
```

### En Modo Lambda
Verifica que las funciones estén activas en LocalStack:
```bash
awslocal lambda list-functions
```

Verifica los logs (usando CloudWatch en LocalStack):
```bash
awslocal logs tail /aws/lambda/gofibercore-local-sqs-consumer --follow
```

---

## 6. Producción

Para cambiar el modo en un entorno real (Staging/Prod), simplemente actualiza tu archivo de variables (`terraform.tfvars` o similar):

**Para ir a EKS:**
```hcl
deploy_mode = "eks"
```

**Para ir a Lambda:**
```hcl
deploy_mode = "lambda"
```

Al ejecutar `terraform apply`, Terraform se encargará de la migración de recursos de cómputo automáticamente. La capa de datos (RDS, ElastiCache, SQS) **no se ve afectada** y persiste entre cambios de modo.
