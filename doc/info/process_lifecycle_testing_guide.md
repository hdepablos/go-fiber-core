# Guía de Pruebas de Process Lifecycle

A continuación se detallan los pasos para probar cada uno de los escenarios implementados.

**Nota Importante:** Los IDs mostrados a continuación asumen una base de datos limpia. Si has corrido el seeder múltiples veces, los IDs pueden variar.

## 1. Ejecutar Seeder

El seeder limpia y repobla las tablas necesarias (`process_types`, `process_versions`, `process_steps`).

**Usando Makefile (Recomendado):**
Este comando ejecuta el seeder dentro del contenedor Docker, asegurando que tenga acceso correcto a la red y base de datos.

```bash
make seed-one name=process_lifecycle_manager
```

**Opción Manual (Solo si tienes Go y BD local configurada):**
```bash
go run cmd/cmd-cli/main.go seed --only process_lifecycle_manager
```

---

## Caso 1: Ejecución Secuencial Simple (SYNC)

**Descripción:** 3 pasos simples (Validar -> Calcular -> Notificar) ejecutados en memoria.
**IDs Esperados:** `process_type_id: 2`, `process_version_id: 2` (DRAFT)

### Request
**Endpoint:** `POST /api/v1/process-lifecycle/run`

```json
{
  "process_type_id": 2,
  "sede_id": 1,
  "override_process_version_id": 2,
  "roadmap": 0,
  "input": {
    "age": 25,
    "email": "test@example.com"
  }
}
```

### Salida Esperada
```json
{
  "status": "success",
  "data": {
    "process_version_id": 2,
    "result": {
      "common/validate": { "status": "completed", "data": { "valid": true } },
      "common/calculate": { "status": "completed", "data": { "result": 37.5 } }, // 25 * 1.5
      "common/notify": { "status": "completed", "data": { "sent": true } }
    }
  }
}
```

**Explicación:** El motor ejecuta los 3 pasos uno tras otro y devuelve el resultado consolidado inmediatamente.

---

## Caso 2: Ejecución Paralela y Recursiva (ASYNC Batching)

**Descripción:** 4 Workers Async (Paralelos) + 1 Consolidación Final.
**IDs Esperados:** `process_type_id: 3`, `process_version_id: 3` (DRAFT)

### Request
**Endpoint:** `POST /api/v1/process-lifecycle/run`

```json
{
  "process_type_id": 3,
  "sede_id": 1,
  "override_process_version_id": 3,
  "roadmap": 0,
  "input": {
    "partition_id": "A1",
    "last_id_processed": 0
  }
}
```

### Salida Esperada
```json
{
  "status": "success",
  "data": {
    "process_version_id": 3,
    "result": {
      "batch/processor": { 
        "status": "pending", 
        "message": "Step dispatched to queue: batch-queue" 
      }
    }
  }
}
```

**Explicación:**
1. La API responde rápido indicando que el proceso "batch/processor" fue enviado a la cola (ASYNC).
2. En background (SQS), 4 workers procesarán lotes.
3. Cada worker se re-invocará (`auto_invoke`) hasta terminar su partición.
4. Cuando terminen, se disparará el paso "batch/consolidate".

---

## Caso 3: Flujo Híbrido (Sync -> Async -> Sync)

**Descripción:** Validación Sync -> Proceso Pesado Async -> Notificación.
**IDs Esperados:** `process_type_id: 4`, `process_version_id: 4` (DRAFT)

### Request
**Endpoint:** `POST /api/v1/process-lifecycle/run`

```json
{
  "process_type_id": 4,
  "sede_id": 1,
  "override_process_version_id": 4,
  "roadmap": 0,
  "input": {
    "file_url": "http://s3...",
    "user_id": 99
  }
}
```

### Salida Esperada
```json
{
  "status": "success",
  "data": {
    "process_version_id": 4,
    "result": {
      "common/validate": { 
        "status": "completed", 
        "data": { "file_valid": true } 
      },
      "heavy/process": { 
        "status": "pending", 
        "message": "Step dispatched to queue: vip-queue" 
      }
    }
  }
}
```

**Explicación:**
1. El paso "common/validate" se ejecuta y pasa (SYNC).
2. El paso "heavy/process" se detecta como ASYNC y se manda a la cola `vip-queue`.
3. La API detiene la ejecución aquí y responde al cliente.
4. El paso "common/notify" NO se ejecuta todavía; esperará a que el worker termine el paso pesado.
