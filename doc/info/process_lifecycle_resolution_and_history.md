# Resolución de Versiones y Historial en Process Lifecycle

Este documento detalla el comportamiento del motor de ciclo de vida de procesos, específicamente cómo maneja la jerarquía de configuraciones (Global vs. Sede Específica) y cómo se registra el historial de cambios.

## 1. Jerarquía de Resolución (Herencia con Sobrescritura)

El sistema utiliza un modelo de herencia donde las configuraciones globales aplican a todas las sedes, a menos que una sede específica tenga su propia configuración "sobrescrita".

### Regla de Oro
> **"Ante la duda, usa la Global. Ante la especificidad, usa la Sede."**

### Lógica de `resolve_process_version`

Cuando se solicita una versión para una Sede (ej. `sede_id = 5`):

1.  **Búsqueda Específica:** El sistema busca si existe una versión activa (`PROD`) explícitamente para `process_type_id + sede_id = 5`.
    *   Si existe -> **La usa.** (Ignora la global).
2.  **Fallback Global:** Si no encuentra una específica, busca una versión activa (`PROD`) para `process_type_id + sede_id = NULL`.
    *   Si existe -> **La usa.**
    *   Si no existe -> **Error:** `No active version found`.

### Matriz de Comportamiento

| Config Global (NULL) | Config Sede 5 | Petición Sede 5 | Petición Sede 8 | Resultado Sede 5 | Resultado Sede 8 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| ✅ Existe (v1) | ❌ No existe | `resolve(5)` | `resolve(8)` | **Global (v1)** | **Global (v1)** |
| ✅ Existe (v1) | ✅ Existe (v2) | `resolve(5)` | `resolve(8)` | **Sede 5 (v2)** | **Global (v1)** |
| ❌ No existe | ✅ Existe (v2) | `resolve(5)` | `resolve(8)` | **Sede 5 (v2)** | ❌ Error |

---

## 2. Historial de Versiones (`process_version_history`)

El historial permite auditar **quién**, **cuándo** y **por qué** se promovió una versión a producción.

### Estructura Clave
La tabla `process_version_history` incluye el campo `process_type_id`, lo que permite consultas agregadas eficientes.

```sql
CREATE TABLE process_version_history (
    ...
    process_version_id BIGINT NOT NULL, -- Versión que FUE promovida
    process_type_id BIGINT NOT NULL,    -- Tipo de proceso (para filtrado rápido)
    promoted_from_status ...            -- Estado previo (TEST/HISTORY)
    ...
);
```

### Comportamiento del Historial

El historial es **lineal e independiente por ámbito**.
*   Las versiones globales tienen su propia línea de tiempo.
*   Las versiones de cada sede tienen su propia línea de tiempo.

#### Ejemplo de Ciclo de Vida Completo

**Paso 1: Inicio Global**
*   Se crea y promueve la Versión 35 (Global).
*   `process_versions`: ID 35 (PROD, Global).
*   `history`: ID 35 promovida a PROD.

**Paso 2: Divergencia (Sede 5)**
*   Se necesita un flujo especial para Sede 5.
*   Se crea Versión 55 (Sede 5). Se promueve.
*   `process_versions`:
    *   ID 35 (PROD, Global) <- Sigue activa para el resto.
    *   ID 55 (PROD, Sede 5) <- Activa solo para Sede 5.
*   `history`: ID 55 promovida a PROD.

**Paso 3: Actualización Sede 5**
*   Se mejora el flujo de Sede 5 con la Versión 60.
*   Al promover ID 60:
    1.  El sistema detecta que ID 55 era la PROD anterior para Sede 5.
    2.  Pasa ID 55 a `HISTORY`.
    3.  Pone ID 60 en `PROD`.
*   `process_versions`:
    *   ID 35 (PROD, Global).
    *   ID 55 (HISTORY, Sede 5).
    *   ID 60 (PROD, Sede 5).
*   `history`: ID 60 promovida a PROD (Sede 5).

### Consultas Útiles de Auditoría

**1. Ver evolución completa de un tipo de proceso (Global + Sedes):**
```sql
SELECT
    h.promoted_at,
    v.version_number,
    CASE WHEN v.sede_id IS NULL THEN 'GLOBAL' ELSE 'SEDE ' || v.sede_id END as scope,
    h.comment
FROM process_version_history h
JOIN process_versions v ON v.id = h.process_version_id
WHERE h.process_type_id = 2
ORDER BY h.promoted_at DESC;
```

**2. Ver solo cambios globales:**
```sql
SELECT h.*
FROM process_version_history h
JOIN process_versions v ON v.id = h.process_version_id
WHERE h.process_type_id = 2 AND v.sede_id IS NULL
ORDER BY h.promoted_at DESC;
```
