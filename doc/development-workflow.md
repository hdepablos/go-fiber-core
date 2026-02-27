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

## Nivel 2: Simulación de Producción (Kubernetes Local)
**Objetivo:** Validar comportamiento en entorno orquestado (K8s) idéntico a AWS EKS.
**Herramientas:** `Skaffold`, `OrbStack`/`Kind`, `Kustomize`.

En este modo, tu código corre DENTRO de pods de Kubernetes en tu máquina. Skaffold observa tus cambios y actualiza los pods en segundos (sin push a registro).

*   ✅ Valida configuración de despliegue (Deployment, Service, Ingress).
*   ✅ Valida conexión entre microservicios en red K8s.
*   ✅ Valida variables de entorno reales.

### Comandos
```bash
# Iniciar desarrollo en modo Cluster
make dev-k8s
```
*Presiona `Ctrl+C` para detener y limpiar los recursos.*

---

## Nivel 3: GitOps Local (ArgoCD)
**Objetivo:** Garantizar que el flujo de despliegue automatizado funciona.
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
