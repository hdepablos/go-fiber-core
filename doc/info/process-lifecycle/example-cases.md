# Example Cases de Process Lifecycle

## Objetivo

Este documento organiza los casos de ejemplo reproducibles del proyecto para que se puedan:

- recrear bajo demanda;
- seedear sin recordar nombres manuales;
- probar desde Bruno con bodies listos;
- y eliminar del repo cuando ya no se necesiten.

La idea es mantener limpio el código productivo y mover los ejemplos a un flujo explícito de `create/delete/recreate`.

## Comandos principales

Listar casos disponibles:

```bash
make list-example-cases
```

Crear un caso:

```bash
make create-example-case case=process_lifecycle_manager
```

Seedear un caso:

```bash
make seed-example-case case=process_lifecycle_manager
```

Recrear un caso completo:

```bash
make recreate-example-case case=process_lifecycle_manager
```

Eliminar un caso:

```bash
make delete-example-case case=process_lifecycle_manager
```

Crear o limpiar todos los casos:

```bash
make create-example-case case=all
make seed-example-case case=all
make recreate-example-case case=all
make delete-example-case case=all
```

## Qué crea cada caso

`create-example-case` recrea:

- servicios Go bajo `internal/services/test/...`;
- imports activos mediante `internal/services/examplesregistry/zz_generated_imports.go`;
- carpeta Bruno bajo `bruno/legacy/process-lifecycle/example-cases/<case>/`.

`delete-example-case` elimina solo los archivos administrados por el sistema nuevo y respeta archivos compartidos entre casos, por ejemplo `mqb1t`.

## Catálogo

| `case` | Seeder | `process_type` sembrados | Idea del caso |
|---|---|---|---|
| `process_lifecycle_manager` | `process_lifecycle_manager` | `Order process lifecycle` · `Case 1: Sequential Execution` · `Case 2: Parallel Batch Processing` · `Case 3: Hybrid Flow` · `Loan risk lifecycle` | Base del motor y ejemplos canónicos de ejecución |
| `test_process_scenarios` | `test_process_scenarios` | `Test Proceso de steps concurrente` · `Test Multi-Sede Logic` | Concurrencia por orden y resolución de versiones por sede |
| `process_lifecycle_auto_invoke` | `process_lifecycle_auto_invoke` | `Test Auto Invoke Process` · `Test Auto Invoke Process + async` · `Test Auto Invoke Process + async + finalized` | Recursión, `auto_invoke` y `next_step` |
| `multi_queue_batch_one_table_process_lifecycle` | `multi_queue_batch_one_table_process_lifecycle` | `MultiQueueBatchProcessorOneTableV1` | Fan-out por lotes sobre una sola tabla con 10k registros |
| `multi_queue_batch_one_table_recreate_records` | `multi_queue_batch_one_table_recreate_records` | `MultiQueueBatchProcessorOneTableV1` | Mismo proceso, pero recreando 200k registros para volumen |

## Caso `process_lifecycle_manager`

Comandos:

```bash
make recreate-example-case case=process_lifecycle_manager
```

`process_type` incluidos:

- `Order process lifecycle`
- `Case 1: Sequential Execution`
- `Case 2: Parallel Batch Processing`
- `Case 3: Hybrid Flow`
- `Loan risk lifecycle`

Servicios que recrea:

- `internal/services/test/*.go`
- `internal/services/test/common/*.go`
- `internal/services/test/heavy/*.go`
- `internal/services/test/loanrisk/*.go`

Lógica:

- deja un flujo mínimo `validate_input -> apply_business_rules -> persist_results`;
- deja un caso sync clásico con `common/validate`, `common/calculate`, `common/notify`;
- deja un caso batch paralelo con `batch/processor` y `batch/consolidate`;
- deja un caso híbrido `SYNC -> ASYNC -> SYNC`;
- deja un pipeline `loanrisk/*` para probar `required_keys`, tolerancia de errores y composición de resultados.

Bruno:

- `bruno/legacy/process-lifecycle/example-cases/process_lifecycle_manager/`

## Caso `test_process_scenarios`

Comandos:

```bash
make recreate-example-case case=test_process_scenarios
```

`process_type` incluidos:

- `Test Proceso de steps concurrente`
- `Test Multi-Sede Logic`

Servicios que recrea:

- `internal/services/test/steps_concurrent/concurrent_steps.go`

Lógica:

- demuestra varios steps en el mismo `step_order`;
- deja un paso secuencial posterior para validar consolidación;
- deja un segundo `process_type` para validar resolución entre versión global y versión específica por `sede_id`.

Bruno:

- `bruno/legacy/process-lifecycle/example-cases/test_process_scenarios/`

## Caso `process_lifecycle_auto_invoke`

Comandos:

```bash
make recreate-example-case case=process_lifecycle_auto_invoke
```

`process_type` incluidos:

- `Test Auto Invoke Process`
- `Test Auto Invoke Process + async`
- `Test Auto Invoke Process + async + finalized`

Servicios que recrea:

- `internal/services/test/test_auto_invoke.go`
- `internal/services/test/test_auto_invoke_finalize.go`

Lógica:

- muestra cómo propagar `last_id_processed`;
- muestra `auto_invoke` sincrónico y asincrónico;
- muestra un `next_step` final para consolidar el total procesado.

Bruno:

- `bruno/legacy/process-lifecycle/example-cases/process_lifecycle_auto_invoke/`

## Caso `multi_queue_batch_one_table_process_lifecycle`

Comandos:

```bash
make recreate-example-case case=multi_queue_batch_one_table_process_lifecycle
```

`process_type` incluido:

- `MultiQueueBatchProcessorOneTableV1`

Servicios que recrea:

- `internal/services/test/mqb1t/deps.go`
- `internal/services/test/mqb1t/organize.go`
- `internal/services/test/mqb1t/process_batch.go`
- `internal/services/test/mqb1t/finalize.go`

Lógica:

- organiza registros pendientes;
- despacha lotes asíncronos;
- controla progreso en Redis;
- finaliza imprimiendo estadísticas del proceso.

Bruno:

- `bruno/legacy/process-lifecycle/example-cases/multi_queue_batch_one_table_process_lifecycle/`

## Caso `multi_queue_batch_one_table_recreate_records`

Comandos:

```bash
make recreate-example-case case=multi_queue_batch_one_table_recreate_records
```

`process_type` incluido:

- `MultiQueueBatchProcessorOneTableV1`

Lógica:

- reutiliza exactamente los mismos servicios de `mqb1t`;
- cambia el volumen sembrado a `200000` filas;
- sirve para validar batching, tiempos y fan-out sobre una tabla única ya poblada.

Bruno:

- `bruno/legacy/process-lifecycle/example-cases/multi_queue_batch_one_table_recreate_records/`

## Relación con `seed-one`

Los seeders siguen siendo la fuente de verdad para datos y configuración en DB.

Regla práctica:

- `create-example-case` recrea código y Bruno;
- `seed-example-case` ejecuta el seeder;
- `recreate-example-case` hace ambas cosas en secuencia.

Ejemplo:

```bash
make recreate-example-case case=process_lifecycle_auto_invoke
```

## Cleanup de ejemplos viejos

El sistema nuevo reemplaza el uso de:

- imports fijos a `internal/services/test/...` en `cmd/api`, `cmd/sqs-consumer` y `cmd/cmd-cli`;
- requests legacy genéricos tipo `RunProc -> process_type_id = N.bru`.

La activación de ejemplos ahora pasa por:

- `internal/services/examplesregistry/`
- `make create-example-case ...`
- `make delete-example-case ...`
