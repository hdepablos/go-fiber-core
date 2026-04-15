# Guía de Despliegue en EKS (Local con OrbStack)

Este documento detalla cómo desplegar y desarrollar los servicios en un entorno Kubernetes local utilizando OrbStack, manteniendo LocalStack por separado para los servicios de AWS.

## 1. Comandos de Desarrollo

### `make watch-eks`

Este comando permite un ciclo de desarrollo rápido (hot-reload) en Kubernetes, similar a `make watch` pero para EKS.

**Uso:**
```bash
make watch-eks service=sqs-consumer
```

**Qué hace internamente:**
1.  Construye la imagen Docker del servicio especificado (`sqs-consumer`, `api`, etc.) usando `Dockerfile.universal`.
2.  Etiqueta la imagen como `:local` para que Kubernetes la use sin necesidad de un registro externo.
3.  Reinicia el despliegue (`rollout restart`) para forzar al pod a usar la nueva imagen.
4.  Muestra los logs en tiempo real del pod recién creado.

## 2. Archivos de Configuración Kubernetes

### `k8s/sqs-consumer/deployment.yaml`
Define **CÓMO** se ejecuta la aplicación en el cluster.
- **Imagen**: Usa `sqs-consumer:local` (construida por `make watch-eks`).
- **Variables de Entorno**: Mapea la configuración necesaria (Base de datos, Redis, AWS) para que la aplicación funcione en el contenedor.
    - **Nota Importante**: Usa `host.docker.internal` para conectar con servicios fuera del cluster (Postgres local, LocalStack).
- **Recursos**: Límites de CPU y Memoria.

### `k8s/sqs-consumer/scaledobject.yaml`
Define **CUÁNDO** escalar la aplicación (Auto-scaling con KEDA).
- **Trigger**: Basado en la longitud de la cola SQS (`aws-sqs-queue`).
- **Polling**: Revisa la cola cada 5 segundos.
- **Escalado**:
    - Mínimo: 0 replicas (si no hay mensajes, se apaga).
    - Máximo: 5 replicas (para procesar carga alta en paralelo).
- **Autenticación**: Usa credenciales falsas (LocalStack) definidas en `TriggerAuthentication`.

## 3. Notas sobre la Instalación de KEDA

Si al instalar KEDA ves un mensaje como:
> *"Release name already exists"*

**No te preocupes**. Esto simplemente significa que KEDA ya estaba instalado en tu cluster. El comando es idempotente y seguro de ejecutar múltiples veces.

## 4. Troubleshooting Común

### Error: `connection refused` a `host.docker.internal:4566`
Si ves errores de conexión a SQS o AWS en los logs del pod:
```
dial tcp ...:4566: connect: connection refused
```
**Causa**: LocalStack no está corriendo o no es accesible.
**Solución**:
1.  Asegúrate de levantar LocalStack:
    ```bash
    docker-compose -f docker-compose.localstack.yml up -d
    ```
2.  Despliega la infraestructura (colas, etc.):
    ```bash
    make infra-deploy
    ```

### Error: `driver GORM no soportado: ""`
**Causa**: Las variables de entorno de base de datos no se están pasando correctamente al pod.
**Solución**: Verifica que `deployment.yaml` tenga todas las variables `GORM_...` y `PGX_...` definidas correctamente.

---

## 5. Siguientes Pasos: Despliegue Híbrido

Para una gestión avanzada de infraestructura que soporte tanto Lambda como EKS usando el mismo código de Terraform, consulta la guía de arquitectura híbrida:

👉 **[Estrategia de Despliegue Híbrido: Lambda vs EKS](./hybrid-deployment.md)**
