# Guía del Makefile

Este documento agrupa el propósito del `Makefile` raíz y organiza sus comandos por dominio funcional para que el equipo pueda descubrir flujos sin leer el archivo completo.

## Objetivo

El `Makefile` funciona como capa operativa unificada para:

- desarrollo local,
- empaquetado y despliegue,
- soporte a infraestructura,
- utilidades de datos,
- integración con AWS y Kubernetes,
- generación de boilerplate.

## Cómo explorar comandos

Comando base:

```bash
make help
```

Catálogo específico de scaffolds:

```bash
make list-scaffolds
```

Catálogo amplio de utilidades:

```bash
make list-tools
```

Comando de diagnóstico rápido:

```bash
make show-all-variables
```

## Contexto de ejecución

Antes de agregar o usar un comando nuevo del `Makefile`, hay que distinguir dónde vive su contexto real de ejecución.

Regla práctica:

- si el comando abre conexiones a Postgres, Redis, colas o servicios que en la configuración aparecen como hosts tipo `postgres` o `redis`, debe ejecutarse dentro de Docker Compose y no asumir que funcionará con `go run` desde host;
- si el comando fue diseñado para correr desde host, su configuración local debe resolver hosts accesibles desde host y esa condición debe quedar documentada explícitamente;
- cuando exista un wrapper operativo del `Makefile`, debe preferirse ese wrapper sobre ejecutar el binario manualmente.

Checklist para futuros comandos Go bajo `cmd/tools/`:

- revisar qué hostnames resuelve `internal/appconfig/config.yml` en el entorno real;
- verificar si el comando comparte contexto con otros targets DB-aware ya existentes;
- usar `$(DC_RUN)` o un wrapper equivalente cuando dependa de la red interna de Compose;
- documentar si el comando soporta host, Docker o ambos;
- evitar exponer como “comando simple” un `go run` que sólo funciona dentro de la red del contenedor.

## Grupos principales de comandos

### 1. Setup y validación

Ejemplos:

- `make check-env`
- `make generate-tfvars`
- `make show-all-variables`
- `make color-messages`

Uso:

- validar entorno,
- preparar variables,
- inspeccionar configuración activa.

### 2. Redis y utilidades de estado

Ejemplos:

- `make redis-list-project-keys`
- `make redis-get-key k="go-fiber-core:lifecycle-2"`
- `make redis-del key="catalogs*"`
- `make create-bulk-job-config process_type_id=13`
- `make cancel-process-run bulk_job_id=2`

Uso:

- depuración,
- limpieza controlada,
- inspección de claves operativas,
- creación rápida de configuraciones `bulk_job_configs` listas para frontend/importación.

### 3. Generación de código y scaffolding

Ejemplos:

- `make create-step name=folder/service_name`
- `make create-export-manager process_name="generar archivo x" service_slug="generar_archivo_x" file="exports/x/y"`
- `make create-batch-process process_name="procesar x" service_slug="procesar_x"`
- `make create-batch-process process_name="procesar x" service_slug="procesar_x" mode=bulk_jobs`
- `make create-batch-process process_name="procesar x" service_slug="procesar_x" type_process=batch-oriented`
- `make create-batch-process process_name="procesar x" service_slug="procesar_x" source_mode=cursor`
- `make create-command name=nuevoComando`
- `make list-example-cases`
- `make recreate-example-case case=process_lifecycle_manager`
- `make list-scaffolds`
- `make list-tools`
- `make delete-process kind=batch-process service_slug=punitorios`
- `make create-migration name=create_users_table`
- `make wire`
- `make wire-sync`

Uso:

- scaffolding de servicios,
- scaffolding de exportadores,
- scaffolding de procesos batch,
- descubrimiento centralizado de scaffolds y generadores relacionados,
- descubrimiento de utilidades operativas agrupadas por dominio,
- recreación y cleanup de casos ejemplo reproducibles,
- limpieza de procesos scaffold,
- generación de código de inyección.

Notas:

