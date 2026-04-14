# Bulk Export Generate File V1

## Modelo de ejecución

Este proceso usa un patrón **async, pero secuencial por lote**.

- **Async** significa que el step `bulk/export/v1/write_csv_batch` no se ejecuta dentro del request HTTP de `run`. El motor lo despacha a cola.
- **Secuencial por lote** significa que, aunque cada ejecución ocurre por cola, el proceso avanza lote por lote.
- No se disparan todos los lotes al mismo tiempo.
- El lote siguiente se encola cuando termina el lote actual.
- Cuando el último lote termina, recién ahí se encola el step final `bulk/export/v1/merge_multipart`.

## Configuración del step de procesamiento por lote

```json
{
  "batch_size": 5000,
  "execution_policy": {
    "mode": "ASYNC",
    "label": "generar archivo v1",
    "next_step": "bulk/export/v1/merge_multipart",
    "auto_invoke": {
      "enabled": true,
      "cursor_field": "batch_index",
      "stop_condition": "is_last_batch"
    }
  }
}
```

## Significado de cada key

### `batch_size`

- Define el tamaño del lote.
- En este flujo, el valor se usa principalmente en el step 1 (`bulk/export/v1/organize`) para dividir los registros del `bulk_job` en bloques.
- Cada bloque queda referenciado en Redis para que el step 2 procese un lote por ejecución.

### `execution_policy`

- Agrupa las reglas de ejecución del step.
- Define si el step corre en línea o por cola.
- También define si el step debe reencolarse automáticamente y qué step sigue al finalizar.

### `execution_policy.mode`

- Valor actual: `ASYNC`.
- Indica que el step se despacha a la cola y no se ejecuta dentro del request HTTP que inicia el proceso.
- Permite desacoplar la ejecución del step del tiempo de respuesta del endpoint.

### `execution_policy.label`

- Valor actual: `generar archivo v1`.
- Es una etiqueta de negocio/observabilidad.
- Se usa para identificar el loop async en logs y trazabilidad.
- No define el nombre del archivo final.

### `execution_policy.next_step`

- Valor actual: `bulk/export/v1/merge_multipart`.
- Indica qué step debe ejecutarse cuando el loop de lotes termina.
- En este proceso, ese step toma todas las partes generadas en S3 y arma el archivo final.

### `execution_policy.auto_invoke`

- Activa la re-invocación automática del mismo step async.
- No reemplaza `mode`.
- `mode` define **cómo** se ejecuta el step.
- `auto_invoke` define **cómo se repite** el step hasta terminar todos los lotes.

### `execution_policy.auto_invoke.enabled`

- Valor actual: `true`.
- Activa la lógica de reencolar el mismo step después de terminar un lote.
- Si estuviera en `false`, el step correría solo una vez y el loop no continuaría automáticamente.

### `execution_policy.auto_invoke.cursor_field`

- Valor actual: `batch_index`.
- Define el nombre del campo que el worker toma del resultado del step para saber cuál es el siguiente cursor.
- En este flujo, `write_csv_batch` devuelve el siguiente `batch_index`.
- Ese valor se copia al input del siguiente mensaje para procesar el siguiente lote.

### `execution_policy.auto_invoke.stop_condition`

- Valor actual: `is_last_batch`.
- No almacena directamente `true` o `false`.
- Define el nombre de la key que el worker debe leer en el resultado del step para evaluar la condición de parada del `auto_invoke`.
- En este flujo, la key configurada es `is_last_batch`.
- El valor real de esa key sí debe ser booleano:
  - `is_last_batch = false` → el loop continúa
  - `is_last_batch = true` → el loop termina
- Cuando el step devuelve `is_last_batch = false`, el worker reencola el mismo step con el siguiente cursor.
- Cuando el step devuelve `is_last_batch = true`, el worker deja de reencolar el step actual y encola `next_step`.

Ejemplo de salida del step:

```json
{
  "batch_index": 4,
  "is_last_batch": false
}
```

- En este ejemplo:
  - `stop_condition = "is_last_batch"` le dice al worker qué campo revisar.
  - El valor de `is_last_batch` es `false`, por lo tanto el `auto_invoke` sigue.

## Secuencia del flujo

1. `bulk/export/v1/organize`
2. Divide los registros del `bulk_job` en lotes.
3. Guarda en Redis las referencias de los lotes.
4. Cambia `bulk_jobs.status_code` a `PROCESSING`.
5. `bulk/export/v1/write_csv_batch`
6. Procesa un lote por mensaje.
7. Genera un CSV parcial y lo sube a S3.
8. Devuelve el siguiente `batch_index` y el valor de `is_last_batch`.
9. Si no es el último lote, el worker vuelve a encolar el mismo step.
10. Si es el último lote, el worker encola `bulk/export/v1/merge_multipart`.
11. `bulk/export/v1/merge_multipart`
12. Integra las partes de S3 en un archivo final.
13. Guarda el registro en `bulk_job_outputs`.
14. Cambia `bulk_jobs.status_code` a `PROCESSED`.

## Configuración del step final

El step final usa una configuración adicional:

```json
{
  "file": "exports/bank/colombia/pago-colombia"
}
```

- `file` define la base de la ruta final en S3.
- El proceso concatena el `bulk_job_id`.
- Ejemplo:

```text
exports/bank/colombia/pago-colombia-34.csv
```
