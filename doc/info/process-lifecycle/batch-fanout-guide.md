# Batch Fanout Guide

## Objetivo

Documentar el nuevo modo de ejecucion batch con fan-out distribuido para correr bien en:

- Lambda
- EKS

Este documento explica el comportamiento operativo y la forma recomendada de configurar los bodies y steps.
Las reglas verificables viven en `doc/specs/process-lifecycle/batch-fanout-spec.md`.

Documento complementario:

- `doc/info/process-lifecycle/dispatch-pacing-guide.md`

## Problema que resuelve

Con `auto_invoke` puro el proceso es asíncrono, pero sigue siendo secuencial.
Eso funciona para corridas simples, pero desaprovecha escalado horizontal en Lambda y EKS.

El modo fan-out agrega:

- dispatch de varios shards iniciales,
- auto-invoke por shard,
- finalize único coordinado en Redis.

## Modelo recomendado de versiones

Para un proceso batch nuevo, la convención recomendada es:

- un solo `process_type` por negocio,
- una `process_version` base `sequential`,
- una `process_version` adicional `fanout`.

Ejemplo:

- `process_type`: `imputations`
- seeder base: `batch_process_imputations`
- seeder fanout: `batch_process_imputations_fanout`
- label base recomendado: `imputations`
- label fanout recomendado: `imputations fanout`

Esto evita crear procesos de negocio distintos solo para representar una diferencia técnica de ejecución.

## Flujo resultante

El pipeline batch queda conceptualmente así:

1. `start`
2. `dispatch_shards`
3. `process_batch`
4. `finalize`

### `start`

- valida el padre,
- carga registros,
- parte en batches,
- guarda summary y batches en Redis.

### `dispatch_shards`

- calcula cuántos shards reales se usarán,
- registra el estado global del fan-out en Redis,
- despacha un mensaje inicial de `process_batch` por shard,
- corta la cadena síncrona original.

### `process_batch`

- procesa batches del shard actual,
- usa `auto_invoke` para continuar solo su propia rama,
- al cerrar un shard marca progreso global,
- si es el último shard que termina dispara `finalize`.

### `finalize`

- corre una sola vez,
- consolida el resultado,
- actualiza el padre,
- limpia el estado Redis.

## Estrategia elegida

La estrategia inicial es `stride`.

Ejemplo:

- `total_batches = 12`
- `parallel_shards = 3`

Distribución:

- shard 0: `0, 3, 6, 9`
- shard 1: `1, 4, 7, 10`
- shard 2: `2, 5, 8, 11`

Esto ayuda a repartir mejor batches de costo heterogéneo.

## Configuración recomendada del step

El step `process_batch` debe quedar con una configuración de este estilo:

```json
{
  "concurrent_batches": 2,
  "dispatch_pacing": {
    "enabled": true,
    "messages_per_interval": 100,
    "interval_seconds": 10
  },
  "parallel_shards": 4,
  "execution_mode": {
    "type": "fanout",
    "parallel_shards": 4,
    "strategy": "stride"
  },
  "execution_policy": {
    "mode": "ASYNC",
    "label": "imputations fanout",
    "auto_invoke": {
      "enabled": true,
      "cursor_field": "batch_index",
      "stop_condition": "is_shard_complete"
    },
    "next_step": "bulk/process/x/finalize"
  }
}
```

## Significado de los campos

- `concurrent_batches`: batches procesados en paralelo dentro de una misma invocación.
- `dispatch_pacing`: dosificación temporal del procesamiento por tandas.
- `parallel_shards`: cantidad de ramas distribuidas.
- `execution_mode.type`: modo del proceso; para este caso `fanout`.
- `execution_mode.strategy`: estrategia de reparto; actualmente `stride`.
- `cursor_field`: cursor del shard actual.
- `stop_condition`: debe mirar fin de shard, no fin global.
- `next_step`: finalize a ejecutar cuando el último shard cierre.

## Relación con `dispatch_pacing`

Cuando `dispatch_pacing` está activo en `process_batch`, el fanout sigue funcionando, pero con una precisión importante:

- el pacing es global por corrida,
- no por shard.

