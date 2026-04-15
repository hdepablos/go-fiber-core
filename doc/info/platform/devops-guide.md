# Guía de Operaciones DevOps y Configuración de Entornos

Esta guía detalla cómo configurar el proyecto para los entornos **Lambda** y **EKS**, y proporciona instrucciones paso a paso para el equipo de DevOps.

---

## ⚠️ Aclaración Importante: Local vs Remoto

Es fundamental distinguir entre tu entorno de desarrollo local y el despliegue real en la nube:

1.  **Entorno Local (Tu Máquina):**
    *   **NO** usa `git push` para actualizarse.
    *   Usa comandos: `make watch-lambda` o `make watch-eks`.
    *   Esto simula la nube en tu ordenador usando LocalStack.

2.  **Entorno Remoto (AWS Real):**
    *   **SÍ** usa `git push`.
    *   Al subir cambios a GitHub, el sistema de CI/CD (GitHub Actions) detecta el cambio, compila y despliega automáticamente a AWS.

---

## 1. Guía de Configuración por Entorno

Aquí es donde debes modificar variables según el entorno que quieras ajustar.

### 🅰️ Entorno Lambda (Serverless)
Si estás desplegando en Lambda, estos son los archivos clave:

*   **Variables de Infraestructura:**
    *   Archivo: `terraform/variables.tf` o `terraform/terraform.tfvars`
    *   Qué cambiar: Nombres de funciones, memoria, timeouts, variables de entorno de la Lambda.
*   **Variables de Entorno (Runtime):**
    *   Archivo: `.env` (en local) o Configuración de Lambda en AWS Console (en producción).
    *   Nota: Terraform toma las variables de tu `.env` local si usas `make generate-tfvars`.

### 🅱️ Entorno EKS (Kubernetes)
Si estás desplegando en Kubernetes:

*   **Configuración del Cluster:**
    *   Archivo: `terraform/main.tf` (Sección EKS/Helm).
    *   Qué cambiar: Número de nodos, tipo de instancia, versiones de charts.
*   **Definición del Servicio (Helm):**
    *   Archivo: `terraform/charts/gofiber-app/values.yaml`
    *   Qué cambiar: Límites de CPU/RAM, réplicas, puertos, variables de entorno del pod.

### 🔄 CI/CD (GitHub Actions)
Para controlar cómo se compila y despliega en la nube:

*   **Control de Modo (Lambda vs EKS):**
    *   Lugar: **GitHub Repository Settings -> Secrets and variables -> Actions -> Variables**.
    *   Variable: `DEPLOY_MODE`
    *   Valores: `lambda` o `eks`.

---

## 2. Guía Paso a Paso para el Equipo DevOps (Nivel Junior)

Instrucciones simples para desplegar el proyecto desde cero y mantenerlo.

### Paso 1: Configuración Inicial (Solo la primera vez)
*El equipo DevOps debe hacer esto una sola vez al recibir el proyecto.*

