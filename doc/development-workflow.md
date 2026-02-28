# Flujo de Trabajo y Niveles de Desarrollo

Este proyecto soporta 3 niveles de desarrollo, desde la iteración rápida local hasta la simulación completa de producción con GitOps.

## Nivel 0: Desarrollo Clásico (Docker Todo-en-Uno)
**Objetivo:** Levantar todo el entorno (App + DB + Redis) con un solo comando sin complicaciones. Ideal para empezar rápidamente.
**Herramientas:** `Docker Compose`, `Air` (dentro del contenedor).

### Comandos
```bash
# Inicia API (con Air), Postgres y Redis en contenedores
make watch
```
*Nota: Este comando levanta Postgres en el puerto 5432 y Redis en el 6379 de tu máquina.*

---

## Nivel 1: Desarrollo Local Rápido (Solo Código)
**Objetivo:** Iteración ultra-rápida de lógica de negocio (API).
**Herramientas:** `Air` (Hot Reload), Docker Compose (para dependencias: Postgres, Redis).

En este modo, la API corre nativamente en tu máquina (sin contenedores para la app), conectándose a servicios auxiliares en Docker.

### Comandos
```bash
# 1. Levantar dependencias (LocalStack)
make infra-up

# 2. Asegúrate de tener Postgres y Redis corriendo (o usa 'make watch' para levantarlos)

# 3. Correr la API con Hot Reload (en tu host)
make dev-local
```

---

## Nivel 2: Simulación de Producción (AWS LocalStack)
**Objetivo:** Validar comportamiento en infraestructura idéntica a AWS (Lambda o EKS) usando LocalStack.
**Herramientas:** `LocalStack`, `Terraform`, `Docker`, `Make`.

Este nivel es crítico para asegurar que tu código funciona en el entorno de nube antes de subirlo.

### Opción A: Lambda (Serverless)
Simula AWS Lambda, API Gateway y SQS localmente.
```bash
# Despliega y actualiza todo el entorno Serverless
make watch-lambda
```
*   ✅ Compila binarios optimizados para Lambda (Linux ARM64).
*   ✅ Despliega infraestructura con Terraform en LocalStack.
*   ✅ Actualiza Bruno automáticamente para pruebas inmediatas.

### Opción B: EKS (Kubernetes)
Simula un cluster EKS completo con LoadBalancers y Nodos.
```bash
# Despliega y actualiza todo el entorno Kubernetes
make watch-eks
```
*   ✅ Construye imágenes Docker universales.
*   ✅ Levanta cluster K8s local (OrbStack/Docker).
*   ✅ Despliega charts de Helm y servicios.

---

## Nivel 3: CI/CD y GitOps (Automatización)
**Objetivo:** Garantizar que el flujo de despliegue automatizado funciona al hacer commit.
**Herramientas:** `GitHub Actions`, `ArgoCD`.

### Flujo de Integración Continua (CI)
Al hacer `git commit` y `push`, se dispara automáticamente el pipeline definido en `.github/workflows/deploy.yml`:
1.  **Test:** Ejecuta pruebas unitarias (`make ci-test`).
2.  **Build Lambda:** Genera los ZIPs de las funciones (`make ci-build-lambda`).
3.  **Build EKS:** Construye las imágenes Docker (`make ci-build-eks`).
4.  **Deploy (Opcional):** Si la rama es `main`, Terraform aplica los cambios en el entorno real.

Esto asegura que **cualquier cambio en el código** sea compatible tanto con Lambda como con EKS, sin importar cuál esté activo en producción.

---

## Nivel 4: GitOps Local (ArgoCD - Opcional)
**Objetivo:** Sincronización automática de manifiestos K8s.
**Herramientas:** `ArgoCD`, `Git`.

En este modo, **NO** despliegas manualmente. Haces cambios, los subes a Git, y ArgoCD sincroniza el cluster. Esto es exactamente lo que ocurre en Staging/Producción.

### Flujo
1.  Realiza cambios en tu código o manifiestos (`k8s/overlays/local`).
2.  Haz commit y push a tu rama (ej. `main`).
3.  ArgoCD detectará los cambios y actualizará el cluster automáticamente.

### Comandos Útiles
```bash
# Ver contraseña de ArgoCD
make argocd-pass

# Abrir UI de ArgoCD (https://localhost:8080)
make argocd-ui
```
