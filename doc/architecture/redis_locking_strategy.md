# Estrategia de Redis Locking Cache-Aside

## Contexto
En sistemas de alta concurrencia, existe una condición de carrera crítica al actualizar configuraciones o datos maestros. Si una petición entra justo entre el momento en que se actualiza la base de datos y se invalida la caché, puede leer datos antiguos y volver a cachearlos, perpetuando la inconsistencia.

Para mitigar esto, implementamos una estrategia de **Locking Cache-Aside**.

## Lógica del Mecanismo

El flujo se basa en bloquear las lecturas de caché ANTES de realizar cambios en la base de datos.

### 1. Lectura (`Get`)
Antes de devolver un valor de Redis, el sistema verifica si existe un "Lock" (una key especial) asociado a ese recurso.

- **Si NO existe Lock**: Retorna el valor de la caché normalmente.
- **Si EXISTE Lock**: Retorna `nil` (Cache Miss forzado), obligando a la aplicación a consultar la Base de Datos para obtener la información más reciente y autoritativa.

### 2. Escritura Crítica (`Lock` & `Unlock`)
Cuando se va a realizar una actualización crítica (ej. `PromoteProcessVersion`):

1.  **Bloqueo (`Lock`)**: Se crea una key temporal en Redis (ej. `lock:process:config:1`) con un TTL corto (ej. 10s) ANTES de iniciar la transacción en BD.
    -   *Efecto*: Todas las lecturas concurrentes empezarán a ir directo a BD.
2.  **Operación en BD**: Se realiza el `UPDATE` o `INSERT` en la base de datos.
3.  **Desbloqueo y Limpieza (`Unlock`)**:
    -   Se elimina la key de datos original (ej. `process:config:1`).
    -   Se elimina la key de bloqueo (ej. `lock:process:config:1`).
    -   *Efecto*: La próxima lectura no encontrará lock ni datos, consultará la BD nueva y repoblará la caché con la versión correcta.

## Implementación

La lógica se encapsula en un servicio centralizado de Redis para garantizar que todos los módulos del sistema respeten este protocolo.

### Convención de Naming
- **Key de Datos**: `app:module:id` (ej. `go-fiber-core:process:2`)
- **Key de Lock**: `lock:app:module:id` (ej. `lock:go-fiber-core:process:2`)
