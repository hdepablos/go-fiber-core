# Escenarios de Uso del Process Lifecycle

Este documento detalla los escenarios de ejecución soportados por el motor de procesos, con sus respectivas configuraciones JSON (`process_steps.config`).

## 1. Conceptos Generales

El comportamiento de cada paso se define en la sección `execution_policy` del JSON de configuración.

- **`mode`**: `"SYNC"` (por defecto) o `"ASYNC"`.
- **`required_keys`**: Lista de claves obligatorias en el input.
- **`error_tolerance`**: `"critical"` (detiene proceso) o `"tolerable"` (continúa).
- **`queue_target` (opcional)**:
  - Permite **personalizar la cola destino** por step cuando `mode = "ASYNC"`.
  - Si no se envía, el despachador usa la cola por defecto configurada en `SQS_QUEUE_NAME` (archivo `.env`).
  - La cola indicada (sea `queue_target` o `SQS_QUEUE_NAME`) **debe existir** en SQS/LocalStack; si no existe, el runtime falla con `AWS.SimpleQueueService.NonExistentQueue`.

---

## Caso 1: Ejecución Secuencial Simple (1x1)

**Descripción:** Flujo clásico donde cada paso se ejecuta uno tras otro en el mismo hilo (bloqueante). Ideal para validaciones y respuestas inmediatas.

**Configuración (`process_steps`):**
- Step 1: Orden 1 (Validar)
- Step 2: Orden 2 (Calcular)
- Step 3: Orden 3 (Notificar)

**JSON de Ejemplo (Step 1):**
```json
{
  "min_age": 18,
  "required_keys": ["age"],
  "execution_policy": {
    "mode": "SYNC"
  }
}
```

---

## Caso 2: Ejecución Paralela con Recursión (Batching Async)

**Descripción:** Múltiples workers procesan tareas pesadas en paralelo. Cada worker utiliza recursión (`auto_invoke`) para procesar grandes volúmenes de datos por lotes sin timeout.

**Configuración (`process_steps`):**
- Step 1 (Worker 1): Orden 1
- Step 1 (Worker 2): Orden 1
- Step 1 (Worker 3): Orden 1
- Step 1 (Worker 4): Orden 1
- Step 2 (Consolidación): Orden 2 (Se ejecuta cuando se despachan los anteriores)

**JSON de Ejemplo (Worker 1 - Batch Processor):**
```json
{
  "batch_size": 500,
  "required_keys": ["partition_id", "last_id_processed", "is_last_batch"],
  "execution_policy": {
    "mode": "ASYNC",                 // Enviar a cola SQS
    "queue_target": "batch-queue",   // Cola específica (opcional)
    
    // Lógica de Recursión (Auto-Invoke)
    "auto_invoke": {
      "enabled": true,               // Activar loop
      "cursor_field": "last_id_processed", // Output -> Input para la siguiente vuelta
      "stop_condition": "is_last_batch"    // Freno del loop
    }
  }
}
```

**Flujo de Auto-Invoke:**
1. El motor manda el mensaje inicial a la cola.
2. El worker procesa 500 registros.
3. El worker detecta `auto_invoke: true` y `is_last_batch: false`.
4. El worker actualiza `last_id_processed` en el input y se envía un **nuevo mensaje a sí mismo**.
5. Repite hasta que `is_last_batch: true`.

---

## Caso 3: Flujo Híbrido (Sync -> Async -> Sync)

**Descripción:** Combina validaciones rápidas con procesos pesados en background.

**Configuración:**
1. **Validación (Sync):** Verifica datos. Si falla, retorna error HTTP inmediato.
2. **Proceso Pesado (Async):** Se encola para ejecución diferida. La API responde "202 Accepted" aquí.
3. **Notificación (Sync/Async):** Se ejecuta tras el proceso pesado.

**JSON Step 1 (Sync):**
```json
{ "execution_policy": { "mode": "SYNC" } }
```

**JSON Step 2 (Async):**
```json
{ "execution_policy": { "mode": "ASYNC", "queue_target": "vip-queue" } }
```

---

## Resumen de Claves JSON (`process_steps.config`)

| Key | Tipo | Descripción |
| :--- | :--- | :--- |
| `execution_policy.mode` | string | `"SYNC"` (memoria) o `"ASYNC"` (cola SQS). |
| `execution_policy.queue_target` | string | Nombre de la cola destino (opcional). Si no existe, falla el dispatch. Si se omite, usa `SQS_QUEUE_NAME`. |
| `execution_policy.auto_invoke.enabled` | bool | Activa la recursión del paso. |
| `execution_policy.auto_invoke.cursor_field` | string | Campo del output que actualiza el input. |
| `execution_policy.auto_invoke.stop_condition` | string | Campo booleano del output que detiene el loop. |
| `required_keys` | []string | Lista de claves obligatorias en el Input. |
| `error_tolerance` | string | `"critical"`, `"tolerable"`, `"inherit"`. |
