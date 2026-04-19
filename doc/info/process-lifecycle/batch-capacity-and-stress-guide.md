# Batch Capacity And Stress Guide

## Objetivo

Definir una guía práctica para:

- estimar capacidad antes de activar fanout en Lambda o EKS,
- ejecutar pruebas de estrés de forma controlada,
- y revisar rápidamente logs específicos cuando aparezcan errores de Redis o rate limit.

Este documento es operativo para personas.
Las reglas verificables viven en `doc/specs/process-lifecycle/batch-observability-spec.md`.

## Tipos de logs específicos

El batch fanout deja dos familias de logs estructurados:

- `log_type=redis_guard`
- `log_type=rate_limit_guard`

La intención es que en producción puedas filtrar esos eventos sin leer todos los logs del servicio.

## Qué cubre cada familia

### `redis_guard`

Se usa para:

- fallos de lectura o escritura en Redis,
- fallos al registrar shards,
- fallos al marcar fin de shard,
- fallos al tomar el lock de finalize,
- fallos en contadores o cleanup del estado batch.

Campos típicos:

- `log_type=redis_guard`
- `event_type=redis_operation_error`
- `operation`
- `redis_key`
- `parent_id`
- `component`

### `rate_limit_guard`

Se usa para:

- rate limit interno del core batch,
- `cooldown` interno,
- `max_in_flight` excedido,
- límite de ventana excedido,
- respuestas HTTP `429` de dependencias externas,
- errores de transporte hacia dependencias externas.

Campos típicos:

- `log_type=rate_limit_guard`
- `event_type`
- `scope`
- `source`
- `status_code`
- `endpoint`
- `retry_after`

## Dónde buscar en producción

Según la estrategia actual del repositorio:

- en Lambda o EKS los logs deben salir a `stdout`,
- CloudWatch captura esos eventos JSON,
- y luego se filtran por `log_type` y `event_type`.

Filtros útiles:

```text
{ $.log_type = "redis_guard" }
```

```text
{ $.log_type = "rate_limit_guard" }
```

```text
{ $.log_type = "rate_limit_guard" && $.event_type = "external_http_429" }
```

```text
{ $.log_type = "rate_limit_guard" && $.scope = "internal" }
```

## Checklist de capacidad antes de habilitar fanout

Antes de subir `parallel_shards` o mover el proceso a Lambda/EKS, revisar:

1. Redis
   - latencia promedio y p95,
   - memoria disponible,
   - evictions en cero,
   - conexiones máximas y uso real,
   - CPU sin saturación.
2. Proceso batch
   - tamaño de lote real,
   - cantidad de batches esperada,
   - `parallel_shards` inicial acotado,
   - `concurrent_batches=1` al arrancar.
3. Dependencias externas o internas
   - existencia de rate limit conocido,
   - comportamiento documentado ante `429`,
   - retry/backoff definido,
   - timeout explícito,
   - necesidad de dosificación por tandas usando `dispatch_pacing`.
4. Entorno
   - cantidad esperada de workers/pods/Lambdas concurrentes,
   - pool de conexiones a Redis,
   - límites de red y CPU del contenedor o función.
5. Observabilidad
   - filtros CloudWatch listos para `redis_guard`,
   - filtros CloudWatch listos para `rate_limit_guard`,
   - alarmas o revisión manual definida para picos.

## Configuración inicial recomendada

Para la primera salida controlada:

- `parallel_shards = 2` o `4`
- `concurrent_batches = 1`
- `dispatch_pacing` activo si el destino no tolera bursts
- `batch_size` moderado
- TTL Redis suficiente para la corrida completa

No conviene arrancar con:

- `parallel_shards` alto y `concurrent_batches` alto a la vez,
- `parallel_shards` alto sin pacing cuando el destino requiere cadencia,
- batches gigantes,
- múltiples procesos fanout pesados compartiendo la misma instancia Redis sin medición previa.

## Plan mínimo de stress test

### Fase 1. Baseline secuencial

Ejecutar la versión secuencial y registrar:

- duración total,
- total de registros,
- total de batches,
- CPU/memoria del worker,
- latencia y errores Redis,
- errores HTTP externos,
- cantidad de `429`.

### Fase 2. Fanout conservador

Ejecutar la versión fanout con:

- `parallel_shards=2`
- `concurrent_batches=1`

Registrar exactamente las mismas métricas del baseline.

Si el destino necesita ritmo controlado, correr también una variante con `dispatch_pacing` activo.

### Fase 3. Fanout objetivo

Subir gradualmente a:

- `parallel_shards=4`
- luego evaluar `parallel_shards=6` u `8` solo si Redis y dependencias responden bien.

No subir `concurrent_batches` mientras sigan apareciendo:

- eventos `redis_guard`,
- `external_http_429`,
- o `internal_rate_limit_*`.

## Criterios de corte del stress test

Detener la prueba o no promover configuración si aparece cualquiera de estos síntomas:

- errores recurrentes en `redis_guard`,
- `external_http_429` sostenidos,
- `internal_rate_limit_window` repetido,
- `internal_rate_limit_max_inflight` repetido,
- crecimiento visible del tiempo total sin mejora proporcional,
- finalize duplicado o shard completion inconsistente.

## Señales de que Redis está sufriendo

Indicadores típicos:

- picos de latencia Redis,
- fallos intermitentes de `SetNX`, `IncrBy`, `LRange`, `Get`,
- acumulación de retries,
- demasiadas llaves activas por corrida,
- caída del beneficio del fanout respecto al baseline.

## Señales de que el cuello está fuera de Redis

Indicadores típicos:

- `external_http_429`,
- latencia alta en APIs externas,
- `timeout` de adapters,
- base de datos lenta,
- workers o Lambdas saturados sin errores Redis.

## Recomendación para Data Providers y Adapters

Cuando un `DataProvider` o un adapter haga llamadas HTTP:

- debe registrar `external_http_429` si la respuesta es `429`,
- debe registrar `external_dependency_error` si hay error de red o timeout,
- y debe incluir `source`, `endpoint`, `method` y `retry_after` cuando exista.

Esto no reemplaza el manejo de negocio del error; solo mejora la observabilidad.

## Trazabilidad

- `internal/logger/guard_logs.go`
- `internal/services/batchflow/state_store.go`
- `internal/services/batchflow/throttle.go`
- `internal/services/batchflow/dispatch_pacing.go`
- `internal/adapters/backoffice_adapter.go`
- `internal/adapters/discord_adapter.go`
- `doc/info/operations/logs.md`
- `doc/info/process-lifecycle/batch-fanout-guide.md`
