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

Comando de diagnóstico rápido:

```bash
make show-all-variables
```

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

Uso:

- depuración,
- limpieza controlada,
- inspección de claves operativas.

### 3. Generación de código y scaffolding

Ejemplos:

- `make create-step name=folder/service_name`
- `make create-export-manager process_name="generar archivo x" file="exports/x/y"`
- `make create-batch-process process_name="procesar x" service_slug="procesar_x"`
- `make delete-process kind=batch-process service_slug=punitorios`
- `make create-command name=nuevoComando`
- `make create-migration name=create_users_table`
- `make wire`
- `make wire-sync`

Uso:

- scaffolding de servicios,
- scaffolding de exportadores,
- scaffolding de procesos batch,
- limpieza de procesos scaffold,
- generación de código de inyección.

Notas:

- `create-batch-process` ya no genera carpeta Bruno específica por proceso.
- Para pruebas batch se usa la carpeta genérica `bruno/legacy/process-lifecycle/test-batch-process/`.
- `delete-process` soporta `dry_run=true` para revisar el alcance antes de borrar archivos.

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
