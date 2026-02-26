# Migración a EKS: Prerrequisitos y Configuración del Entorno Local

Este documento detalla las herramientas necesarias para replicar un entorno de producción EKS (Elastic Kubernetes Service) en local, permitiendo desarrollar y probar la migración de Lambda a Kubernetes con total fidelidad.

## 1. Stack Tecnológico Necesario (Ideal)

Para garantizar la paridad con producción, necesitamos simular tres capas: el orquestador (K8s), la nube (AWS) y el escalado automático (KEDA).

| Herramienta | Función | ¿Por qué es necesaria? |
| :--- | :--- | :--- |
| **OrbStack** | Motor de Kubernetes y Docker | Es el reemplazo ligero de Docker Desktop. Provee el cluster de K8s donde correrán nuestros pods. |
| **Kubectl** | CLI de Kubernetes | Es la herramienta estándar para comunicarse con el cluster (ver pods, logs, deployments). |
| **Helm** | Gestor de Paquetes de K8s | Necesario para instalar aplicaciones complejas en el cluster (como KEDA y LocalStack) de forma limpia y mantenible. |
| **KEDA** | Autoscaler de Eventos | Es el componente crítico que permite escalar los pods basándose en la longitud de una cola SQS (algo que K8s nativo no hace). |
| **LocalStack** | Emulador de AWS | Simula SQS, SNS y DynamoDB localmente. Lo instalaremos *dentro* del cluster para facilitar la red interna. |
| **Terraform** | Infraestructura como Código | Gestionará la creación de recursos tanto en AWS real como en LocalStack, asegurando consistencia. |
| **AWS CLI** | Cliente de AWS | Útil para enviar mensajes de prueba manuales a las colas SQS locales y verificar el estado. |

---

## 2. Estado Actual de tu Entorno

Basado en el análisis de tu máquina, este es el estado actual:

- ✅ **OrbStack**: Instalado y corriendo.
- ✅ **Docker**: Instalado y corriendo.
- ✅ **Kubectl**: Instalado (v1.32.7).
- ✅ **Terraform**: Instalado.
- ✅ **AWS CLI**: Instalado.
- ✅ **Helm**: Instalado.
- ✅ **Kubernetes (en OrbStack)**: Activado y corriendo (`kubectl get nodes` responde `Ready`).
- ✅ **KEDA**: Instalado y corriendo en el namespace `keda`.

---

## 3. Pasos de Instalación y Configuración

Sigue estos pasos para completar la preparación de tu entorno local.

### Paso 1: Activar Kubernetes en OrbStack
Si OrbStack ya está instalado pero K8s no responde, ejecuta:

```bash
orb config set k8s.enable true
```

Espera unos segundos a que el cluster arranque. Puedes verificarlo con:

```bash
kubectl get nodes
```

### Paso 2: Instalar KEDA en el Cluster (Usando Helm)
Una vez el cluster esté activo, instala el auto-escalador con el siguiente comando unificado:

```bash
helm repo add kedacore https://kedacore.github.io/charts && \
helm repo update && \
helm install keda kedacore/keda --namespace keda --create-namespace
```

> **Nota sobre "Release name already exists":**
> Si al ejecutar este comando recibes un error como:
> `Error: INSTALLATION FAILED: release name check failed: cannot reuse a name that is still in use`
>
> **¡Es una buena noticia!** Significa que KEDA ya estaba instalado en tu cluster (posiblemente de una sesión anterior). No necesitas hacer nada más; Helm simplemente te avisa para no sobrescribir la configuración existente.

### Paso 3: Instalar LocalStack (Opcional)
*Nota: En tu configuración actual, LocalStack corre por separado fuera del cluster para compartir recursos con Lambdas locales.*

---
## 4. Verificación Final

Una vez completados los pasos, deberías ver todos los pods del sistema corriendo:

```bash
kubectl get pods -n keda
```

Deberías ver pods como `keda-operator` y `keda-metrics-apiserver` en estado `Running`.
