# Optimización de Lambdas y Benchmark (Go + ARM64)

Este documento detalla las configuraciones de rendimiento y optimización aplicadas a las funciones Lambda del proyecto `go-fiber-core`.

## 1. Benchmark de Núcleos en Logs

Se ha añadido un bloque de código en el `Handler` de las funciones críticas (`api` y `sqs-consumer`) para inspeccionar en tiempo real los recursos disponibles. Esto permite verificar en CloudWatch qué está viendo Go realmente.

**Código implementado:**

```go
// Esto te dirá cuántos núcleos lógicos puede usar Go en este momento
numCPU := runtime.NumCPU()

// Esto te dice cuántas Goroutines están vivas ahora mismo
numGoroutines := runtime.NumGoroutine()

log.Printf("🚀 --- LOGS DE RENDIMIENTO ---\n")
log.Printf("💻 CPUs disponibles: %d\n", numCPU)
log.Printf("🔄 Goroutines iniciales: %d\n", numGoroutines)
log.Printf("🏗️ Arquitectura: %s\n", runtime.GOARCH)
```

## 2. Arquitectura ARM64 (Graviton2)

Se ha configurado la infraestructura para utilizar la arquitectura **ARM64** en lugar de x86_64.

*   **Ventaja:** Go corre de maravilla en ARM.
*   **Costo:** AWS cobra aproximadamente un **20% menos** por el mismo rendimiento en comparación con x86.
*   **Configuración:** Visible en `terraform/main.tf` y templates SAM (`Architectures: [arm64]`).
*   **Compilación:** Se ha actualizado `dockerfiles/Dockerfile.func.lambda` para compilar con `GOARCH=arm64`.

## 3. Memoria: 1769 MB (El "Sweet Spot")

Se ha establecido la memoria base en **1769 MB** para la API y el SQS-Consumer.

*   **Razón:** En AWS Lambda, la CPU escala proporcionalmente con la memoria. **1769 MB es el punto exacto donde se asigna 1 vCPU completo**.
*   **Impacto:** El salto de rendimiento desde los 128MB por defecto es brutal, evitando el "time-slicing" agresivo de CPU que ocurre en configuraciones de baja memoria.

## 4. Validación

*   **En LocalStack:** Configurar todo igual que en AWS para validar que los templates (`template.yml` o `serverless.yml`) estén bien escritos.
*   **En AWS:** Verificar en la consola que las funciones estén corriendo en "arm64" y tengan 1769MB asignados.

## 5. Resiliencia y Manejo de Errores SQS

Para garantizar que los mensajes fallidos se reintenten correctamente y lleguen a la Dead Letter Queue (DLQ), se ha configurado:

*   **ReportBatchItemFailures:** En Terraform, el trigger de SQS incluye `function_response_types = ["ReportBatchItemFailures"]`.
    *   *Por qué es vital:* El handler de Go devuelve `events.SQSEventResponse` con los IDs fallidos y `error = nil`. Sin esta configuración en Terraform, SQS asume éxito total y borra el mensaje aunque haya fallado, rompiendo el ciclo de reintentos.
*   **Redrive Policy:** `maxReceiveCount = 3`. El mensaje se intenta 3 veces antes de ir a DLQ.
*   **Visibility Timeout:** Configurado a **30 segundos** (valor estándar recomendado) para producción.

## 6. Flujo de Trabajo y Comandos Clave

Para agilizar el desarrollo, utiliza los siguientes comandos según el tipo de cambio que realices:

### A) Si cambias archivos de Infraestructura (Terraform)
Si modificas archivos `.tf` (ej. `terraform/sqs.tf` o `terraform/main.tf`):
```bash
make infra-deploy
```
*Este comando actualiza la configuración de la infraestructura (colas, permisos, memoria) sin recompilar el código.*

### B) Si cambias Código de Funciones (Go)
Si modificas el código fuente en `cmd/...` o `internal/...`:

**Para actualizar UNA función rápidamente (recomendado):**
```bash
make fast-deploy FOLDER=sqs-consumer
```
*(Sustituye `sqs-consumer` por `api`, `every-1min-cron`, etc.)*

**Para actualizar TODAS las funciones (ej. cambio en servicio compartido):**
```bash
make fast-deploy-all
```

*Nota: Estos comandos `fast-deploy` usan compilación nativa local y suben el código directamente a LocalStack, ahorrando el tiempo de compilación con Docker.*