- `create-batch-process` ya no genera carpeta Bruno específica por proceso.
- `create-batch-process` genera tres seeders: base `sequential`, companion `_fanout` y companion `_cursor`.
- `create-batch-process` usa `mode=generic` por default y soporta `mode=bulk_jobs` para generar la base funcional tipo `punitorios`.
- `create-batch-process` usa `type_process=item-oriented` por default y soporta `type_process=batch-oriented` para modelar la estrategia del processor.
- `create-batch-process` usa `source_mode=materialized` por default y soporta `source_mode=cursor` para dejar la variante incremental.
- `create-batch-process force=true` permite regenerar un scaffold existente sobrescribiendo los archivos generados.
- `create-batch-process` soporta además la variante técnica `dispatch_pacing` con `pacing=true`.
- `create-batch-process pacing=true` acepta `pacing_messages` y `pacing_interval` para generar el config del `process_batch`.
- En el runtime actual, `source_mode=cursor` conserva `dispatch_pacing`, cancelación y auto-cancel, pero fuerza ejecución secuencial con `parallel_shards=1`.
- `clone-process-version` y `add-process-pacing` se presentan en `list-scaffolds` como operaciones hijas del dominio `batch-process`, no como familias independientes.
- `clone-process-version` es el comando genérico para obtener variantes nuevas como `fanout + pacing` o `sequential + pacing`.
- `add-process-pacing` funciona como wrapper conveniente de `clone-process-version` con `with_pacing=true`.
- `create-export-manager` acepta `service_slug` opcional; si no se envía, se deriva desde `process_name`.
- `create-export-manager force=true` permite regenerar archivos existentes porque el `Makefile` ya propaga ese flag al scaffold.
- `create-export-manager` mantiene un flujo `item-oriented`: `BuildBodyLines(...)` delega en `BodyBuilder.renderItem(...)`.
- preview y run del export reutilizan la misma ruta de render por item, lo que permite testear el archivo sin duplicar lógica.
- `list-scaffolds` funciona como catálogo humano de scaffolds disponibles y debe mantenerse sincronizado cuando se agreguen comandos nuevos de ese tipo.
- El Makefile debe pasar booleanos en formato `-flag=false` para que `go run` no corte el parseo de argumentos antes de flags posteriores como `-force`.
- `create-external-api-config` agrega una entrada `apis.xxx` en `internal/appconfig/config.yml`.
- `create-external-adapter` genera un adapter HTTP reutilizable basado en `apis.xxx` de `config.yml`.
- `create-external-integration` ejecuta ambos pasos en una sola corrida.
- Para pruebas batch se usa la carpeta genérica `bruno/legacy/process-lifecycle/test-batch-process/`.
- Los casos ejemplo reproducibles de process lifecycle se recrean con `create-example-case`, se siembran con `seed-example-case` y generan Bruno bajo `bruno/legacy/process-lifecycle/example-cases/<case>/`.
- `delete-process` soporta `dry_run=true` para revisar el alcance antes de borrar archivos.

## `list-scaffolds`

Uso:

```bash
make list-scaffolds
```

Objetivo:

- listar los scaffolds disponibles,
- mostrar el comando exacto de creación,
- mostrar variantes técnicas y capacidades relevantes como `force=true`,
- resumir qué genera cada uno,
- y recordar el cleanup o comandos relacionados.

Cobertura actual:

- `service-step`
- `batch-process`
- `export-manager`
- `external-api-config`
- `external-adapter`
- `external-integration`
- `cli-command`

Regla de mantenimiento:

- si se agrega un comando nuevo orientado a scaffold o generación reutilizable, debe evaluarse su inclusión en `list-scaffolds`.
- si un scaffold agrega una capacidad importante como `force=true`, un modo técnico como `source_mode=cursor` o una variante relevante como `dispatch_pacing`, `list-scaffolds` y esta guía deben reflejarlo.

## `list-tools`

Uso:

```bash
make list-tools
```

Objetivo:

- ofrecer un mapa corto de utilidades operativas,
- agrupar comandos por dominio,
- y derivar a `list-scaffolds` cuando la necesidad sea generar código base.

