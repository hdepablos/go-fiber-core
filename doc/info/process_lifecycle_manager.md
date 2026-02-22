## Gestor de Ciclo de Vida de Procesos (`process_lifecycle_manager`)

Esta migración define el modelo de datos y las funciones PL/pgSQL para administrar el ciclo de vida de procesos versionados (tipos de proceso, versiones, pasos, promociones y resolución de versión activa).

Incluye:

- Tipo ENUM: `process_version_status`
- Tablas: `process_types`, `process_versions`, `process_steps`, `process_version_history`
- Funciones: `promote_process_version`, `replicate_process_version`, `resolve_process_version`

---

## Tablas y tipos

### ENUM `process_version_status`

Define el estado de cada versión de proceso:

- `DRAFT`
- `TEST`
- `PROD`
- `HISTORY`

Se usa en las tablas `process_versions` y `process_version_history`.

### `process_types`

Catálogo de tipos de proceso, con soporte de archivado lógico.

- `id BIGSERIAL PK`
- `name VARCHAR(150) NOT NULL`: nombre del tipo de proceso.
- `description TEXT`: descripción libre.
- `archived_at TIMESTAMP NULL`: marca de archivado (soft delete lógico).
- `created_at TIMESTAMP NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMP NOT NULL DEFAULT NOW()`

### `process_versions`

Versiones por tipo de proceso (soporta múltiples sedes).

- `id BIGSERIAL PK`
- `process_type_id BIGINT NOT NULL` → FK a `process_types(id)`
- `version_number INTEGER NOT NULL`: número de versión incremental por tipo.
- `sede_id BIGINT NULL`: sede específica, o `NULL` para versión global.
- `status process_version_status NOT NULL DEFAULT 'DRAFT'`
- `archived_at TIMESTAMP NULL`: archivado lógico de la versión.
- `created_at TIMESTAMP NOT NULL DEFAULT NOW()`
- `updated_at TIMESTAMP NOT NULL DEFAULT NOW()`

Índice clave:

- `ux_unique_prod_per_type_sede`:

  ```sql
  CREATE UNIQUE INDEX ux_unique_prod_per_type_sede
  ON process_versions(process_type_id, sede_id)
  WHERE status = 'PROD' AND archived_at IS NULL;
  ```

  Garantiza como máximo **una versión `PROD` por `(process_type_id, sede_id)`**, ignorando versiones archivadas.

### `process_steps`

Pasos de una versión de proceso.

- `id BIGSERIAL PK`
- `process_version_id BIGINT NOT NULL` → FK a `process_versions(id)` con `ON DELETE CASCADE`
- `step_order INTEGER NOT NULL`: orden secuencial del paso.
- `name VARCHAR(150) NOT NULL`: nombre/etiqueta del paso.
- `execution_key VARCHAR(150) NOT NULL`: clave técnica estable para identificar el paso en tiempo de ejecución (por ejemplo, nombre de clase/servicio, key de orquestador, etc.).
- `config JSONB`: configuración arbitraria del paso (payload libre).
- `created_at TIMESTAMP NOT NULL DEFAULT NOW()`

### `process_version_history`

Historial de cambios de estado relevantes (especialmente promociones a `PROD`).

- `id BIGSERIAL PK`
- `process_version_id BIGINT NOT NULL` → FK a `process_versions(id)`
- `promoted_from_status process_version_status NOT NULL`: estado previo desde el que se promovió o cambió.
- `promoted_at TIMESTAMP NOT NULL DEFAULT NOW()`: timestamp de la acción.
- `operator_id BIGINT NOT NULL`: identificador del usuario/operador.
- `comment VARCHAR(300) NOT NULL`: comentario obligatorio de la promoción (máx. 300 caracteres).

Cada ejecución de `promote_process_version` agrega uno o dos registros:

- Uno para la versión que deja de ser `PROD` (si existía).
- Uno para la nueva versión `PROD` (con su `v_old_status`).

---

## Función `promote_process_version`

```sql
CREATE OR REPLACE FUNCTION promote_process_version(
    p_process_version_id BIGINT,
    p_operator_id BIGINT,
    p_comment VARCHAR
)
RETURNS VOID
```

Responsabilidad: promover una versión de proceso a `PROD` de forma **consistente**, manteniendo unicidad de `PROD` por tipo+sede y registrando historial.

### Flujo interno

1. **Validar longitud de comentario**

   Si `length(p_comment) > 300` lanza excepción (`Promotion comment exceeds 300 characters`).

2. **Leer y bloquear la versión objetivo**

   Usa `SELECT ... FOR UPDATE` sobre `process_versions` para:

   - Cargar `process_type_id`, `sede_id` y `status` actual (`v_old_status`).
   - Asegurar que `archived_at IS NULL`.
   - Bloquear la fila para evitar condiciones de carrera.

   Si no se encuentra, lanza: `Process version not found or archived`.

3. **Validar que la versión tenga pasos**

   Cuenta filas en `process_steps` para `p_process_version_id`.  
   Si `v_step_count = 0`, lanza: `Cannot promote version without steps`.

4. **Buscar y bloquear la versión `PROD` actual (misma sede)**

   Busca una versión `PROD` actual para el mismo `process_type_id` y misma `sede_id` (incluyendo comparaciones con `NULL` usando `IS NOT DISTINCT FROM`) y `archived_at IS NULL`, también con `FOR UPDATE`.

