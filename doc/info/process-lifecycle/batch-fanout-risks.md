# Batch Fanout Risks

## Objetivo

Listar cosas que no conviene hacer, riesgos operativos y anti-patrones del nuevo modo fan-out.

Este documento es deliberadamente preventivo.
No reemplaza la guía principal:

- `doc/info/process-lifecycle/batch-fanout-guide.md`

## No asumir que más paralelismo siempre mejora

Subir al mismo tiempo:

- `parallel_shards`
- `concurrent_batches`

puede empeorar el sistema si el cuello está en:

- base de datos,
- locks,
- API externa,
- Redis,
- ancho de red,
- costo de Lambda.

## No arrancar con valores agresivos

Evitar empezar con algo como:

- `parallel_shards = 16`
- `concurrent_batches = 4`

sin benchmark previo.

Eso puede disparar demasiada presión concurrente desde el primer despliegue.

## No pensar `auto_invoke` como fan-out por sí solo

`auto_invoke` simple sigue siendo una sola cadena secuencial.
El fan-out real necesita:

- dispatch de varios shards,
- coordinación global,
- finalize único.

## No usar `is_last_batch` como condición de fan-out

En modo distribuido la condición correcta es fin de shard, no fin global.

Usar `is_last_batch` como `stop_condition` en fan-out puede:

- cortar mal una rama,
- disparar finalize de forma incorrecta,
- mezclar semántica secuencial con distribuida.

## No disparar finalize desde todos los shards

Ese es uno de los peligros principales.

Si cada shard intenta finalizar por su cuenta, aparecen:

- dobles cierres,
- limpieza prematura de Redis,
- errores intermitentes muy difíciles de reproducir.

El finalize debe quedar protegido por coordinación distribuida.

## No asumir que Lambda evita duplicados

Lambda y SQS son at-least-once.
Puede haber reintentos y duplicados.

Por eso:

- la marca de shard completado debe ser idempotente,
- el finalize debe tener lock,
- el proceso de negocio idealmente también debería tolerar reintentos.

## No ignorar el costo del throttle global

Si una API externa tiene límites, no alcanza con throttlear por invocación individual.
En fan-out, varios shards pueden pegar al mismo destino.

El throttle debe pensarse como recurso compartido global.

## No usar `apply_changes=true` como reemplazo del lifecycle real

`apply_changes` sirve para desarrollo local y pruebas acotadas.
No reemplaza:

- `start`
- dispatch de shards
- finalize real
- consolidación completa del padre

## No mezclar filtros de prueba con corrida masiva sin revisarlos

Si en Bruno o en pruebas manuales quedaron filtros como:

- `row_number`
- `item_ids`
- `status_code`

y luego se reutilizan accidentalmente, la corrida puede procesar solo una muestra y generar una falsa sensación de éxito.

## No reutilizar `key_redis` sin intención

Reusar la misma `key_redis` puede mezclar observación de corridas o hacer más difícil el troubleshooting.
En desarrollo puede ser útil para repetir una simulación, pero hay que hacerlo conscientemente.

## No olvidar que `delete-process` no limpia base de datos

El cleanup actual remueve:

- archivos,
- imports,
- wiring,
- seeders del repositorio

pero no elimina automáticamente:

- `process_types`
- `process_versions`
- `process_steps`

ya sembrados en la base.

## No crear colecciones Bruno por proceso batch

La convención actual es una sola carpeta genérica:

- `bruno/legacy/process-lifecycle/test-batch-process/`

Crear carpetas por proceso vuelve a fragmentar el modelo y reintroduce mantenimiento duplicado.

## No separar el negocio en dos `process_type` por solo cambiar el modo técnico

Evitar crear cosas como:

- `imputations_sequential`
- `imputations_fanout`

como procesos de negocio distintos, si la lógica funcional es la misma.

La recomendación es:

- un solo `process_type`
- múltiples `process_versions`

## No asumir que EKS y Lambda se comportan igual

El contrato del motor debe soportar ambos, pero operativamente cambian cosas como:

- latencia de arranque,
- cantidad de workers activos,
- límites de concurrencia,
- costo por invocación,
- observabilidad.

La misma configuración puede rendir distinto en cada plataforma.

## Señales de alerta

Revisar si aparecen:

- finalize ejecutándose más de una vez,
- shards sin marcarse como completos,
- tiempos peores después de subir paralelismo,
- presión de CPU/DB/Redis demasiado alta,
- errores 429 o rate limit externo,
- mensajes duplicados o retried spikes en cola.

## Recomendación operativa

Escalar de forma progresiva:

1. `parallel_shards = 1`, `concurrent_batches = 1`
2. subir `parallel_shards`
3. medir
4. recién luego subir `concurrent_batches`

## Trazabilidad

- `doc/info/process-lifecycle/batch-fanout-guide.md`
- `doc/specs/process-lifecycle/batch-fanout-spec.md`