Cobertura esperada:

- scaffolds y generadores
- procesos, seeds y cleanup
- redis y estado
- CLI y base de datos
- entorno y diagnóstico
- logs y observabilidad
- Bruno y entorno local

Regla de mantenimiento:

- si se agrega una utilidad operativa de uso humano frecuente, debe evaluarse su inclusión en `list-tools`.
- `list-tools` no reemplaza `make help`; funciona como catálogo resumido y curado.
- Las utilidades `list-example-cases`, `create-example-case`, `seed-example-case`, `recreate-example-case` y `delete-example-case` pertenecen al dominio operativo de ejemplos reproducibles.

## `create-external-api-config`

Uso:

```bash
make create-external-api-config api_key=customer_api
```

Efectos:

- agrega `apis.customer_api` en `internal/appconfig/config.yml`,
- crea placeholders de entorno para `url` y `token`,
- deja `timeout_seconds: 10` como valor inicial.

Ejemplo generado:

```yaml
apis:
  customer_api:
    url: ${CUSTOMER_API_URL}
    token: ${CUSTOMER_API_TOKEN}
    timeout_seconds: 10
```

Reglas:

- falla si `apis.customer_api` ya existe,
- `force=true` permite sobrescribir el bloque,
- el resultado queda listo para resolver con `appConfig.APIConfig("customer_api")`.

## `create-bulk-job-config`

Uso:

```bash
make create-bulk-job-config process_type_id=13
```

Opcional:

```bash
make create-bulk-job-config \
  process_type_id=13 \
  sede_id=0 \
  override_process_version_id=0 \
  roadmap=0
```

Efectos:

- crea un registro en `bulk_job_configs`,
- calcula el siguiente `ref_code` numérico global en saltos de `5`,
- deja `is_active=true`,
- guarda un `config` default compatible con `POST /api/v1/process-lifecycle/run`,
- e imprime el `ref_code` generado para reutilizarlo en el Excel y en `bulk_jobs`.
- el target `make` se ejecuta dentro de Docker Compose para reutilizar la red donde resuelve el hostname `postgres`.

Config default generado:

```json
{
  "process_type_id": 13,
  "sede_id": 0,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {}
}
```

Reglas:

- `operator_id` queda interno con valor default `1`,
- `process_type_id` es obligatorio,
- `sede_id`, `override_process_version_id` y `roadmap` quedan en `0` por defecto,
- el flag `-operator-id` sigue existiendo como override técnico opcional del comando CLI si alguna vez se necesita,
- si ejecutas el binario directamente fuera de Docker, debes asegurarte de que tu configuración local resuelva correctamente los hosts de base de datos,
- para uso operativo normal debe preferirse `make create-bulk-job-config`, no `go run` directo desde host,
- el comando toma solo `ref_code` numéricos existentes para calcular el siguiente consecutivo.

## `cancel-process-run`

Uso por `bulk_job_id`:

```bash
make cancel-process-run bulk_job_id=2
```

Uso por `run_key`:

```bash
make cancel-process-run run_key=bulkjob:2:abc123 reason=manual_cancel
```

Efectos:

- marca la corrida como cancelada en Redis,
- deja razón y origen de cancelación para observabilidad,
- permite que workers asincrónicos dejen de re-encolar `auto_invoke`,
- y sirve tanto para operaciones manuales como para soporte.

## `create-external-adapter`

Uso:

```bash
make create-external-api-config api_key=customer_api
make create-external-adapter adapter_name=customer_api config_key=customer_api
```

Efectos:

- crea `internal/adapters/customer_api_adapter.go`,
- usa `config.ApiConfig`,
- usa `externalhttp.NewClientFromAPIConfig(...)`,
- y deja un método `Do(...)` como base.

Reglas:

- `config_key` debe existir bajo `apis.xxx` en `internal/appconfig/config.yml`,
- nuevos adapters no deben usar `resty.New()` directamente,
- la configuración debe resolverse desde `appConfig.APIConfig("xxx")`.

