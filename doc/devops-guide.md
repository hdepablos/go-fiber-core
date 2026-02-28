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