Eso significa que la coordinación se hace con Redis usando `input.RedisKey`.

Ejemplo:

- `parallel_shards = 4`
- `dispatch_pacing.messages_per_interval = 100`
- `dispatch_pacing.interval_seconds = 10`

Interpretación correcta:

- entre todos los shards juntos se habilitan `100` items por ventana de `10` segundos.

Interpretación incorrecta:

- cada shard puede procesar `100` por su cuenta en la misma ventana.

## Cuándo conviene combinar fanout con pacing

Conviene cuando:

- el proceso necesita fanout por volumen,
- pero el destino final no tolera bursts altos,
- y se quiere controlar el ritmo global de salida.

No conviene asumir que fanout con pacing implica throughput lineal por shard.

## Cuerpo del `run`

El body HTTP del `run` no necesita campos nuevos para fan-out si la configuración vive en los steps.
Sigue siendo algo así:

```json
{
  "process_type_id": 19,
  "sede_id": 0,
  "override_process_version_id": 22,
  "roadmap": 0,
  "input": {
    "id": 2
  }
}
```

Si se quiere reducir el alcance, se usan filtros en `input.filters`.

Ejemplo:

```json
{
  "process_type_id": 19,
  "sede_id": 0,
  "override_process_version_id": 22,
  "roadmap": 0,
  "input": {
    "id": 2,
    "filters": {
      "status_code": "IMPORTED",
      "row_number": [2, 3, 4, 5, 6]
    }
  }
}
```

## Redis y coordinación global

El fan-out usa Redis para:

- total de shards,
- shards completados,
- marcas idempotentes por shard,
- lock de finalize.

Esto permite que Lambda o múltiples pods en EKS coordinen el fin del proceso sin doble cierre.

## Relación entre `parallel_shards` y `concurrent_batches`

Hay dos niveles de paralelismo:

- horizontal: `parallel_shards`
- local por worker: `concurrent_batches`

Ejemplo:

- `parallel_shards = 4`
- `concurrent_batches = 2`

Paralelismo teórico máximo:

- hasta `8` batches procesándose a la vez

Eso no implica que siempre convenga usar valores altos.

## Cuándo usar cada modo

### `sequential`

Conviene cuando:

- el proceso es liviano,
- el origen es pequeño,
- la base o API externa no tolera mucha concurrencia,
- se quiere maximizar simplicidad operativa.

### `fanout`

Conviene cuando:

- hay muchos registros,
- el entorno corre sobre Lambda o EKS,
- se busca aprovechar escalado horizontal,
- el proceso necesita reducir tiempo total de pared.

## Recomendación inicial para despliegue

Empezar con:

- `parallel_shards = 2` o `4`
- `concurrent_batches = 1`

Y recién luego subir `concurrent_batches` si:

- la base responde bien,
- no aparecen locks problemáticos,
- no hay presión excesiva sobre Redis,
- no hay rate limit externo.

## Bruno y testing

La carpeta batch sigue siendo genérica:

- `bruno/legacy/process-lifecycle/test-batch-process/`

El fan-out no cambia la estructura del body de `run`.
La diferencia principal vive en la configuración seed del proceso.

Si además quieres validar pacing sin correr el `run` completo, puedes usar:

- `POST /api/v1/process-lifecycle/batch-preview`
- `apply_changes=true`
- y revisar `apply_changes_metadata.dispatch_pacing`

## Capacidad y estrés

Antes de promover una configuración fanout a Lambda o EKS conviene ejecutar:

- checklist de capacidad,
- baseline secuencial,
- corrida fanout conservadora,
- y stress test gradual.

Guía operativa:

- `doc/info/process-lifecycle/batch-capacity-and-stress-guide.md`

## Trazabilidad

- `internal/services/batchflow/contracts.go`
- `internal/services/batchflow/manager.go`
- `internal/services/batchflow/state_store.go`
- `internal/services/bulkprocess/steps.go`
- `internal/services/punitorios/steps.go`
- `cmd/sqs-consumer/main.go`
- `doc/info/process-lifecycle/batch-fanout-risks.md`
- `doc/specs/process-lifecycle/batch-fanout-spec.md`
