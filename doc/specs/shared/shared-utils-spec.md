---
domain: shared
summary: Contrato funcional de helpers reutilizables en internal/utils para conversiones, lectura compartida, CSV, fechas, JSON y filtros comunes.
when_to_read:
  - cambios en helpers compartidos
  - extraccion o consolidacion de utilidades reutilizables
  - refactors que mueven logica comun a internal/utils
code_paths:
  - internal/utils/
  - internal/utils/shared_helpers.go
  - internal/utils/shared_helpers_test.go
related_info:
  - doc/info/development/service-design-conventions.md
  - doc/info/development/create-services-steps.md
related_specs:
  - doc/specs/architecture/service-design-spec.md
status: active
---

# Shared Utils Spec

## Objetivo

Este documento define la especificacion funcional del paquete compartido `internal/utils/shared_helpers.go`.
La intencion es sostener un enfoque de Spec-Driven Development: primero se acuerda el contrato observable de los helpers reutilizables y luego se implementan o refactorizan los consumidores para cumplirlo.

## Alcance

El paquete compartido centraliza helpers que antes estaban repetidos en servicios y layouts, especialmente en:

- `internal/services/generar_archivo_banco_galicia`
- `internal/services/test/bulkexportV2`
- `internal/services/test/bulkexportv1`
- `internal/services/test/mqb1t`

## Especificacion

### 1. Conversiones de valores dinamicos

- `ToInt(value)` debe convertir `int`, `int64`, `float64`, `float32`, `json.Number` y `string`.
- `ToInt64(value)` debe convertir `int`, `int64`, `float64`, `float32`, `json.Number` y `string`.
- Si el valor no es convertible o es invalido, ambas funciones deben devolver `0`.

### 2. Lectura compartida desde `ServiceContext`

- `MustGetInputValue(ctx, key)` debe devolver el valor almacenado o `nil` si la clave no existe.
- `GetInputValueOrDefault(ctx, key, defaultValue)` debe devolver el valor del contexto cuando exista.
- Si la clave no existe, `GetInputValueOrDefault` debe devolver `defaultValue` sin modificarlo.

### 3. Formato CSV

- `BuildCSVLine(fields, comma)` debe serializar una fila CSV valida.
- Si `comma` es `0`, debe usar el separador estandar del writer.
- Si `comma` tiene valor, debe usarlo como delimitador.
- La salida no debe terminar con salto de linea.

### 4. Fechas exportables

- `FormatDate(dateStr, outputFormat)` debe aceptar entradas en `RFC3339`, `YYYY-MM-DD HH:mm:ss` y `YYYY-MM-DD`.
- Debe soportar estos formatos de salida:
  - `YYYY-MM-DD`
  - `DDMMYYYY`
  - `YYYY-MM-DD HH:mm:ss`
  - `HH:mm:ss`
  - `YYYY-MM-DD HH:mm:ss Z`
- Si el formato de salida no esta soportado, debe devolver error.
- Si la fecha no puede parsearse, debe devolver error.

### 5. Extraccion de datos JSON

- `ExtractJSONFields(raw, keys)` debe aceptar:
  - objeto JSON
  - string JSON serializado
  - string en base64 que contiene JSON valido
- Debe devolver los campos en el mismo orden en que llegan en `keys`.
- Si el payload es nulo o vacio, debe devolver valores vacios sin error.

### 6. Filtros de `bulk_job_items`

- `ApplyBulkJobItemFilters(query, rawFilters)` debe normalizar filtros mediante `exportmanager.NormalizeFilters`.
- Debe aplicar filtros sobre:
  - `status_code`
  - `reference_key`
  - `row_number`
  - `id`
  - `bulk_job_id`
- Debe informar si se aplico o no un filtro de `status_code`, porque algunos consumidores usan ese dato para decidir defaults.

## Criterios de aceptacion

- Ningun servicio consumidor debe mantener copias locales de los helpers compartidos incluidos en este contrato.
- Los servicios deben importar el paquete compartido en lugar de redefinir conversiones o builders equivalentes.
- Las pruebas de `internal/utils/shared_helpers_test.go` deben cubrir el comportamiento minimo esperado de conversion, CSV, fechas y payloads JSON.

## Riesgo controlado

- Los wrappers estrictamente de adaptacion por firma pueden existir de forma temporal, pero la logica reusable debe vivir solo en `internal/utils`.
- Si se agrega un nuevo helper reutilizable, primero debe sumarse a esta especificacion o a una especificacion equivalente antes de duplicarse en otro paquete.