## `create-external-integration`

Uso:

```bash
make create-external-integration api_key=customer_api
```

Opcional:

```bash
make create-external-integration api_key=customer_api adapter_name=customer_gateway
```

Efectos:

- ejecuta `create-external-api-config`,
- ejecuta `create-external-adapter`,
- deja lista la entrada `apis.xxx`,
- y deja listo el adapter para usar `appConfig.APIConfig("xxx")`.

Reglas:

- si no se informa `adapter_name`, usa `api_key`,
- `force=true` se propaga a ambos pasos,
- conviene usar este comando como flujo por defecto para integraciones nuevas.

### 4. Dependencias, build y calidad

Ejemplos:

- `make vendor`
- `make install-pkg pkg=...`
- `make install-all-pkg`
- `make compile-all`
- `make build-image FOLDER=api`
- `make build-all-images`
- `make lint`
- `make coverage`
- `make coverage-unit`
- `make ci-test`

Uso:

- mantener dependencias,
- compilar artefactos,
- validar calidad de código.

### 5. Desarrollo local y hot reload

Ejemplos:

- `make watch`
- `make dev-local`
- `make localstack-up`
- `make infra-up`

Uso:

- levantar entorno local,
- acelerar iteración,
- preparar dependencias compartidas.

### 6. Lambda, LocalStack y despliegue clásico

Ejemplos:

- `make render-template`
- `make render-templates`
- `make compile-fn FOLDER=api`
- `make package-lambda FOLDER=api`
- `make package-all`
- `make deploy`
- `make deploy-all`
- `make fast-deploy`
- `make fast-deploy-all`
- `make infra-deploy`
- `make infra-destroy`
- `make sam-deploy`

Uso:

- empaquetado lambda,
- despliegue local con Terraform/SAM,
- actualización rápida de funciones.

### 7. EKS y modo híbrido

Ejemplos:

- `make k8s-up`
- `make k8s-down`
- `make watch-eks`
- `make terraform-deploy-eks`
- `make terraform-deploy-lambda`
- `make clean-k8s-apps`
- `make check-k8s`
- `make check-k8s-schedulable`

Uso:

- desarrollo y operación sobre Kubernetes local,
- cambio de modo de despliegue,
- limpieza de releases y verificación del cluster.

### 8. Logs y observabilidad

Ejemplos:

- `make logs-all`
- `make logs-all-k8s`
- `make logs-all-lambda`
- `make logs-docker`
- `make logs-tail`
- `make logs-tail-slow-sql`
- `make logs-tail-slow-sql-cloudwatch`

Uso:

- seguimiento de ejecución,
- slow SQL,
- inspección centralizada por entorno.

### 9. S3 y artefactos

Ejemplos:

- `make s3-check`
- `make s3-ls`
- `make s3-download key=...`
- `make s3-upload key=... file=...`
- `make s3-rm key=...`
- `make s3-rm-dir prefix=...`

Uso:

- validar conectividad,
- subir o bajar archivos,
- limpiar artefactos de forma controlada.

### 10. Base de datos y seeders

Ejemplos:

- `make migrate-up`
- `make migrate-down`
- `make migrate-status`
- `make migrate-reset`
- `make seed`
- `make seed-one name=catalog_items`
- `make seed-list`

Uso:

- evolucionar esquema,
- resetear entorno,
- poblar datos base.

## Comandos sensibles

Requieren especial cuidado:

- `make infra-destroy`
- `make aws-down`
- `make s3-rm`
- `make s3-rm-dir`
- `make migrate-reset`
- `make redis-del`
- `make delete-process`

Antes de ejecutarlos conviene revisar:

- entorno activo,
- variables cargadas,
- prefijos o keys concretas,
- impacto sobre datos o artefactos.

## Recomendación de uso

Si no conoces un comando, primero revisa:

1. `make help`
2. este documento
3. la spec normativa en `doc/specs/platform/makefile-automation-spec.md`

Así evitas usar objetivos destructivos o dependientes de entorno sin contexto suficiente.
