# Process Lifecycle Manager - SQL Cheatsheet

Este documento contiene una referencia rápida para interactuar con las funciones del motor de ciclo de vida de procesos directamente desde la base de datos (SQL).

## 1. Replicar una versión (Clonar)
Crea una copia exacta de una versión existente y la guarda como una nueva versión en estado `DRAFT`.
Retorna el ID de la nueva versión creada.

```sql
-- Parámetros: replicate_process_version(VERSION_ID, OPERATOR_ID)
SELECT replicate_process_version(100, 5);
```

## 2. Mover a TEST
Mueve una versión desde el estado `DRAFT` al estado `TEST` para su validación.
Solo funciona si la versión está en estado `DRAFT`.

```sql
-- Parámetros: move_process_version_to_test(VERSION_ID)
SELECT move_process_version_to_test(123);
```

## 3. Promover a PROD
Promueve una versión desde `TEST` o `HISTORY` al estado `PROD`.
Retorna `VOID`.

```sql
-- Parámetros: promote_process_version(VERSION_ID, OPERATOR_ID, 'Comentario')
SELECT promote_process_version(123, 1, 'Promoción aprobada para producción');
```

## 4. Resolver Versión (Obtener pasos)
Obtiene la versión vigente y sus pasos para una configuración dada. Retorna una tabla con `process_version_id` y `process_steps`.

**Parámetros:**
1. `process_type_id` (bigint)
2. `sede_id` (bigint)
3. `override_process_version_id` (bigint | NULL): Para forzar una versión específica.
4. `roadmap` (int): Segmento del roadmap a resolver (Default: 0).

### Ejemplos de uso

**Caso estándar (Roadmap 0):**
Busca la versión `PROD` vigente para el tipo de proceso 1 y sede 1.
```sql
SELECT * FROM resolve_process_version(1, 1, NULL, 0);
```

**Caso con Override (Forzar versión específica):**
Útil para probar versiones en `DRAFT` o `TEST` antes de promoverlas.
```sql
SELECT * FROM resolve_process_version(1, 1, 123, 0);
```

**Caso para otro Roadmap:**
Obtiene los pasos configurados para el segmento 1 del roadmap.
```sql
SELECT * FROM resolve_process_version(1, 1, NULL, 1);
```