5. **Si existe PROD actual, moverla a HISTORY + historial**

   - Actualiza esa versión a `status = 'HISTORY'` y `updated_at = NOW()`.
   - Inserta en `process_version_history`:
     - `process_version_id = v_current_prod_id`
     - `promoted_from_status = 'PROD'`
     - `operator_id = p_operator_id`
     - `comment = p_comment`

6. **Promover la nueva versión a `PROD`**

   - Actualiza `status = 'PROD'`, `updated_at = NOW()` para `p_process_version_id`.

7. **Agregar historial para la nueva versión**

   Inserta en `process_version_history`:

   - `process_version_id = p_process_version_id`
   - `promoted_from_status = v_old_status`
   - `operator_id = p_operator_id`
   - `comment = p_comment`

### Uso típico (ejemplo SQL)

```sql
SELECT promote_process_version(
  p_process_version_id => 25,
  p_operator_id        => 123,
  p_comment            => 'Promoción a producción después de pruebas QA'
);
```

---

## Función `replicate_process_version`

```sql
CREATE OR REPLACE FUNCTION replicate_process_version(
    p_process_version_id BIGINT
)
RETURNS BIGINT
```

Responsabilidad: crear una **nueva versión en DRAFT** copiando la estructura de pasos de una versión existente.

Devuelve: `id` de la nueva fila en `process_versions`.

### Flujo interno

1. **Buscar versión origen activa**

   Lee `process_type_id` y `sede_id` desde `process_versions` para `p_process_version_id` con `archived_at IS NULL`.  
   Si no encuentra, lanza: `Process version not found or archived`.

2. **Calcular siguiente `version_number`**

   Usa `COALESCE(MAX(version_number), 0) + 1` filtrando por `process_type_id`.

3. **Crear nueva versión en `DRAFT`**

   Inserta una nueva fila en `process_versions` con:

   - `process_type_id = v_process_type_id`
   - `version_number = v_next_version_number`
   - `sede_id = v_sede_id`
   - `status = 'DRAFT'`

4. **Copiar pasos**

   Inserta en `process_steps` todas las filas de la versión origen:

   - `process_version_id = v_new_version_id`
   - `step_order`, `name`, `execution_key`, `config` se copian tal cual.

5. **Retornar `id` nuevo**

   `RETURN v_new_version_id;`

### Uso típico (ejemplo SQL)

```sql
SELECT replicate_process_version(25) AS new_version_id;
```

Luego, probablemente se editen los pasos de la nueva versión (en `DRAFT`/`TEST`) y, una vez validada, se promocione a `PROD` usando `promote_process_version`.

---

## Función `resolve_process_version`

```sql
CREATE OR REPLACE FUNCTION resolve_process_version(
    p_process_type_id BIGINT,
    p_sede_id BIGINT,
    p_override_process_version_id BIGINT DEFAULT NULL
)
RETURNS TABLE (
    process_version_id BIGINT,
    process_steps JSONB
)
```

Responsabilidad: resolver cuál es la versión efectiva a usar para un proceso dado **y** devolver los pasos asociados a esa versión en un solo llamado:

- Respeta override explícito cuando se pasa `p_override_process_version_id`.
- Si no hay override, elige una versión `PROD` por sede, con fallback a versión global.

### Flujo interno

1. **Validar que el `process_type` exista y no esté archivado**

   Si no encuentra `process_types.id = p_process_type_id` con `archived_at IS NULL`, lanza:

   ```sql
   RAISE EXCEPTION 'Process type does not exist or is archived';
   ```

2. **Override explícito (si se pasa `p_override_process_version_id`)**

   - Busca en `process_versions` por:
     - `id = p_override_process_version_id`
     - `process_type_id = p_process_type_id`
     - `archived_at IS NULL`
   - Si no encuentra, lanza: `Override version invalid`.
   - Si encuentra, usa ese `id` como `v_process_version_id`.

3. **Búsqueda por sede**

   Si no hay override:

   - Busca versión `PROD` para:
     - `process_type_id = p_process_type_id`
     - `status = 'PROD'`
     - `sede_id = p_sede_id`
     - `archived_at IS NULL`

   Si existe, usa ese `id` como `v_process_version_id`.

4. **Fallback a versión global**

   Si no hay `PROD` para esa sede:

   - Busca versión `PROD` con:
     - `process_type_id = p_process_type_id`
     - `status = 'PROD'`
     - `sede_id IS NULL`
     - `archived_at IS NULL`

   Si existe, usa ese `id` como `v_process_version_id`.

5. **Sin versión activa**

   Si tampoco encuentra versión global, lanza:

   ```sql
   RAISE EXCEPTION 'No active version found';
   ```

6. **Retornar versión y pasos**

   Una vez resuelto `v_process_version_id`, la función retorna una sola fila con:

   - `process_version_id`: el id resuelto.
   - `process_steps`: un `JSONB` que es un array de objetos, ordenados por `step_order`, con la forma:
     - `name`
     - `execution_key`
     - `config`
     - `step_order`

---

## Consideraciones de concurrencia y diseño

- `promote_process_version` usa `SELECT ... FOR UPDATE` sobre `process_versions` (y sobre la versión `PROD` actual) para evitar condiciones de carrera en promociones concurrentes.
- `ux_unique_prod_per_type_sede` garantiza unicidad de `PROD` por tipo+sede ignorando versiones archivadas.
- `process_version_history` registra:
  - De dónde venía la versión que se convierte en `PROD` (`promoted_from_status`).
  - Cambios de estado de versiones que dejan de ser `PROD`.
  - Operador (`operator_id`) y comentario obligatorio (`comment`).