1.  **Crear Repositorio en GitHub:** Subir el código.
2.  **Configurar Credenciales AWS en GitHub:**
    *   Ir a `Settings` -> `Secrets and variables` -> `Actions`.
    *   Crear Secrets: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`.
3.  **Definir Modo de Despliegue:**
    *   Ir a la pestaña `Variables` (al lado de Secrets).
    *   Crear Variable: `DEPLOY_MODE` con valor `lambda` (o `eks`).
4.  **Crear Repositorio ECR (Solo si usan EKS):**
    *   Si usan EKS, deben crear un repositorio en AWS ECR para subir las imágenes Docker.

### Paso 2: Despliegue Rutinario (Día a Día)
*Lo que hacen los desarrolladores/devops cada día.*

1.  **Hacer cambios en el código.**
2.  **Ejecutar:**
    ```bash
    git add .
    git commit -m "Nueva funcionalidad"
    git push origin main
    ```
3.  **¡Listo!**
    *   GitHub Actions detectará el `git push`.
    *   Leerá la variable `DEPLOY_MODE`.
    *   Si es `lambda`: Compilará ZIPs y desplegará en Lambda.
    *   Si es `eks`: Construirá Docker Images y desplegará en Kubernetes.
    *   **No tienen que hacer nada manual.**

### Paso 3: Cambiar de Estrategia (Lambda ↔ EKS)
*Si deciden migrar de Serverless a Kubernetes o viceversa.*

1.  Ir a **GitHub Settings -> Variables**.
2.  Editar `DEPLOY_MODE`.
3.  Cambiar valor de `lambda` a `eks` (o viceversa).
4.  Hacer un nuevo `git push` (o re-ejecutar el último job).
5.  El pipeline se adaptará automáticamente y desplegará en la nueva infraestructura.

---

## Resumen de Archivos Clave

| Entorno | Qué Configurar | Archivo / Ubicación |
| :--- | :--- | :--- |
| **Local** | Variables de Entorno | `.env` |
| **Lambda** | Memoria, Timeout | `terraform/variables.tf` |
| **EKS** | CPU, RAM, Réplicas | `terraform/charts/gofiber-app/values.yaml` |
| **CI/CD** | Credenciales AWS | GitHub Secrets (`AWS_...`) |
| **CI/CD** | Switch Lambda/EKS | GitHub Variables (`DEPLOY_MODE`) |

## 5. Simulación de CI/CD Local (Git Hooks)

Para simular el flujo de "Commit -> Deploy" en tu entorno local (tal cual ocurriría en GitHub Actions), hemos incluido un **Git Hook**.

### Instalación
Ejecuta una única vez:
```bash
./tools/install-hooks.sh
```

### Uso
Cada vez que hagas un commit:
```bash
git add .
git commit -m "feat: mi nuevo cambio"
```

El sistema detectará automáticamente tu `DEPLOY_MODE` en el archivo `.env`:
- Si `DEPLOY_MODE=lambda`: Recompila y actualiza las funciones Lambda en LocalStack.
- Si `DEPLOY_MODE=eks`: Reconstruye las imágenes Docker y actualiza el cluster Kubernetes.

### Activación / Desactivación
Es ideal poder "apagar" esta simulación cuando haces commits pequeños o no quieres esperar el despliegue.

Para controlarlo, usa la variable `SIMULATE_CICD` en tu archivo `.env`:

**Para ACTIVAR (Simulación ON):**
```env
SIMULATE_CICD=true
```

**Para DESACTIVAR (Simulación OFF):**
```env
SIMULATE_CICD=false
```

Si la variable está en `false` o no existe, el hook se saltará silenciosamente y el commit será instantáneo.

## 6. Estrategias de Despliegue (Zero Downtime)

Para garantizar que el servicio **NUNCA** se caiga durante una actualización, hemos implementado estrategias nativas para cada entorno. Estas configuraciones ya están aplicadas en el código y no requieren costo adicional.

### En Lambda (Serverless)
Usamos **Lambda Aliases & Versioning**.
- **Cómo funciona:** Cada despliegue crea una nueva **Versión** inmutable de tu código.
- **Protección:** Terraform actualiza el **Alias** `prod` para que apunte a la nueva versión solo si el despliegue es exitoso.
- **Rollback:** Si algo falla, el alias sigue apuntando a la versión anterior.

### En EKS (Kubernetes)
Usamos **Rolling Updates** con **Health Probes**.
- **Liveness Probe:** Verifica si tu aplicación está viva (`/api/v1/health`). Si falla, reinicia el pod.
- **Readiness Probe:** Verifica si tu aplicación está lista para recibir tráfico. Kubernetes **NO** enviará usuarios a la nueva versión hasta que este endpoint responda `200 OK`.
- **Configuración:**
  - Archivo: `terraform/charts/gofiber-app/values.yaml`
  - Endpoint: `/api/v1/health`
  - Delay Inicial: 15 segundos (tiempo para conectar a DB/Redis).

---

## 7. 🔄 Guía de Migración de Cómputo (Lambda ↔ EKS)

Este proyecto está diseñado con **Arquitectura Hexagonal**, lo que lo hace altamente portable. Si necesitas cambiar de Lambda a EKS (o viceversa), aquí detallamos el impacto real y los pasos necesarios.

### 1. Lo que NO Cambia (✅)
Tu código Go y la lógica de negocio son idénticos para ambos entornos.
- **Aplicación:** El `main.go` detecta automáticamente si está en Lambda o EKS y se comporta acorde (Handler vs Servidor HTTP/Polling).
- **Datos:** Bases de datos (RDS), Caché (Redis) y Almacenamiento (S3) son externos y persisten independientemente del modo de cómputo.

### 2. Impacto de la Migración (⚠️)
Al cambiar la variable `DEPLOY_MODE` en Terraform/CI-CD, ocurrirá lo siguiente:

#### A. Infraestructura (Destructivo)
- Terraform **destruirá** los recursos del modo anterior (ej. borrará las Lambdas) y **creará** los nuevos (ej. creará el Cluster EKS).
- **Tiempo de caída:** Habrá un tiempo de inactividad durante el despliegue (5-20 minutos dependiendo de si el cluster EKS ya existe).

#### B. Cambio de URL (Crítico 🚨)
- **Lambda:** Usa API Gateway (`https://xyz.execute-api...`).
- **EKS:** Usa Load Balancer (`http://k8s-default-api...`).
- **Acción Requerida:** Debes actualizar tus registros DNS (CNAME) en Route53 para que tu dominio (ej. `api.miapp.com`) apunte al nuevo destino.

#### C. Comportamiento de Consumers (SQS)
- **Lambda (Push):** AWS empuja los mensajes. Escalado casi instantáneo.
- **EKS (Pull):** Los pods hacen "polling" a la cola.
- **Acción Requerida:** Monitorizar la latencia de procesamiento. KEDA se encargará del auto-escalado en EKS, pero puede requerir ajuste de parámetros (`pollingInterval`, `minReplicaCount`).

### 3. Checklist de Migración
Pasos recomendados para realizar el cambio en Producción:

1.  [ ] **Notificar Mantenimiento:** Avisar de una posible interrupción de servicio.
2.  [ ] **Cambiar Variable:** Actualizar `DEPLOY_MODE` a `eks` (o `lambda`) en GitHub Variables.
3.  [ ] **Ejecutar Pipeline:** Lanzar el despliegue vía GitHub Actions.
4.  [ ] **Actualizar DNS:** Una vez finalizado el despliegue, obtener la nueva URL (Output de Terraform) y actualizar el DNS de tu dominio.
5.  [ ] **Verificar Logs:** Confirmar que la aplicación arranca y procesa mensajes correctamente en el nuevo entorno.
