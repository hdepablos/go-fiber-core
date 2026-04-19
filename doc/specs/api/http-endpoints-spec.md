---
domain: api
summary: Contrato documental mínimo para endpoints HTTP, handlers, DTOs y requests Bruno canónicos organizados por URL real.
when_to_read:
  - cambios en endpoints HTTP
  - cambios en handlers o DTOs
  - cambios en rutas o auth
  - cambios en requests Bruno canónicos
code_paths:
  - internal/server/register_routes.go
  - internal/routes/
  - internal/handlers/
  - internal/dtos/
  - bruno/api/
related_info:
  - doc/info/api/http-endpoints-guide.md
related_specs:
  - doc/specs/documentation-governance-spec.md
status: active
---

# HTTP Endpoints Spec

## Objetivo

Formalizar el contrato documental minimo de la API HTTP para que:

- la documentacion humana y la coleccion Bruno usen la misma taxonomia,
- los endpoints puedan localizarse por URL,
- los request bodies tengan ejemplos consistentes,
- y futuras automatizaciones trabajen sobre contratos verificables.

## Alcance

Aplica a los endpoints registrados en:

- `internal/server/register_routes.go`
- `internal/routes/*.go`

Y a sus artefactos asociados:

- `doc/info/api/http-endpoints-guide.md`
- `bruno/api/`

## Reglas

### 1. Organizacion por URL

- La coleccion Bruno canonica debe agruparse siguiendo la URL real.
- Los endpoints bajo `/api/v1/...` deben vivir bajo `bruno/api/v1/...`.
- Endpoints fuera de `/api/v1`, como `/` u `/oauth/...`, deben ubicarse en carpetas coherentes con su path.

### 2. Request examples

- Todo endpoint con body JSON debe tener un ejemplo de request en la documentacion humana.
- Los endpoints `paginated` deben reutilizar una estructura consistente basada en `PaginationRequest`.
- Los endpoints multipart deben documentar el tipo de body y el nombre del campo archivo.

### 3. Auth

- Debe distinguirse con claridad entre endpoints publicos y protegidos.
- Los endpoints protegidos deben tener request Bruno con bearer token reutilizable por variable.
- La coleccion debe permitir autenticarse y reutilizar `access_token`.
- Los requests operativos deben usar `X-Client-Code: bruno` por defecto, salvo excepción justificada.

### 4. Respuesta base

- La documentacion debe asumir como envoltura base `status`, `message`, `data` y opcionalmente `errors`.
- Las specs no deben congelar payloads internos que cambian por dominio, salvo que el contrato sea estable y relevante.

### 5. Legacy

- Requests historicos o experimentales pueden preservarse en `bruno/legacy/`.
- La coleccion recomendada para uso cotidiano debe ser la canonica por URL.

### 6. Requests reutilizables

- Todo endpoint `POST .../paginated` debe usar una estructura base compatible con `PaginationRequest`.
- Los endpoints multipart deben declarar el campo archivo y las variables requeridas en Bruno.
- Los ejemplos documentales y los requests de Bruno deben derivarse de los DTOs reales cuando existan.

### 7. Endpoints operativos de control

- Si un flujo asincrono o batch necesita cancelacion operativa, la API debe exponer un endpoint claro y documentado para activarla.
- Ese endpoint debe tener request Bruno canonico y ejemplo humano alineado.
- Si el endpoint acepta múltiples formas de resolver la corrida, por ejemplo `run_key` o `bulk_job_id`, la documentación debe dejar claro cuál es el mínimo obligatorio para operar.

## Endpoints minimos cubiertos

La capa documental y Bruno deben cubrir, como minimo:

- root
- app health
- auth publico y auth protegido
- users
- banks
- roles
- menus
- menu-users
- catalogs
- database health
- process-lifecycle
- imports

## Acceptance Criteria

- Existe una guia humana con endpoints agrupados por URL y ejemplos de request body.
- Existe una coleccion Bruno canonica organizada por ruta y no por nombre historico de prueba.
- Los endpoints protegidos reutilizan el token obtenido por login o variable equivalente.
- Los requests historicos no bloquean la navegacion principal porque quedan separados en `legacy`.
- Los endpoints operativos batch relevantes, como cancelacion de corrida, tienen request Bruno canonico alineado con sus DTOs.

## Trazabilidad

- `AGENTS.md`
- `doc/info/api/http-endpoints-guide.md`
- `internal/server/register_routes.go`
- `internal/routes/`
- `bruno/api/`
