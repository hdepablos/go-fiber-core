## Seeders de `catalog_items`

### Esquema de la tabla

Migración: [20260217190327_create_catalog_items_table.sql](file:///private/var/www/go-fiber-core/internal/database/migrations/postgres/20260217190327_create_catalog_items_table.sql)

Campos principales:

- `id`: BIGSERIAL, PK
- `name`: nombre legible del ítem
- `code`: código lógico (BIGINT), único por nivel
- `parent_id`: referencia a otro `catalog_items.id` para jerarquía
- `sort_order`: entero para ordenar hermanos (default 0)
- `metadata`: JSONB con información auxiliar
- `is_active`: booleano (default TRUE)

Restricciones e índices relevantes:

- FK `fk_catalog_parent` sobre `parent_id` hacia `catalog_items(id)`
- CHECK `chk_no_self_parent` evita `parent_id = id`
- Índice único `uq_catalog_code_per_parent` en `(code, parent_id)` con `deleted_at IS NULL`
- Índices de soporte para `parent_id`, `code` y GIN sobre `metadata`

### Archivo de datos JSONC

Ruta: [catalog_items.jsonc](file:///private/var/www/go-fiber-core/internal/database/seeders/files/catalog_items.jsonc)

Formato base (JSON con comentarios tipo JSONC):

```jsonc
[
  /* BLOQUE: Tablas base */
  {
    "id": 1,
    "name": "table",
    "code": 1,
    "parent_id": 0,
    "sede_id": 1,
    "order": 1,
    "active": 1,
    "created_at": null,
    "updated_at": null
  },
  {
    "id": 2,
    "name": "table_filter",
    "code": 2,
    "parent_id": 0,
    "sede_id": 1,
    "order": 2,
    "active": 1,
    "created_at": null,
    "updated_at": null
  }
]
```

Significado de campos:

- `id`: identificador interno solo para el archivo JSONC
- `name`: se mapea a `catalog_items.name`
- `code`: puede ser número o string; siempre se guarda como texto
- `parent_id`: referencia al `id` de otro elemento del mismo JSONC; `0` = sin padre
- `sede_id`, `order`, `active`, timestamps: se guardan dentro de `metadata` o en `sort_order`

### Lógica del seeder `catalog_items`

Implementación: [catalog_items_seeder.go](file:///private/var/www/go-fiber-core/internal/database/seeders/catalog_items_seeder.go)

Comportamiento:

- Lee el archivo `catalog_items.jsonc`, elimina comentarios y parsea el JSON
- Resuelve primero los padres y luego los hijos usando `parent_id` (id del JSONC)
- Usa `order` del JSONC para poblar la columna `sort_order`
- Calcula `is_active` a partir de `active == 1`
- Construye un `metadata` JSONB con:
  - `source_id` (id del JSONC)
  - `sede_id`
  - `order`
  - `parent_json_id`

Idempotencia:

- No hace `TRUNCATE` de la tabla
- Antes de insertar, busca por `(code, parent_id)` con `deleted_at IS NULL`
- Si ya existe un registro con esa llave, no inserta uno nuevo
- Permite ejecutar el seeder múltiples veces sin duplicar datos

Resolución de `code`:

- `code` soporta:
  - `"ABC123"` → se guarda como `"ABC123"`
  - `100` → se guarda como `"100"`
  - `100.0` → se normaliza a `"100"`

### Ejecución de seeders

CLI principal: [cmd/cmd-cli](file:///private/var/www/go-fiber-core/cmd/cmd-cli)

Ejecutar todos los seeders:

```bash
go run ./cmd/cmd-cli main.go seed
```

Ejecutar solo el seeder de `catalog_items`:

```bash
go run ./cmd/cmd-cli main.go seed --only catalog_items
```

Se pueden ejecutar varios a la vez separados por comas:

```bash
go run ./cmd/cmd-cli main.go seed --only banks,roles,catalog_items
```

Nombres de seeders registrados actualmente:

- `banks`
- `roles`
- `menus`
- `catalog_items`
- `create_test_user`
- `role_user`
- `menu_user`

