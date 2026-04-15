# HTTP API Endpoints Guide

Esta guia resume los endpoints HTTP expuestos por el proyecto, agrupados por URL real y con ejemplos de request body cuando aplica.

## Base URL

- base API: `/api/v1`
- raiz simple: `/`
- callback fuera del prefijo API: `/oauth/google/callback`

## Respuesta estandar

La mayoria de endpoints responde con la envoltura:

```json
{
  "status": "success",
  "message": "texto",
  "data": {}
}
```

En errores controlados, el API usa una forma compatible:

```json
{
  "status": "error",
  "message": "texto",
  "data": {},
  "errors": {}
}
```

## Autenticacion

### Publicos

- `GET /`
- `GET /api/v1/health`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/google`
- `POST /api/v1/auth/google/exchange`
- `GET /oauth/google/callback`

### Protegidos con Bearer token

Todos los demas endpoints bajo `/api/v1` requieren `Authorization: Bearer <token>`.

## Request shapes reutilizables

### Paginacion

Muchos endpoints `POST .../paginated` reutilizan esta estructura:

```json
{
  "sortBy": ["id"],
  "sortDesc": [true],
  "filterBy": [],
  "filterValues": [],
  "rowsPerPage": 10,
  "page": 1,
  "optimize_with_key": ""
}
```

### Asignacion en lote

```json
{
  "user_ids": [1, 2],
  "role_ids": [1]
}
```

o bien:

```json
{
  "menu_ids": [1, 2],
  "user_ids": [10, 11]
}
```

## Endpoints por URL

### Root

#### `GET /`

- auth: none
- uso: smoke test basico del servicio
- body: no aplica

### Health

#### `GET /api/v1/health`

- auth: none
- uso: healthcheck general de la app

#### `GET /api/v1/database/health`

- auth: bearer
- uso: estado agregado de `gorm`, `pgx` y `redis`

#### `GET /api/v1/database/health/redis`

- auth: bearer
- uso: health puntual de Redis

#### `GET /api/v1/database/health/gorm`

- auth: bearer
- uso: health de conexiones GORM read/write

#### `GET /api/v1/database/health/pgx`

- auth: bearer
- uso: health de conexiones PGX read/write

### Auth

#### `POST /api/v1/auth/login`

- auth: none
- headers recomendados: `X-Client-Code: bruno`

```json
{
  "email": "admin@example.com",
  "password": "12345678"
}
```

#### `POST /api/v1/auth/refresh`

- auth: none

```json
{
  "refresh_token": "refresh-token-ejemplo"
}
```

#### `GET /api/v1/auth/google`

- auth: none
- uso: inicia login OAuth con Google
- opcion util: `?mode=json`

#### `POST /api/v1/auth/google/exchange`

- auth: none

```json
{
  "code": "oauth-exchange-code"
}
```

#### `GET /oauth/google/callback`

- auth: none
- query esperada: `code`, `state`
- uso: callback OAuth de Google

#### `POST /api/v1/auth/logout`

- auth: bearer
- body: no aplica

#### `POST /api/v1/auth/revoke-session`

- auth: bearer

```json
{
  "session_id": "session-uuid"
}
```

#### `POST /api/v1/auth/revoke-user-sessions`

- auth: bearer

```json
{
  "user_id": 1
}
```

#### `POST /api/v1/auth/active-sessions`

- auth: bearer
- body: usa `PaginationRequest`

#### `POST /api/v1/auth/revoke-all-sessions`

- auth: bearer
- body: no aplica

### Users

#### `POST /api/v1/users/`

- auth: bearer

```json
{
  "name": "Admin Demo",
  "email": "admin.demo@example.com",
  "password": "12345678",
  "role_ids": [1]
}
```

#### `GET /api/v1/users/`

- auth: bearer
- body: no aplica

#### `GET /api/v1/users/:id`

- auth: bearer
- body: no aplica

#### `PUT /api/v1/users/:id`

- auth: bearer

```json
{
  "name": "Admin Editado",
  "email": "admin.editado@example.com",
  "role_ids": [1, 2]
}
```

#### `POST /api/v1/users/roles`

- auth: bearer

```json
{
  "user_ids": [1, 2],
  "role_ids": [2]
}
```

#### `POST /api/v1/users/assign-roles`

- auth: bearer

```json
{
  "user_ids": [1, 2],
  "role_ids": [1]
}
```

#### `DELETE /api/v1/users/:id`

- auth: bearer
- body: no aplica

#### `DELETE /api/v1/users/hard/:id`

- auth: bearer
- body: no aplica

#### `PATCH /api/v1/users/:id/activate`

- auth: bearer
- body: no aplica

#### `PATCH /api/v1/users/:id/deactivate`

- auth: bearer
- body: no aplica

#### `POST /api/v1/users/set-active-bulk`

- auth: bearer

```json
{
  "ids": [1, 2, 3],
  "active": true
}
```

#### `POST /api/v1/users/paginated`

- auth: bearer
- body: usa `PaginationRequest`

### Banks

#### `POST /api/v1/banks/`

- auth: bearer

```json
{
  "name": "Banco Demo",
  "entity_code": "123"
}
```

#### `PUT /api/v1/banks/:id`

- auth: bearer

```json
{
  "name": "Banco Demo Editado",
  "entity_code": "123",
  "enabled": true
}
```

#### `DELETE /api/v1/banks/soft/:id`

- auth: bearer
- body: no aplica

#### `DELETE /api/v1/banks/hard/:id`

- auth: bearer
- body: no aplica

#### `GET /api/v1/banks/`

- auth: bearer

#### `GET /api/v1/banks/:id`

- auth: bearer

#### `POST /api/v1/banks/paginated`

- auth: bearer
- body: usa `PaginationRequest`

### Roles

#### `POST /api/v1/roles/`

- auth: bearer

```json
{
  "name": "Supervisor"
}
```

#### `PUT /api/v1/roles/:id`

- auth: bearer

```json
{
  "name": "Supervisor Senior",
  "is_active": true
}
```

#### `DELETE /api/v1/roles/soft/:id`

- auth: bearer

#### `DELETE /api/v1/roles/hard/:id`

- auth: bearer

#### `GET /api/v1/roles/`

- auth: bearer

#### `GET /api/v1/roles/:id`

- auth: bearer

#### `POST /api/v1/roles/paginated`

- auth: bearer
- body: usa `PaginationRequest`

### Menus

#### `GET /api/v1/menus/my`

- auth: bearer
- uso: devuelve arbol de menu del usuario autenticado

#### `POST /api/v1/menus/users-agregar/bulk`

- auth: bearer

```json
{
  "menu_ids": [1, 2],
  "user_ids": [10, 11]
}
```

#### `POST /api/v1/menus/users-delete/bulk`

- auth: bearer

```json
{
  "menu_ids": [1, 2],
  "user_ids": [10, 11]
}
```

### Menu Users

#### `POST /api/v1/menu-users/paginated`

- auth: bearer
- body: usa `PaginationRequest`

#### `POST /api/v1/menu-users/eliminar/:userId`

- auth: bearer
- body: usa `PaginationRequest`
- uso: lista menus ya asociados al usuario para posible remocion

#### `POST /api/v1/menu-users/agregar/:userId`

- auth: bearer
- body: usa `PaginationRequest`
- uso: lista menus no asociados al usuario para posible asignacion

#### `GET /api/v1/menu-users/status/:userId`

- auth: bearer
- uso: devuelve conteo total y asignado

### Catalogs

#### `GET /api/v1/catalogs/`

- auth: bearer
- body: no aplica

### Process Lifecycle

#### `POST /api/v1/process-lifecycle/replicate`

- auth: bearer

```json
{
  "process_version_id": 1,
  "operator_id": 1
}
```

#### `POST /api/v1/process-lifecycle/promote`

- auth: bearer

```json
{
  "process_version_id": 1,
  "comment": "Promocion a PROD desde Bruno",
  "promoted_by": 1
}
```

#### `POST /api/v1/process-lifecycle/resolve`

- auth: bearer

```json
{
  "process_type_id": 11,
  "sede_id": 0,
  "override_process_version_id": 0,
  "roadmap": 0
}
```

#### `POST /api/v1/process-lifecycle/current-version`

- auth: bearer

```json
{
  "process_type_id": 11,
  "sede_id": 0,
  "override_process_version_id": 0,
  "roadmap": 0
}
```

#### `POST /api/v1/process-lifecycle/to-test`

- auth: bearer

```json
{
  "process_version_id": 1
}
```

#### `POST /api/v1/process-lifecycle/versions/paginated`

- auth: bearer
- body: usa `PaginationRequest`

#### `GET /api/v1/process-lifecycle/versions/:id`

- auth: bearer

#### `POST /api/v1/process-lifecycle/run`

- auth: bearer

```json
{
  "process_type_id": 11,
  "sede_id": 0,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "id": 2,
    "key_redis": "run-demo",
    "filters": [
      {
        "field": "status_code",
        "operator": "eq",
        "value": "ERROR_PROCESS"
      }
    ]
  }
}
```

#### `POST /api/v1/process-lifecycle/export-preview`

- auth: bearer

```json
{
  "process_type_id": 17,
  "sede_id": 0,
  "override_process_version_id": 0,
  "roadmap": 0,
  "mode": "prepare",
  "batch_size": 100,
  "limit": 20,
  "offset": 0,
  "item_ids": [1, 2],
  "row_numbers": [10, 20],
  "input": {
    "id": 2,
    "key_redis": "preview-001",
    "filters": [
      {
        "field": "status_code",
        "operator": "eq",
        "value": "ERROR_PROCESS"
      }
    ]
  }
}
```

### Imports

#### `POST /api/v1/imports/all/:branchId/:refCode/:total/:key`

- auth: bearer
- content-type: `multipart/form-data`
- archivo requerido: `file`
- extensiones aceptadas: `.csv`, `.txt`

Parametros de ruta:

- `branchId`: sucursal
- `refCode`: codigo de referencia funcional
- `total`: total esperado de filas
- `key`: clave logica del import

Forma esperada:

- body multipart con campo `file`

## Bruno

La coleccion canonica organizada por URL queda en:

- `bruno/api/`

La coleccion historica previa queda preservada en:

- `bruno/legacy/`

Convenciones operativas:

- endpoints nuevos bajo `/api/v1/...` deben crearse en `bruno/api/v1/...`
- endpoints protegidos deben reutilizar `{{access_token}}`
- requests operativos deben usar `X-Client-Code: bruno`
- requests `POST .../paginated` deben partir de la misma estructura base de paginación
- variantes históricas o exploratorias no deben mezclarse con la colección canónica

Si necesitas pruebas operativas rapidas, usa primero la coleccion canonica por URL.
