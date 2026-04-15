# Análisis del Motor de Procesos: Runtime y Arquitectura Futura

## 1. Análisis del Endpoint `api/v1/process-lifecycle/run`

El diseño del **Service Context** como una matriz global de JSON (`Input`) que fluye a través de todos los pasos, enriqueciéndose con `Data` y volcando instantáneas en `StepResult.Input`, es un patrón excelente del tipo "Blackboard" o "Pipe and Filter".

### Comportamiento con y sin *Override* (Redis vs BD)

Esta es una de las decisiones de diseño más elegantes para un entorno productivo híbrido:

*   **`override_process_version_id = 0` (o nulo) -> MODO PRODUCCIÓN**
    *   **¿Qué pasa con Redis?** El motor utiliza una estrategia *Cache-Aside* en Redis. Va a Redis a preguntar "Cuál es el `process_version_id` ACTIVO para el proceso X en la sede Y?". Si lo encuentra, ahorra latencia y carga a PostgreSQL. Las configuraciones de los `process_steps` muy probablemente también estén en caché.
    *   **Resultado:** Alto rendimiento, cero métricas de sobrecarga (overhead), respuesta limpia. Hecho para escalar y aguantar picos de peticiones.

*   **`override_process_version_id > 0` -> MODO TEST / PERFORMANCE**
    *   **¿Qué pasa con Redis?** **Se ignora por completo (Bypass).** El motor hace una consulta fuerte (directa) a PostgreSQL (`resolve_process_version`). 
    *   **¿Por qué?** Porque al testear un *Draft* o una configuración en beta, se busca evitar que una caché desactualizada en Redis enmascare la prueba. Se necesita la base de datos cruda.
    *   **Bonus track:** Este modo inyecta un objeto `performance` global en milisegundos y un `duration_us` (microsegundos) por cada Step en la respuesta JSON. Maravilloso para perfilar cuellos de botella antes de pasar a producción.

---

## 2. Propuestas Arquitectónicas para la Evolución del Motor

A continuación, presento ideas de diseño para modelar la máquina de estados y la concurrencia.

### Modelo A: Concurrencia (DAG vs Peso Lineal)

Actualmente, si usas `step_order` de manera lineal (1, 2, 3), estás forzado a ejecutar secuencialmente. 

**Mi propuesta: Migrar de un modelo de "Peso" a un DAG (Grafo Acíclico Dirigido).**

¿Por qué? Porque en procesamiento masivo, el tiempo es vital (especialmente en AWS Lambda donde se factura por ms).
Si el `Paso 2 (Validar Email)` y el `Paso 3 (Consultar Buró de Crédito)` no dependen el uno del otro, **deben ejecutarse en paralelo**, disminuyendo el tiempo de respuesta total del proceso.

**¿Cómo modelarlo en tu BD?**
Cambia o agrega un campo en tu tabla `process_steps`:
En lugar de `step_order INT`, usa un formato de dependencias explícitas `depends_on JSONB` (o un array de IDs/UUIDs).

*Ejemplo:*
*   `Paso 1 (Recibir Input)`: `depends_on: []` (Se ejecuta de inmediato).
*   `Paso 2 (Validar Edad)`: `depends_on: [Paso1_ID]`
*   `Paso 3 (Consultar API Banco)`: `depends_on: [Paso1_ID]` (¡El motor lanza el Paso 2 y 3 concurrentemente en goroutines!)
*   `Paso 4 (Decisión Final)`: `depends_on: [Paso2_ID, Paso3_ID]` (Mecanismo de "Join"/Barrera. Go Fiber espera a que terminen ambos para continuar).

*Beneficio:* Tus métricas en `performance.total_duration_ms` bajarán drásticamente.

### Modelo B: Máquina de Estados (State Machine) para Instancias

Por ahora, pasas un JSON y el proceso se completa sincrónicamente en una sola llamada API (Paso 1 al N).
¿Qué pasa si el Paso 3 es: "Esperar a que el usuario suba su DNI"? La llamada HTTP moriría por Timeout. 

Necesitas persistir el **Estado de la Ejecución**.

**Mi propuesta: Implementar el Patrón "Saga" o "Event Sourcing" para Instancias.**

Crearías las tablas:
1.  **`process_instances`**:
    *   `id` (UUID)
    *   `process_version_id` 
    *   `status` (RUNNING, PAUSED, COMPLETED, FAILED, CANCELLED)
    *   `global_context` (JSONB): El `ServiceContext.Input` persistido al momento exacto de la pausa o el estado actual de la bolsa de negocio.
2.  **`process_instance_events`** (o step_logs):
    *   `instance_id`
    *   `step_id`
    *   `status` (SUCCESS, FAILED)
    *   `output_data` (JSONB): Resultado puntual de la ejecución.

**El Flujo:**
1. El request entra a `/run`. Se crea una fila en `process_instances`.
2. El motor ejecuta los pasos. Si un paso devuelve un estado `domain.ErrPause` o `ASYNC_WAIT`, la función Serializa el array `ctx.Results` y el `ctx.Input` al campo `global_context` y la ejecución en Go "muere" (ahorrando memoria RAM y liberando el worker de Lambda/EKS).
3. Tres días después, el usuario sube su DNI vía webhook (un callback). Haces un POST a `/api/v1/process-lifecycle/resume/{instance_id}`.
4. Go Fiber carga el JSON desde la BD `global_context`, rehidrata tu objeto `*ServiceContext` exactamente como estaba hace 3 días, mira el último evento completado en `process_instance_events`, y continúa matemáticamente desde el siguiente paso en el roadmap o grafo de dependencias.

Este diseño convierte tu orquestador sincrónico actual en una **verdadera Máquina de Estados Asincrónica de nivel empresarial.**
