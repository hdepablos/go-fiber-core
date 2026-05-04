# Carga las variables desde el archivo .env (o cae a .env-example) y las exporta.
ENV_FILE ?= .env
ifneq ("$(wildcard $(ENV_FILE))","")
include $(ENV_FILE)
else
ENV_FILE := .env-example
include $(ENV_FILE)
endif
export

GOTOOLCHAIN ?= auto

# .DEFAULT_GOAL define el comando que se ejecuta si solo escribís "make".
.DEFAULT_GOAL := help

###############################################################################
## Diferentes colores para mejorar la legibilidad en la terminal.
###############################################################################
RESET       = \033[0m       # Restablece el color por defecto
INFO        = \033[0;36m    # Cian para información general
SUCCESS     = \033[0;32m    # Verde para operaciones exitosas
WARNING     = \033[0;33m    # Amarillo para advertencias
ERROR       = \033[0;31m    # Rojo para errores críticos
PROMPT      = \033[0;35m    # Magenta para preguntas de usuario
HEADER      = \033[1;34m    # Azul brillante para encabezados
HIGHLIGHT   = \033[1;33m    # Amarillo brillante para destacar algo

###############################################################################
## Variables
###############################################################################
SERVICE_NAME := $(PROJECT_SLUG)
PROJECT_NAME_LOWERCASE := $(subst -, ,$(PROJECT_SLUG))
PROJECT_NAME_LOWERCASE := $(subst _, ,$(PROJECT_NAME_LOWERCASE))
PROJECT_NAME_LOWERCASE := $(strip $(PROJECT_NAME_LOWERCASE))
PROJECT_NAME_LOWERCASE := $(shell echo $(PROJECT_NAME_LOWERCASE) | tr -d ' ' | tr '[:upper:]' '[:lower:]')
PROJECT_NAME_PASCAL := $(shell echo $(PROJECT_SLUG) | awk -F '[-_]' '{for(i=1;i<=NF;i++){printf "%s", toupper(substr($$i,1,1)) tolower(substr($$i,2))}}')
STACK_NAME := $(PROJECT_NAME_LOWERCASE)-stack-$(APP_ENV)
FOLDERS := $(shell echo "$(FUNCTIONS)" | tr ',' ' ')
LOCALSTACK_ENDPOINT_BASE ?= http://127.0.0.1:4566
S3_BUCKET ?= shared-local-dev
AUTO_FIX_K8S_DISK_PRESSURE ?= 1
AUTO_FIX_K8S_DISK_PRESSURE_WAIT_SECONDS ?= 45
AUTO_FIX_K8S_PRUNE_VOLUMES ?= 0
SQS_QUEUE_NAME=${PROJECT_NAME_LOWERCASE}queue
SQS_DLQ_NAME=${PROJECT_NAME_LOWERCASE}dlq
SQS_QUEUE_URL=${LOCALSTACK_ENDPOINT_BASE}/000000000000/${SQS_QUEUE_NAME}
SQS_DLQ_URL=${LOCALSTACK_ENDPOINT_BASE}/000000000000/${SQS_DLQ_NAME}
FUNCTION_NAME_SQS_CONSUMER=${PROJECT_NAME_PASCAL}SqsConsumer
# Variables de Terraform
TF_DIR := ./terraform
TF_VARS := generated.tfvars.json

.PHONY: generate-tfvars
generate-tfvars: ## 🔄 Genera variables de Terraform dinámicamente según el modo (lambda/eks). Uso: make generate-tfvars MODE=lambda ENV_FILE=.env ENVIRONMENT=local
	@ENV_FILE=$${ENV_FILE:-.env}; \
	ENVIRONMENT=$${ENVIRONMENT:-local}; \
	echo "$(INFO)🔄 Generando variables de Terraform para modo: $(MODE) (Env: $$ENVIRONMENT, File: $$ENV_FILE)...$(RESET)"; \
	cd tools/env-manager && go run main.go -mode=$(MODE) -env=../../$$ENV_FILE -output=../../terraform/$(TF_VARS) -environment=$$ENVIRONMENT


ifeq ($(APP_ENV),local)
    SAM_ENDPOINT_ARG=--endpoint-url $(LOCALSTACK_ENDPOINT_BASE)
    AWS_ENDPOINT_ARG=--endpoint-url $(LOCALSTACK_ENDPOINT_BASE)
    AWS_PROFILE_ARG=
else
    SAM_ENDPOINT_ARG=
    AWS_ENDPOINT_ARG=
    AWS_PROFILE_ARG=--profile $(AWS_PROFILE_NAME)
endif

DOCKER_FILE := docker-compose-$(APP_ENV).yml
# Si el archivo no existe, usar el docker-compose por defecto
ifeq ($(wildcard $(DOCKER_FILE)),)
    DOCKER_FILE := docker-compose-local.yml
endif
DC_BASE = docker compose -f docker-compose-base.yml -f $(DOCKER_FILE)
DC_RUN  = $(DC_BASE) run --rm -e GOTOOLCHAIN=$(GOTOOLCHAIN) $(SERVICE_NAME)

LOCALSTACK_DOCKER_FILE := docker-composes/docker-compose.localstack.yml
ifeq ($(wildcard $(LOCALSTACK_DOCKER_FILE)),)
	LOCALSTACK_DOCKER_FILE := docker-composes/docker-compose.localstack-antes.yml
endif

###############################################################################
# Comandos disponibles
###############################################################################
.PHONY: redis-del
redis-del: ## 🧹 Elimina keys de Redis. Si usas *, se restringe al proyecto. Uso: make redis-del key="catalogs*"
	@if [ -z "$(key)" ]; then \
		echo "$(ERROR)❌ Debes especificar la key o patrón. Uso: make redis-del key=\"catalogs*\"$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🔍 Ejecutando limpieza de Redis con patrón: $(key)$(RESET)"
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go redis-clear-keys --pattern "$(key)"

.PHONY: help
help: ## ℹ️ Muestra todos los comandos disponibles con su descripción. Uso: make help
	@awk -F ':|##' '/^[a-zA-Z0-9_-]+:.*?##/ {printf "\033[36m%-20s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST)

.PHONY: show-all-variables
show-all-variables: ## 🔍 Muestra las variables principales del proyecto. Uso: make show-all-variables
	@echo "$(INFO)🔍 Visualizando variables del sistema:$(RESET)"
	@echo "PROJECT_SLUG: $(PROJECT_SLUG)"
	@echo "PROJECT_NAME_LOWERCASE: $(PROJECT_NAME_LOWERCASE)"
	@echo "PROJECT_NAME_PASCAL: $(PROJECT_NAME_PASCAL)"
	@echo "SERVICE_NAME: $(SERVICE_NAME)"
	@echo "DOCKER_FILE: $(DOCKER_FILE)"
	@echo "DC_BASE: $(DC_BASE)"
	@echo "DC_RUN: $(DC_RUN)"
	@echo "FUNCTION_NAME_SQS_CONSUMER: $(FUNCTION_NAME_SQS_CONSUMER)"

.PHONY: color-messages
color-messages: ## 🎨 Ejemplos de los diferentes colores de mensajes. Uso: make color-messages
	@echo "$(RESET) RESET  🚀 Color del mensaje$(RESET)"
	@echo "$(INFO)      INFO  🚀 Color del mensaje$(RESET)"
	@echo "$(SUCCESS)       SUCCESS  🚀 Color del mensaje$(RESET)"
	@echo "$(WARNING)       WARNING  🚀 Color del mensaje$(RESET)"
	@echo "$(ERROR)     ERROR  🚀 Color del mensaje$(RESET)"
	@echo "$(PROMPT)        PROMPT  🚀 Color del mensaje$(RESET)"
	@echo "$(HEADER)        HEADER  🚀 Color del mensaje$(RESET)"
	@echo "$(HIGHLIGHT)     HIGHLIGHT  🚀 Color del mensaje$(RESET)"

.PHONY: check-env
check-env: ## ⚖️ Verifica que existan las variables de entorno indispensables. Uso: make check-env
	@echo "$(INFO)Verificando variables de entorno en el .env indispensables para el Makefile$(RESET)"
	@if [ -z "$(APP_ENV)" ]; then echo "❌ APP_ENV no está definido en .env"; exit 1; fi
	@if [ -z "$(PROJECT_SLUG)" ]; then echo "❌ PROJECT_SLUGS no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_ACCESS_SECRET)" ]; then echo "❌ JWT_ACCESS_SECRET no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_REFRESH_SECRET)" ]; then echo "❌ JWT_REFRESH_SECRET no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_ACCESS_TTL_MINUTES)" ]; then echo "❌ JWT_ACCESS_TTL_MINUTES no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_REFRESH_TTL_DAYS)" ]; then echo "❌ JWT_REFRESH_TTL_DAYS no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_ACCESS_SECRET)" ]; then echo "❌ JWT_ACCESS_SECRET no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_REFRESH_SECRET)" ]; then echo "❌ JWT_REFRESH_SECRET no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_ACCESS_TTL_MINUTES)" ]; then echo "❌ JWT_ACCESS_TTL_MINUTES no está definido en .env"; exit 1; fi
	@if [ -z "$(JWT_REFRESH_TTL_DAYS)" ]; then echo "❌ JWT_REFRESH_TTL_DAYS no está definido en .env"; exit 1; fi
	@if [ -z "$(REDIS_HOST)" ]; then echo "❌ REDIS_HOST no está definido en .env"; exit 1; fi
	@if [ -z "$(REDIS_PORT)" ]; then echo "❌ REDIS_PORT no está definido en .env"; exit 1; fi
	@if [ -z "$(REDIS_PASSWORD)" ]; then echo "❌ REDIS_PASSWORD no está definido en .env"; exit 1; fi
	@if [ -z "$(REDIS_DATABASE)" ]; then echo "❌ REDIS_DATABASE no está definido en .env"; exit 1; fi
	@if [ -z "$(REDIS_EXPIRES_IN_SECONDS)" ]; then echo "❌ REDIS_EXPIRES_IN_SECONDS no está definido en .env"; exit 1; fi

	@echo "✅ Todas las variables de entorno están definidas en .env"

.PHONY: redis-list-project-keys
redis-list-project-keys: ## 🧱 Lista todas las keys de Redis del proyecto (prefijo APP_NAME o go-fiber-core). Uso: make redis-list-project-keys
	@RAW_PREFIX=$${APP_NAME:-go-fiber-core}; \
	APP_PREFIX=$$(echo "$$RAW_PREFIX" | tr -d '"'); \
	echo "$(INFO)Listando keys de Redis para el proyecto con prefijo: $$APP_PREFIX$(RESET)"; \
	if ! command -v redis-cli >/dev/null 2>&1; then \
		echo "$(ERROR)redis-cli no está instalado en el sistema. Instálalo para usar este comando.$(RESET)"; \
		exit 1; \
	fi; \
	if [ -z "$(REDIS_HOST)" ] || [ -z "$(REDIS_PORT)" ] || [ -z "$(REDIS_DATABASE)" ]; then \
		echo "$(ERROR)Variables REDIS_HOST, REDIS_PORT o REDIS_DATABASE no están definidas.$(RESET)"; \
		exit 1; \
	fi; \
	HOST="$(REDIS_HOST)"; \
	if [ "$$HOST" = "redis" ]; then HOST="127.0.0.1"; fi; \
	REDISCLI_AUTH="$(REDIS_PASSWORD)" redis-cli -h "$$HOST" -p "$(REDIS_PORT)" -n "$(REDIS_DATABASE)" KEYS "$$APP_PREFIX:*"

.PHONY: redis-get-key
# redis-get-key: ## 🔎 Muestra el contenido de una key de Redis. Uso: make redis-get-key k="go-fiber-core:lifecycle-2"
redis-get-key: ## 🔎 Muestra el contenido de una key de Redis. Uso: make redis-get-key k="go-fiber-core:lifecycle-2"
	@if [ -z "$(k)" ]; then \
		echo "$(ERROR)Debes pasar el nombre de la key con k=\"nombre_key\".$(RESET)"; \
		exit 1; \
	fi; \
	if ! command -v redis-cli >/dev/null 2>&1; then \
		echo "$(ERROR)redis-cli no está instalado en el sistema. Instálalo para usar este comando.$(RESET)"; \
		exit 1; \
	fi; \
	if [ -z "$(REDIS_HOST)" ] || [ -z "$(REDIS_PORT)" ] || [ -z "$(REDIS_DATABASE)" ]; then \
		echo "$(ERROR)Variables REDIS_HOST, REDIS_PORT o REDIS_DATABASE no están definidas.$(RESET)"; \
		exit 1; \
	fi; \
	HOST="$(REDIS_HOST)"; \
	if [ "$$HOST" = "redis" ]; then HOST="127.0.0.1"; fi; \
	echo "$(INFO)Mostrando contenido de la key: $(k)$(RESET)"; \
	REDISCLI_AUTH="$(REDIS_PASSWORD)" redis-cli -h "$$HOST" -p "$(REDIS_PORT)" -n "$(REDIS_DATABASE)" GET "$(k)"

.PHONY: create-step
create-step: ## 👣 Crea un nuevo servicio (Step) con boilerplate y auto-wiring. Uso: make create-step name=folder/service_name
	@if [ -z "$(name)" ]; then \
		echo "$(ERROR)❌ Debes especificar el nombre: make create-step name=folder/service_name$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🚀 Creando servicio $(name)...$(RESET)"
	@go run tools/make-service/main.go -name "$(name)"
	@echo "$(SUCCESS)✨ Servicio creado e inyectado correctamente.$(RESET)"
	@echo "$(INFO)📝 Ahora edita el archivo generado para implementar tu lógica.$(RESET)"

.PHONY: create-export-manager
create-export-manager: ## 🧩 Genera un scaffold de exportmanager. Uso: make create-export-manager process_name="generar archivo x" [service_slug=generar_archivo_x] file="exports/x/y"
	@if [ -z "$(process_name)" ]; then \
		echo "$(ERROR)❌ Debes especificar process_name: make create-export-manager process_name=\"generar archivo x\" [service_slug=generar_archivo_x] file=\"exports/x/y\"$(RESET)"; \
		exit 1; \
	fi
	@if [ -z "$(file)" ]; then \
		echo "$(ERROR)❌ Debes especificar file: make create-export-manager process_name=\"generar archivo x\" [service_slug=generar_archivo_x] file=\"exports/x/y\"$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🚀 Generando scaffold exportmanager para $(process_name)...$(RESET)"
	@go run ./cmd/tools/export-manager-scaffold \
		-process-name "$(process_name)" \
		-service-slug "$(service_slug)" \
		-file "$(file)" \
		-batch-size "$(or $(batch_size),5000)" \
		-part-prefix "$(or $(part_prefix),)" \
		-redis-ttl-hours "$(or $(redis_ttl_hours),24)" \
		-bulk-job-id "$(or $(bulk_job_id),0)" \
		$(if $(filter true,$(force)),-force,)
	@echo "$(SUCCESS)✨ Scaffold exportmanager generado correctamente.$(RESET)"

.PHONY: create-batch-process
create-batch-process: ## 🧩 Genera un scaffold de batchflow. Uso: make create-batch-process process_name="procesar x" [service_slug=procesar_x] [mode=generic|bulk_jobs] [type_process=item-oriented|batch-oriented] [source_mode=materialized|cursor] [pacing=true pacing_messages=100 pacing_interval=2]
	@if [ -z "$(process_name)" ]; then \
		echo "$(ERROR)❌ Debes especificar process_name: make create-batch-process process_name=\"procesar x\" [service_slug=procesar_x]$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🚀 Generando scaffold batchflow para $(process_name)...$(RESET)"
	@go run ./cmd/tools/batch-process-scaffold \
		-process-name "$(process_name)" \
		-service-slug "$(service_slug)" \
		-mode "$(or $(mode),generic)" \
		-type-process "$(or $(type_process),item-oriented)" \
		-source-mode "$(or $(source_mode),materialized)" \
		-batch-size "$(or $(batch_size),500)" \
		-concurrent-batches "$(or $(concurrent_batches),1)" \
		-parallel-shards "$(or $(parallel_shards),4)" \
		-redis-ttl-hours "$(or $(redis_ttl_hours),24)" \
		-with-pacing=$(if $(filter true,$(pacing)),true,false) \
		-pacing-messages "$(or $(pacing_messages),100)" \
		-pacing-interval "$(or $(pacing_interval),2)" \
		-with-bruno=false \
		$(if $(filter true,$(force)),-force,)
	@echo "$(SUCCESS)✨ Scaffold batchflow generado correctamente.$(RESET)"

.PHONY: list-example-cases
list-example-cases: ## 🧪 Lista los casos ejemplo reproducibles de process lifecycle. Uso: make list-example-cases
	@go run ./cmd/tools/example-case-manager -action list -case all -repo-root . -verbose

.PHONY: create-example-case
create-example-case: ## 🧪 Recrea servicios y Bruno de un caso ejemplo. Uso: make create-example-case case=process_lifecycle_manager|all
	@echo "$(INFO)🧪 Creando caso ejemplo $(or $(case),all)...$(RESET)"
	@go run ./cmd/tools/example-case-manager -action create -case "$(or $(case),all)" -repo-root .
	@echo "$(SUCCESS)✨ Caso ejemplo creado correctamente.$(RESET)"

.PHONY: delete-example-case
delete-example-case: ## 🧹 Elimina servicios y Bruno de un caso ejemplo. Uso: make delete-example-case case=process_lifecycle_manager|all
	@echo "$(INFO)🧹 Eliminando caso ejemplo $(or $(case),all)...$(RESET)"
	@go run ./cmd/tools/example-case-manager -action delete -case "$(or $(case),all)" -repo-root .
	@echo "$(SUCCESS)✨ Caso ejemplo eliminado correctamente.$(RESET)"

.PHONY: seed-example-case
seed-example-case: ## 🌱 Ejecuta el seeder asociado a un caso ejemplo. Uso: make seed-example-case case=process_lifecycle_manager|all
	@if [ -z "$(case)" ] || [ "$(case)" = "all" ]; then \
		for c in process_lifecycle_manager test_process_scenarios process_lifecycle_auto_invoke multi_queue_batch_one_table_process_lifecycle multi_queue_batch_one_table_recreate_records; do \
			$(MAKE) seed-one name=$$c; \
		done; \
	else \
		$(MAKE) seed-one name=$(case); \
	fi

.PHONY: recreate-example-case
recreate-example-case: ## 🔁 Elimina, recrea y siembra un caso ejemplo. Uso: make recreate-example-case case=process_lifecycle_manager|all
	@$(MAKE) delete-example-case case="$(or $(case),all)"
	@$(MAKE) create-example-case case="$(or $(case),all)"
	@$(MAKE) seed-example-case case="$(or $(case),all)"

.PHONY: add-process-pacing
add-process-pacing: ## ⏱️ Clona una process_version existente y agrega dispatch_pacing al step process_batch. Uso: make add-process-pacing source_version_id=2 operator_id=1 pacing_messages=100 pacing_interval=2
	@echo "$(INFO)⏱️ Clonando version $(source_version_id) y agregando dispatch_pacing...$(RESET)"
	@go run ./cmd/tools/clone-process-version \
		-config "$(or $(config),internal/appconfig/config.yml)" \
		-source-version-id "$(source_version_id)" \
		-operator-id "$(operator_id)" \
		-with-pacing=true \
		-pacing-messages "$(or $(pacing_messages),100)" \
		-pacing-interval "$(or $(pacing_interval),2)" \
		$(if $(process_batch_step),-process-batch-step "$(process_batch_step)",)
	@echo "$(SUCCESS)✨ Nueva version clonada con dispatch_pacing.$(RESET)"

.PHONY: clone-process-version
clone-process-version: ## 🧬 Clona una process_version existente, con opcion de aplicar dispatch_pacing. Uso: make clone-process-version source_version_id=19 operator_id=1 [with_pacing=true pacing_messages=100 pacing_interval=2]
	@echo "$(INFO)🧬 Clonando version $(source_version_id)...$(RESET)"
	@go run ./cmd/tools/clone-process-version \
		-config "$(or $(config),internal/appconfig/config.yml)" \
		-source-version-id "$(source_version_id)" \
		-operator-id "$(operator_id)" \
		-with-pacing=$(if $(filter true,$(with_pacing)),true,false) \
		-pacing-messages "$(or $(pacing_messages),100)" \
		-pacing-interval "$(or $(pacing_interval),2)" \
		$(if $(process_batch_step),-process-batch-step "$(process_batch_step)",)
	@echo "$(SUCCESS)✨ Process version clonada correctamente.$(RESET)"

.PHONY: list-scaffolds
list-scaffolds: ## 📚 Lista los comandos tipo scaffold y generadores relacionados. Uso: make list-scaffolds
	@echo "$(INFO)Scaffolds disponibles$(RESET)"
	@echo ""
	@echo "1. service-step"
	@echo "   Comando:"
	@echo "   make create-step name=carpeta/servicio"
	@echo ""
	@echo "   Genera:"
	@echo "   - servicio step en internal/services/<ruta>"
	@echo "   - registro automatico en serviceconfig.Register"
	@echo "   - auto-wiring de imports en cmd/api/main.go"
	@echo "   - auto-wiring de imports en cmd/cmd-cli/main.go"
	@echo ""
	@echo "   Opciones importantes:"
	@echo "   - no soporta force=true; si el archivo existe, falla para evitar sobrescritura"
	@echo ""
	@echo "2. batch-process"
	@echo "   Comando:"
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\""
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" force=true"
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" mode=bulk_jobs"
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" type_process=batch-oriented"
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" source_mode=cursor"
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" mode=bulk_jobs source_mode=cursor"
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" mode=bulk_jobs type_process=item-oriented"
	@echo "   make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" pacing=true pacing_messages=100 pacing_interval=2"
	@echo ""
	@echo "   Variantes:"
	@echo "   - generic: modo por defecto; deja provider/processor/lifecycle comentados para adaptar otra tabla padre/hija"
	@echo "   - source_mode=materialized: modo default; materializa todos los batches en Redis durante start"
	@echo "   - source_mode=cursor: agrega la variante companion _cursor y deja stubs/implementacion incremental por pagina"
	@echo "   - type_process=item-oriented: default; el developer implementa processItemOriented(...)"
	@echo "   - type_process=batch-oriented: el developer implementa processBatchOriented(...)"
	@echo "   - sequential: version base generada automaticamente"
	@echo "   - fanout: version companion _fanout generada automaticamente"
	@echo "   - cursor: version companion _cursor generada automaticamente para corrida incremental secuencial"
	@echo "   - ejemplo cursor: make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" source_mode=cursor"
	@echo "   - dispatch_pacing: variante opcional generable via pacing=true"
	@echo "   - bulk_jobs: genera el scaffold funcional tipo punitorios sobre bulk_jobs/bulk_job_items"
	@echo "     Implementacion esperada del modo bulk_jobs:"
	@echo "     - input.id = bulk_job_id"
	@echo "     - DataProvider consulta bulk_job_items por bulk_job_id"
	@echo "     - ParentLifecycle opera sobre bulk_jobs"
	@echo "     - Finalize calcula progreso y pendientes desde bulk_job_items"
	@echo "     - ProcessBatch actualiza status y mensajes de bulk_job_items"
	@echo ""
	@echo "   Genera:"
	@echo "   - provider.go"
	@echo "   - carpeta del servicio en internal/services/batchprocess/<service_slug>/"
	@echo "   - runtime/provider_context.go"
	@echo "   - data/provider.go"
	@echo "   - processor/processor.go"
	@echo "   - lifecycle/parent.go"
	@echo "   - lifecycle/finalizer.go"
	@echo "   - steps/start.go"
	@echo "   - steps/dispatch_shards.go"
	@echo "   - steps/process_batch.go"
	@echo "   - steps/finalize.go"
	@echo "   - steps/input.go"
	@echo "   - steps/failure.go"
	@echo "   - steps/helpers.go"
	@echo "   - seeder base sequential"
	@echo "   - seeder companion _fanout"
	@echo "   - seeder companion _cursor"
	@echo ""
	@echo "   Opciones importantes:"
	@echo "   - mode=generic|bulk_jobs: generic es el default; bulk_jobs genera la base funcional tipo punitorios"
	@echo "   - type_process=item-oriented|batch-oriented: define la estrategia del processor; item-oriented es el default"
	@echo "   - source_mode=materialized|cursor: materialized es el default; cursor deja habilitada la ruta incremental"
	@echo "   - force=true: regenera y sobrescribe archivos scaffold existentes"
	@echo "   - pacing=true: agrega dispatch_pacing al step process_batch"
	@echo "   - pacing_messages=<n>: items por invocacion cuando pacing=true"
	@echo "   - pacing_interval=<1..10>: delay entre auto_invoke cuando pacing=true"
	@echo "   - modo bulk_jobs: usar input.id como bulk_job_id al ejecutar preview/run y resolver la data desde bulk_job_items"
	@echo ""
	@echo "   Operaciones hijas sobre versiones existentes:"
	@echo "   - make clone-process-version source_version_id=19 operator_id=1"
	@echo "   - make clone-process-version source_version_id=19 operator_id=1 with_pacing=true pacing_messages=100 pacing_interval=2"
	@echo "   - make add-process-pacing source_version_id=19 operator_id=1 pacing_messages=100 pacing_interval=2"
	@echo ""
	@echo "   Batch versioning:"
	@echo "   - clone-process-version: clona una process_version existente"
	@echo "   - add-process-pacing: wrapper de clone-process-version con with_pacing=true"
	@echo ""
	@echo "   Cleanup:"
	@echo "   make delete-process kind=batch-process service_slug=procesar_x"
	@echo ""
	@echo "3. export-manager"
	@echo "   Comando:"
	@echo "   make create-export-manager process_name=\"generar archivo x\" service_slug=\"generar_archivo_x\" file=\"exports/x/y\""
	@echo "   make create-export-manager process_name=\"generar archivo x\" service_slug=\"generar_archivo_x\" file=\"exports/x/y\" force=true"
	@echo "   make create-export-manager process_name=\"generar archivo x\" service_slug=\"generar_archivo_x\" file=\"exports/x/y\" bulk_job_id=2"
	@echo ""
	@echo "   Variantes:"
	@echo "   - service_slug: opcional; si no se envia, se deriva desde process_name"
	@echo "   - item-oriented por contrato: el BodyBuilder delega la logica del registro en renderItem(...)"
	@echo "   - preview y run reutilizan la misma ruta de render desde BodyBuilder.renderItem(...)"
	@echo "   - layout default funcional: header/body/footer CSV con ';', columnas historicas y helper layout_helpers.go"
	@echo "   - generico: deja provider/lifecycle/output para personalizar"
	@echo "   - bulk_jobs: se activa con bulk_job_id=<id> y deja el scaffold funcional sobre bulk_jobs"
	@echo "     Implementacion esperada del modo bulk_jobs:"
	@echo "     - input.id = bulk_job_id"
	@echo "     - DataProvider consulta bulk_job_items por bulk_job_id"
	@echo "     - ParentLifecycle opera sobre bulk_jobs"
	@echo "     - OutputRegistrar registra en bulk_job_outputs"
	@echo ""
	@echo "   Genera:"
	@echo "   - carpeta del servicio en internal/services/exports/<service_slug>/"
	@echo "   - provider.go"
	@echo "   - runtime/provider_context.go"
	@echo "   - data/provider.go"
	@echo "   - layout/header_builder.go"
	@echo "   - layout/body_builder.go"
	@echo "   - layout/footer_builder.go"
	@echo "   - layout/layout_helpers.go"
	@echo "   - lifecycle/parent.go"
	@echo "   - lifecycle/output_registrar.go"
	@echo "   - steps/start.go"
	@echo "   - steps/process_batch.go"
	@echo "   - steps/finalize.go"
	@echo "   - steps/input.go"
	@echo "   - steps/failure.go"
	@echo "   - seeder"
	@echo "   - request Bruno dedicado"
	@echo "   - documentacion base"
	@echo ""
	@echo "   Opciones importantes:"
	@echo "   - force=true: regenera y sobrescribe archivos generados si existen"
	@echo "   - bulk_job_id=<id>: activa el modo bulk_jobs y genera el scaffold funcional sobre bulk_jobs/bulk_job_items"
	@echo ""
	@echo "   Cleanup:"
	@echo "   make delete-process kind=export service_slug=generar_archivo_x"
	@echo ""
	@echo "4. external-api-config"
	@echo "   Comando:"
	@echo "   make create-external-api-config api_key=customer_api"
	@echo "   make create-external-api-config api_key=customer_api force=true"
	@echo ""
	@echo "   Genera:"
	@echo "   - entrada apis.xxx en internal/appconfig/config.yml"
	@echo "   - placeholders de entorno base"
	@echo ""
	@echo "   Opciones importantes:"
	@echo "   - force=true: sobrescribe la entrada apis.xxx si ya existe"
	@echo ""
	@echo "5. external-adapter"
	@echo "   Comando:"
	@echo "   make create-external-adapter adapter_name=customer_api config_key=customer_api"
	@echo "   make create-external-adapter adapter_name=customer_api config_key=customer_api force=true"
	@echo ""
	@echo "   Genera:"
	@echo "   - adapter HTTP externo reusable"
	@echo "   - base alineada con externalhttp y config.ApiConfig"
	@echo ""
	@echo "   Opciones importantes:"
	@echo "   - force=true: sobrescribe el archivo generado si ya existe"
	@echo ""
	@echo "6. external-integration"
	@echo "   Comando:"
	@echo "   make create-external-integration api_key=customer_api [adapter_name=customer_api]"
	@echo "   make create-external-integration api_key=customer_api force=true"
	@echo ""
	@echo "   Genera:"
	@echo "   - config apis.xxx"
	@echo "   - adapter HTTP externo"
	@echo ""
	@echo "   Opciones importantes:"
	@echo "   - force=true: se propaga a config y adapter"
	@echo ""
	@echo "7. cli-command"
	@echo "   Comando:"
	@echo "   make create-command name=nuevoComando"
	@echo ""
	@echo "   Genera:"
	@echo "   - comando Cobra bajo cmd/cmd-cli/cmd/"
	@echo "   - alta inicial via cobra-cli add"
	@echo ""
	@echo "   Opciones importantes:"
	@echo "   - no soporta force=true"
	@echo "   - pensado para ampliar la CLI interna, no para servicios batch"
	@echo ""
	@echo "Comandos relacionados"
	@echo "   - make delete-process kind=batch-process service_slug=..."
	@echo "   - make delete-process kind=export service_slug=..."
	@echo "   - make seed-one name=..."
	@echo "   - make cli-help"
	@echo "   - make help"
	@echo ""
	@echo "Documentacion:"
	@echo "   - doc/info/development/process-scaffold-and-cleanup.md"
	@echo "   - doc/info/platform/makefile-guide.md"

.PHONY: list-tools
list-tools: ## 🧰 Lista utilidades operativas agrupadas por dominio. Uso: make list-tools
	@echo "$(INFO)Herramientas operativas$(RESET)"
	@echo ""
	@echo "1. Scaffolds y generadores"
	@echo "   - make list-scaffolds"
	@echo "   - make create-step name=carpeta/servicio"
	@echo "   - make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\""
	@echo "   - make create-batch-process process_name=\"procesar x\" service_slug=\"procesar_x\" source_mode=cursor"
	@echo "   - make clone-process-version source_version_id=19 operator_id=1 with_pacing=true pacing_messages=100 pacing_interval=2"
	@echo "   - make add-process-pacing source_version_id=2 operator_id=1 pacing_messages=100 pacing_interval=2"
	@echo "   - make create-export-manager process_name=\"generar archivo x\" service_slug=\"generar_archivo_x\" file=\"exports/x/y\""
	@echo "   - make create-external-integration api_key=customer_api"
	@echo "   - make create-command name=nuevoComando"
	@echo ""
	@echo "2. Casos ejemplo"
	@echo "   - make list-example-cases"
	@echo "   - make create-example-case case=process_lifecycle_manager"
	@echo "   - make seed-example-case case=process_lifecycle_manager"
	@echo "   - make recreate-example-case case=process_lifecycle_manager"
	@echo "   - make delete-example-case case=process_lifecycle_manager"
	@echo ""
	@echo "3. Procesos, seeds y cleanup"
	@echo "   - make seed-list"
	@echo "   - make seed-one name=..."
	@echo "   - make delete-process kind=batch-process service_slug=..."
	@echo "   - make delete-process kind=export service_slug=... dry_run=true"
	@echo ""
	@echo "4. Redis y estado"
	@echo "   - make redis-list-project-keys"
	@echo "   - make redis-get-key k=\"go-fiber-core:lifecycle-2\""
	@echo "   - make redis-del key=\"catalogs*\""
	@echo ""
	@echo "4. CLI y base de datos"
	@echo "   - make cli-help"
	@echo "   - make run-cli c=\"redis-clear-keys --pattern go-fiber-core:*\""
	@echo "   - make create-bulk-job-config process_type_id=13"
	@echo "   - make cancel-process-run bulk_job_id=2"
	@echo "   - make create-migration name=create_users_table"
	@echo "   - make migrate-status"
	@echo "   - make migrate-up"
	@echo ""
	@echo "5. Entorno y diagnostico"
	@echo "   - make help"
	@echo "   - make show-all-variables"
	@echo "   - make check-env"
	@echo "   - make color-messages"
	@echo ""
	@echo "6. Logs y observabilidad"
	@echo "   - make logs-tail service=api since=1h"
	@echo "   - make logs-tail-slow-sql"
	@echo "   - make logs-tail-slow-sql-cloudwatch service=api since=1h"
	@echo "   - make logs-all"
	@echo ""
	@echo "7. Bruno y entorno local"
	@echo "   - make update-bruno-url-base"
	@echo "   - make update-bruno-lambda"
	@echo "   - make update-bruno-eks"
	@echo "   - make update-bruno"
	@echo ""
	@echo "Documentacion:"
	@echo "   - doc/info/platform/makefile-guide.md"
	@echo "   - doc/info/development/process-scaffold-and-cleanup.md"

.PHONY: create-external-adapter
create-external-adapter: ## 🌐 Genera un scaffold de adapter HTTP externo. Uso: make create-external-adapter adapter_name=customer_api config_key=customer_api
	@if [ -z "$(adapter_name)" ]; then \
		echo "$(ERROR)❌ Debes especificar adapter_name: make create-external-adapter adapter_name=customer_api config_key=customer_api$(RESET)"; \
		exit 1; \
	fi
	@if [ -z "$(config_key)" ]; then \
		echo "$(ERROR)❌ Debes especificar config_key: make create-external-adapter adapter_name=customer_api config_key=customer_api$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🌐 Generando scaffold de adapter externo $(adapter_name)...$(RESET)"
	@go run ./cmd/tools/external-api-adapter-scaffold \
		-adapter-name "$(adapter_name)" \
		-config-key "$(config_key)" \
		$(if $(filter true,$(force)),-force,)
	@echo "$(SUCCESS)✨ Scaffold de adapter externo generado correctamente.$(RESET)"

.PHONY: create-external-api-config
create-external-api-config: ## ⚙️ Agrega una entrada apis.xxx en config.yml. Uso: make create-external-api-config api_key=customer_api
	@if [ -z "$(api_key)" ]; then \
		echo "$(ERROR)❌ Debes especificar api_key: make create-external-api-config api_key=customer_api$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)⚙️ Generando entrada apis.$(api_key) en config.yml...$(RESET)"
	@go run ./cmd/tools/external-api-config-scaffold \
		-api-key "$(api_key)" \
		$(if $(filter true,$(force)),-force,)
	@echo "$(SUCCESS)✨ Config API externa agregada correctamente.$(RESET)"

.PHONY: create-external-integration
create-external-integration: ## 🌐⚙️ Genera config apis.xxx y adapter HTTP externo. Uso: make create-external-integration api_key=customer_api [adapter_name=customer_api]
	@if [ -z "$(api_key)" ]; then \
		echo "$(ERROR)❌ Debes especificar api_key: make create-external-integration api_key=customer_api [adapter_name=customer_api]$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🌐⚙️ Generando integracion externa $(or $(adapter_name),$(api_key))...$(RESET)"
	@$(MAKE) create-external-api-config api_key="$(api_key)" force="$(force)"
	@$(MAKE) create-external-adapter adapter_name="$(or $(adapter_name),$(api_key))" config_key="$(api_key)" force="$(force)"
	@echo "$(SUCCESS)✨ Integracion externa generada correctamente.$(RESET)"

.PHONY: delete-process
delete-process: ## 🧹 Elimina del codigo un proceso scaffold. Uso: make delete-process kind=batch-process service_slug=punitorios
	@if [ -z "$(kind)" ]; then \
		echo "$(ERROR)❌ Debes especificar kind: make delete-process kind=batch-process service_slug=punitorios$(RESET)"; \
		exit 1; \
	fi
	@if [ -z "$(service_slug)" ]; then \
		echo "$(ERROR)❌ Debes especificar service_slug: make delete-process kind=batch-process service_slug=punitorios$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🧹 Eliminando proceso $(service_slug) de tipo $(kind)...$(RESET)"
	@go run ./cmd/tools/process-cleanup \
		-kind "$(kind)" \
		-service-slug "$(service_slug)" \
		$(if $(filter true,$(dry_run)),-dry-run,)
	@echo "$(SUCCESS)✨ Limpieza del proceso completada.$(RESET)"

###############################################################################
## Golang
###############################################################################
.PHONY: vendor
vendor: ## 📦 Actualiza el archivo go.mod y la carpeta vendor. Uso: make vendor
	@echo "$(SUCCESS)📦 Ordenando y vendoring dependencias...$(RESET)"
	@$(DC_RUN) go mod tidy
	@$(DC_RUN) go mod vendor

.PHONY: install-pkg
install-pkg: ## 📥 Instala un paquete Go específico. Uso: make install-pkg pkg=github.com/ejemplo/modulo
	@echo "$(SUCCESS)📥 Instalando/actualizando paquete: $(pkg)...$(RESET)"
	@$(DC_RUN) go get -u $(pkg)
	@$(MAKE) vendor

.PHONY: install-all-pkg
install-all-pkg: ## 🗂️ Instala todas las dependencias Go necesarias del proyecto. Uso: make install-all-pkg
	@echo "$(INFO)🗂️ Instalando todas las dependencias...$(RESET)"
	@$(DC_BASE) build $(SERVICE_NAME)
	@$(DC_RUN) sh -c 'go get -u github.com/golang-jwt/jwt/v5 golang.org/x/crypto/bcrypt github.com/redis/go-redis/v9 gorm.io/gorm gorm.io/driver/postgres github.com/jackc/pgx/v5 github.com/spf13/viper github.com/gofiber/fiber/v2 github.com/gofiber/fiber/v2/middleware/limiter github.com/gofiber/fiber/v2/middleware/cors github.com/spf13/cobra github.com/robfig/cron/v3 gopkg.in/gomail.v2 github.com/natefinch/lumberjack github.com/russross/blackfriday/v2 github.com/go-resty/resty/v2 github.com/mitchellh/mapstructure github.com/go-playground/locales github.com/go-playground/universal-translator github.com/alicebob/miniredis/v2 github.com/DATA-DOG/go-sqlmock github.com/stretchr/testify/mock github.com/go-playground/locales/es github.com/go-playground/validator/v10 github.com/go-playground/validator/v10/translations/es github.com/aws/aws-sdk-go-v2/aws github.com/aws/aws-sdk-go-v2/service/sns github.com/aws/aws-sdk-go-v2/service/sqs github.com/aws/aws-lambda-go/events github.com/aws/aws-lambda-go/lambda github.com/aws/aws-sdk-go-v2/config && go mod tidy && go mod download && go mod vendor'

.PHONY: wire
wire: ## 🧬 Genera el código de inyección de dependencias con Google Wire. Uso: make wire
	@echo "$(SUCCESS)🧬 Generando inyección de dependencias con Wire...$(RESET)"
	@$(DC_BASE) build $(SERVICE_NAME)
	@$(DC_RUN) sh -c 'go install github.com/google/wire/cmd/wire@latest && wire gen -tags wireinject ./cmd/api/di'

.PHONY: wire-sync
wire-sync: ## 🧬📦 Genera código de Wire y actualiza vendor. Uso: make wire-sync
	@$(MAKE) wire
	@$(MAKE) vendor
	@echo "$(SUCCESS)✅ Proceso de Wire y vendor completado.$(RESET)"

###############################################################################
## S3 Utilities
###############################################################################
.PHONY: s3-check
s3-check: ## ✅ Verifica conectividad y permisos de subida a S3 (AWS o LocalStack). Uso: make s3-check [bucket=mi-bucket] [endpoint=URL]
	@if ! command -v aws >/dev/null 2>&1; then \
		echo "$(ERROR)❌ aws cli no está instalado o no está en el PATH.$(RESET)"; \
		exit 1; \
	fi
	@BUCKET=$${bucket:-$(S3_BUCKET)}; \
	BUCKET=$$(echo "$$BUCKET" | tr -d '"'); \
	if [ -z "$$BUCKET" ]; then \
		echo "$(ERROR)❌ No se detectó bucket. Usa bucket=mi-bucket o define S3_BUCKET en .env$(RESET)"; \
		exit 1; \
	fi; \
	ENDPOINT=$${endpoint:-$${AWS_ENDPOINT_URL:-$${LOCALSTACK_ENDPOINT_BASE}}}; \
	ENDPOINT=$$(echo "$$ENDPOINT" | tr -d '"'); \
	ENDPOINT_ARG=""; \
	if [ -n "$$ENDPOINT" ]; then ENDPOINT_ARG="--endpoint-url $$ENDPOINT"; fi; \
	REGION=$${AWS_DEFAULT_REGION:-us-east-1}; \
	KEY=$${key:-healthcheck/$(PROJECT_SLUG)/$$(date +%Y%m%dT%H%M%S)-$$RANDOM.txt}; \
	TMPFILE=$${tmpfile:-tmp/s3_healthcheck_$$(date +%s)_$$RANDOM.txt}; \
	mkdir -p tmp; \
	printf "s3-healthcheck %s\n" "$$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$$TMPFILE"; \
	echo "$(INFO)🔎 Probando subida a s3://$$BUCKET/$$KEY ...$(RESET)"; \
	IS_LOCALSTACK=0; \
	if [ -n "$$ENDPOINT" ] && echo "$$ENDPOINT" | grep -Eq '(localhost|127\\.0\\.0\\.1).*:4566|localstack'; then IS_LOCALSTACK=1; fi; \
	if [ "$$IS_LOCALSTACK" = "1" ]; then \
		if ! aws s3api list-buckets $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region "$$REGION" >/dev/null 2>&1; then \
			rm -f "$$TMPFILE"; \
			echo "$(ERROR)❌ LocalStack no responde para S3 en $$ENDPOINT$(RESET)"; \
			echo "$(INFO)Ejecuta esto en el proyecto LocalStack: cd /private/var/www/localstack && make aws-up$(RESET)"; \
			exit 1; \
		fi; \
	fi; \
	if ! aws s3api head-bucket --bucket "$$BUCKET" $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region "$$REGION" >/dev/null 2>&1; then \
		rm -f "$$TMPFILE"; \
		echo "$(ERROR)❌ No se pudo acceder al bucket '$$BUCKET' (head-bucket falló).$(RESET)"; \
		if [ "$$IS_LOCALSTACK" = "1" ]; then \
			echo "$(INFO)Crea el bucket en el proyecto LocalStack: cd /private/var/www/localstack && make bucket-create name=$$BUCKET$(RESET)"; \
		fi; \
		exit 1; \
	fi; \
	if ! aws s3 cp "$$TMPFILE" "s3://$$BUCKET/$$KEY" $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region "$$REGION" >/dev/null 2>&1; then \
		rm -f "$$TMPFILE"; \
		echo "$(ERROR)❌ Falló la subida a S3 (aws s3 cp). Revisa credenciales/permisos/endpoint.$(RESET)"; \
		exit 1; \
	fi; \
	aws s3 rm "s3://$$BUCKET/$$KEY" $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region "$$REGION" >/dev/null 2>&1 || true; \
	rm -f "$$TMPFILE"; \
	echo "$(SUCCESS)✅ Conexión OK: se pudo subir (y limpiar) un objeto en S3.$(RESET)"

.PHONY: s3-ls
s3-ls: ## 🪣 Lista todo el contenido del S3. Uso: make s3-ls [bucket=mi-bucket]
	@BUCKET=$${bucket:-$(S3_BUCKET)}; \
	ENDPOINT=$${endpoint:-$${AWS_ENDPOINT_URL:-$${LOCALSTACK_ENDPOINT_BASE}}}; \
	ENDPOINT_ARG=""; \
	if [ -n "$$ENDPOINT" ]; then ENDPOINT_ARG="--endpoint-url $$ENDPOINT"; fi; \
	echo "$(INFO)🪣 Listando contenido recursivo de s3://$$BUCKET...$(RESET)"; \
	aws s3 ls s3://$$BUCKET --recursive $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region $(AWS_DEFAULT_REGION)

.PHONY: s3-download
s3-download: ## ⬇️ Descarga un archivo de S3. Uso: make s3-download key=ruta/al/archivo.csv
	@if [ -z "$(key)" ]; then \
		echo "$(ERROR)❌ Debes pasar la ruta del archivo con key=\"ruta/al/archivo\".$(RESET)"; \
		exit 1; \
	fi; \
	BUCKET=$${bucket:-$(S3_BUCKET)}; \
	ENDPOINT=$${endpoint:-$${AWS_ENDPOINT_URL:-$${LOCALSTACK_ENDPOINT_BASE}}}; \
	ENDPOINT_ARG=""; \
	if [ -n "$$ENDPOINT" ]; then ENDPOINT_ARG="--endpoint-url $$ENDPOINT"; fi; \
	DEST="tmp/s3_downloads/$$(basename $(key))"; \
	mkdir -p tmp/s3_downloads; \
	echo "$(INFO)⬇️ Descargando s3://$$BUCKET/$(key) en $$DEST...$(RESET)"; \
	aws s3 cp s3://$$BUCKET/$(key) $$DEST $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region $(AWS_DEFAULT_REGION); \
	echo "$(SUCCESS)✅ Archivo descargado en: $$DEST$(RESET)"

.PHONY: s3-upload
s3-upload: ## ⬆️ Sube un archivo a S3. Uso: make s3-upload key=ruta/destino file=local/file
	@if [ -z "$(key)" ]; then \
		echo "$(ERROR)❌ Debes pasar el destino con key=\"ruta/destino\".$(RESET)"; \
		exit 1; \
	fi; \
	SRC=$${file:-payload.json}; \
	if [ ! -f "$$SRC" ]; then \
		echo "$(ERROR)❌ No existe el archivo local: $$SRC$(RESET)"; \
		exit 1; \
	fi; \
	BUCKET=$${bucket:-$(S3_BUCKET)}; \
	ENDPOINT=$${endpoint:-$${AWS_ENDPOINT_URL:-$${LOCALSTACK_ENDPOINT_BASE}}}; \
	ENDPOINT_ARG=""; \
	if [ -n "$$ENDPOINT" ]; then ENDPOINT_ARG="--endpoint-url $$ENDPOINT"; fi; \
	echo "$(INFO)⬆️ Subiendo $$SRC a s3://$$BUCKET/$(key)...$(RESET)"; \
	aws s3 cp "$$SRC" "s3://$$BUCKET/$(key)" $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region $(AWS_DEFAULT_REGION); \
	echo "$(SUCCESS)✅ Archivo subido.$(RESET)"

.PHONY: s3-rm
s3-rm: ## 🗑️ Elimina un archivo de S3 (pide confirmación). Uso: make s3-rm key=ruta/archivo.csv
	@if [ -z "$(key)" ]; then \
		echo "$(ERROR)❌ Debes pasar la ruta del archivo con key=\"ruta/archivo\".$(RESET)"; \
		exit 1; \
	fi; \
	BUCKET=$${bucket:-$(S3_BUCKET)}; \
	ENDPOINT=$${endpoint:-$${AWS_ENDPOINT_URL:-$${LOCALSTACK_ENDPOINT_BASE}}}; \
	ENDPOINT_ARG=""; \
	if [ -n "$$ENDPOINT" ]; then ENDPOINT_ARG="--endpoint-url $$ENDPOINT"; fi; \
	echo "$(PROMPT)⚠️  ¿Estás seguro de eliminar el archivo s3://$$BUCKET/$(key)? [y/N]: $(RESET)" && read ans && \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
		echo "$(INFO)🗑️ Eliminando s3://$$BUCKET/$(key)...$(RESET)"; \
		aws s3 rm s3://$$BUCKET/$(key) $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region $(AWS_DEFAULT_REGION); \
		echo "$(SUCCESS)✅ Archivo eliminado.$(RESET)"; \
	else \
		echo "$(WARNING)🛑 Operación cancelada.$(RESET)"; \
	fi

.PHONY: s3-rm-dir
s3-rm-dir: ## 💥 Elimina una carpeta completa de S3 (pide confirmación). Uso: make s3-rm-dir prefix=ruta/carpeta/
	@if [ -z "$(prefix)" ]; then \
		echo "$(ERROR)❌ Debes pasar el prefijo/carpeta con prefix=\"ruta/carpeta/\".$(RESET)"; \
		exit 1; \
	fi; \
	BUCKET=$${bucket:-$(S3_BUCKET)}; \
	ENDPOINT=$${endpoint:-$${AWS_ENDPOINT_URL:-$${LOCALSTACK_ENDPOINT_BASE}}}; \
	ENDPOINT_ARG=""; \
	if [ -n "$$ENDPOINT" ]; then ENDPOINT_ARG="--endpoint-url $$ENDPOINT"; fi; \
	echo "$(WARNING)💥 ADVERTENCIA: Se eliminará todo bajo s3://$$BUCKET/$(prefix)$(RESET)"; \
	echo "$(PROMPT)⚠️  ¿Estás completamente seguro? [y/N]: $(RESET)" && read ans && \
	if [ "$$ans" = "y" ] || [ "$$ans" = "Y" ]; then \
		echo "$(INFO)🗑️ Eliminando recursivamente s3://$$BUCKET/$(prefix)...$(RESET)"; \
		aws s3 rm s3://$$BUCKET/$(prefix) --recursive $$ENDPOINT_ARG $(AWS_PROFILE_ARG) --region $(AWS_DEFAULT_REGION); \
		echo "$(SUCCESS)✅ Carpeta eliminada.$(RESET)"; \
	else \
		echo "$(WARNING)🛑 Operación cancelada.$(RESET)"; \
	fi

###############################################################################
## AWS
###############################################################################
.PHONY: logs-tail
logs-tail: ## 📜 Sigue los logs de un servicio en CloudWatch. Uso: make logs-tail service=api since=1h
	@if [ -z "$(service)" ]; then echo "$(ERROR)❌ Debes pasar el parámetro 'service', ej: make logs-tail service=api$(RESET)"; exit 1; fi
	@GROUP="/app/$(PROJECT_SLUG)/$(service)"; \
	SINCE=$${since:-1h}; \
	echo "$(INFO)📜 Tail del log group: $$GROUP (desde $$SINCE)$(RESET)"; \
	aws logs tail "$$GROUP" $(AWS_ENDPOINT_ARG) $(AWS_PROFILE_ARG) --follow --since "$$SINCE"

.PHONY: logs-tail-slow-sql
logs-tail-slow-sql: ## 🐢📜 Sigue solo las queries lentas locales (archivo DB_SLOW_LOG_FILE). Uso: make logs-tail-slow-sql
	@LOG_FILE="$${DB_SLOW_LOG_FILE:-pkg/logs/db-slow.log}"; \
	echo "$(INFO)🐢📜 Tail de slow SQL local: $$LOG_FILE$(RESET)"; \
	mkdir -p "$$(dirname "$$LOG_FILE")"; \
	touch "$$LOG_FILE"; \
	tail -n 200 -f "$$LOG_FILE"

.PHONY: logs-tail-slow-sql-cloudwatch
logs-tail-slow-sql-cloudwatch: ## 🐢☁️ Sigue slow SQL en CloudWatch filtrando por 'SLOW SQL'. Uso: make logs-tail-slow-sql-cloudwatch service=api since=1h
	@if [ -z "$(service)" ]; then echo "$(ERROR)❌ Debes pasar 'service', ej: make logs-tail-slow-sql-cloudwatch service=api$(RESET)"; exit 1; fi
	@GROUP="/app/$(PROJECT_SLUG)/$(service)"; \
	SINCE=$${since:-1h}; \
	echo "$(INFO)🐢☁️ Tail de SLOW SQL en: $$GROUP (desde $$SINCE)$(RESET)"; \
	aws logs tail "$$GROUP" $(AWS_ENDPOINT_ARG) $(AWS_PROFILE_ARG) --follow --since "$$SINCE" --filter-pattern "SLOW SQL"

.PHONY: logs-all-docker
logs-all-docker: ## 📜 Muestra logs unificados de Docker Compose. Uso: make logs-all-docker
	@echo "$(INFO)📜 Obteniendo logs de todos los servicios...$(RESET)"
	@$(DC_BASE) logs -f

.PHONY: send-message
send-message: check-env ## 📨 Envía un mensaje de prueba a la cola SQS. Uso: make send-message
	@echo "$(INFO)🚀 Enviando mensaje de prueba a SQS...$(RESET)"
	@go run cmd/tools/send_msg/main.go

.PHONY: send-message-error
send-message-error: check-env ## 📨 Envía un mensaje de prueba con error a la cola SQS. Uso: make send-message-error
	@echo "$(INFO)🚀 Enviando mensaje de prueba con error a SQS...$(RESET)"
	@go run cmd/tools/send_msg/main.go -error=true

.PHONY: test-api-aws
test-api-aws: ## 🧪 Realiza pruebas sobre la API Gateway de LocalStack. Uso: make test-api-aws
	@echo "$(INFO)🧪 Obteniendo endpoint de la API...$(RESET)"
	@API_ENDPOINT=$$(aws --profile $(AWS_PROFILE_NAME) cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--endpoint-url=$(LOCALSTACK_ENDPOINT_BASE) \
		--query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" \
		--output text); \
	if [ -z "$$API_ENDPOINT" ] || [ "$$API_ENDPOINT" = "None" ]; then \
		echo "$(ERROR)🚨 Error: No se pudo obtener el endpoint.$(RESET)"; \
		exit 1; \
	fi; \
	echo "$(INFO)🌐 Endpoint detectado: $$API_ENDPOINT$(RESET)"

.PHONY: test-loop
test-loop: ## 🔄 Ejecuta 'make send-message' varias veces. Uso: make test-loop
	@echo "$(INFO)🔄 Iniciando ráfaga secuencial de 6 mensajes...$(RESET)"
	@for i in $$(seq 1 25); do \
		echo "$(INFO)📦 Mensaje iteración $$i:$(RESET)"; \
		$(MAKE) send-message; \
		echo "-------------------------------------------"; \
	done
	@echo "$(SUCCESS)✅ Ráfaga completada.$(RESET)"

.PHONY: test-aws
.PHONY: test-aws-all
test-aws-all: ## 🧪🧬 Realiza pruebas integrales sobre API y SQS. Uso: make test-aws-all
	@echo "$(INFO)🧪 Iniciando pruebas integrales...$(RESET)"
	@export RAW_URL=$$(aws --profile $(AWS_PROFILE_NAME) cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--endpoint-url=$(LOCALSTACK_ENDPOINT_BASE) \
		--query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" \
		--output text); \
	export ID=$$(echo $$RAW_URL | cut -d'/' -f5); \
	export ENDPOINT="http://$$ID.execute-api.localhost.localstack.cloud:4566/Prod"; \
	echo "$(INFO)🌐 Endpoint final: $$ENDPOINT $(RESET)"; \
	curl -s -X POST "$$ENDPOINT/messages" \
		-H "Content-Type: application/json" \
		-d '{"id": "test-1", "content": "Mensaje desde Makefile"}' && echo " ✅ POST exitoso"; \
	aws --profile $(AWS_PROFILE_NAME) sqs get-queue-attributes \
		--endpoint-url=$(LOCALSTACK_ENDPOINT_BASE) \
		--queue-url $(SQS_QUEUE_URL) \
		--attribute-names ApproximateNumberOfMessages \
		--output table

.PHONY: coverage
coverage: ## 📊 Genera reporte de cobertura COMPLETO (unitarios + integración). Uso: make coverage
	@chmod +x ./scripts/generate_coverage_report.sh
	@echo "📊 Generando reporte de cobertura COMPLETO..."
	@$(DC_RUN) go test -tags=integration -coverprofile=coverage.out ./...
	@$(DC_RUN) bash ./scripts/generate_coverage_report.sh

.PHONY: coverage-unit
coverage-unit: ## 📊 Genera reporte de cobertura RÁPIDO (solo unitarios). Uso: make coverage-unit
	@chmod +x ./scripts/generate_coverage_report.sh
	@echo "📊 Generando reporte de cobertura para tests UNITARIOS..."
	@$(DC_RUN) go test -coverprofile=coverage.out ./...
	@$(DC_RUN) bash ./scripts/generate_coverage_report.sh


.PHONY: lint
lint: ## 🎨 Analiza el código en busca de errores y malas prácticas con golangci-lint. Uso: make lint
	@echo "🧹 Limpiando la caché de golangci-lint..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint cache clean
	@echo "Limpiando contenedores huérfanos..."
	@docker compose -f docker-compose-local-lint.yml down --remove-orphans
	@echo "Ejecutando linter..."
	@docker compose -f docker-compose-local-lint.yml build --no-cache lint && docker compose -f docker-compose-local-lint.yml run --rm lint run

.PHONY: lint-check-config
lint-check-config: ## 🔍 Verifica qué archivos está usando golangci-lint. Uso: make lint-check-config
	@echo "🔍 Verificando configuración de golangci-lint..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint config path
	@echo ""
	@echo "📄 Mostrando configuración cargada:"
	@docker compose -f docker-compose-local-lint.yml run --rm lint config dump

.PHONY: lint-verbose
lint-verbose: ## 🔍 Ejecuta el linter en modo verbose para ver qué archivos analiza. Uso: make lint-verbose
	@echo "🔍 Ejecutando linter en modo verbose..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint run -v

.PHONY: lint-test
lint-test: ## 🧪 Prueba si wire_gen.go está siendo ignorado. Uso: make lint-test
	@echo "🧪 Listando archivos que el linter va a analizar..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint run --issues-exit-code=0 2>&1 | grep -i "wire_gen" || echo "✅ wire_gen.go NO aparece en la salida (está siendo ignorado)"

.PHONY: localstack-up
localstack-up: ## 🛠️ Levanta LocalStack en segundo plano. Uso: make localstack-up
	@echo "$(SUCCESS)🛠️ Iniciando LocalStack...$(RESET)"
	@docker compose -p localstack -f $(LOCALSTACK_DOCKER_FILE) up -d --build --force-recreate
	@sleep 10
	@echo "$(SUCCESS)✅ LocalStack listo.$(RESET)"

.PHONY: render-template
render-template: ## 📄 Genera un template SAM basado en stubs. Uso: make render-template folder=api
	@service_name=$$(echo "$(PROJECT_NAME_PASCAL)-$(folder)" | tr "-" " " | awk '{ for (i=1; i<=NF; i++) printf toupper(substr($$i,1,1)) substr($$i,2) }'); \
	if [ "$(folder)" = "api" ]; then stub="stubs/api-lambda.stub"; \
	elif echo "$(folder)" | grep -q -- "-cron$$"; then stub="stubs/cron-lambda.stub"; \
	else stub="stubs/$(folder)-lambda.stub"; fi; \
	mkdir -p templates; \
	sed -e "s|__PROJECT__|$(PROJECT)|g" -e "s|__SERVICE_NAME__|$$service_name|g" -e "s|__PROJECT_LOWER__|$(PROJECT_NAME_LOWERCASE)|g" -e "s|__FOLDER__|$(folder)|g" $$stub > templates/$(folder)-template.yml; \
	echo "$(SUCCESS)📄 Template generado para $(folder)$(RESET)"


.PHONY: render-templates
render-templates: ## 📄📄 Genera todos los templates del proyecto. Uso: make render-templates
	@for folder in $(FOLDERS); do $(MAKE) render-template folder=$$folder; done
	@$(MAKE) render-template folder=sqs-queues


.PHONY: delete-templates
delete-templates: ## 🗑️ Elimina los templates generados. Uso: make delete-templates
	@rm -f templates/*.yml
	@echo "$(SUCCESS)🗑️ Templates eliminados.$(RESET)"

.PHONY: update-api-base
update-api-base: ## 🔗 Obtiene la URL de API Gateway en LocalStack y la guarda en .api_base_tmp. Uso: make update-api-base
	@echo "🔗 Obteniendo API Gateway URL (LocalStack)..."
	@API_ID=$$(aws --profile $(AWS_PROFILE_NAME) apigateway get-rest-apis \
		--endpoint-url=$(LOCALSTACK_ENDPOINT_BASE) \
		--query "items[0].id" \
		--output text); \
	echo "http://localhost:4566/restapis/$$API_ID/Prod/_user_request_/" > .api_base_tmp; \
	echo "✔ API URL: http://localhost:4566/restapis/$$API_ID/Prod/_user_request_/"

.PHONY: update-env-url-base
update-env-url-base: ## ✏️ Actualiza la URL_BASE en el archivo .env. Uso: make update-env-url-base
	@$(MAKE) update-api-base
	@API_BASE=$$(cat .api_base_tmp); \
	if [ "$$(uname)" = "Darwin" ]; then sed -i '' -E "s|^URL_BASE=.*|URL_BASE=$$API_BASE|" .env; \
	else sed -i -E "s|^URL_BASE=.*|URL_BASE=$$API_BASE|" .env; fi

	@echo "$(INFO)🌐 Nueva URL_BASE: $$API_BASE$(RESET)"

	@echo "$(SUCCESS)✅ URL_BASE actualizada en .env$(RESET)"


.PHONY: set-env
set-env: ## ✏️ Setea APP_ENV. Uso: make set-env ENV=local | ENV=lambda
	@if [ -z "$(ENV)" ]; then \
		echo "❌ Debes pasar ENV=local | ENV=lambda"; \
		exit 1; \
	fi; \
	if grep -q '^APP_ENV=' .env; then \
		if [ "$$(uname)" = "Darwin" ]; then \
			sed -i '' -E "s|^APP_ENV=.*|APP_ENV=$(ENV)|" .env; \
		else \
			sed -i -E "s|^APP_ENV=.*|APP_ENV=$(ENV)|" .env; \
		fi; \
	else \
		echo "APP_ENV=$(ENV)" >> .env; \
	fi
	@echo "$(SUCCESS)✅ APP_ENV=$(ENV)$(RESET)"

.PHONY: update-bruno-url-base
update-bruno-url-base: ## ✏️ Actualiza urlBase en Bruno según ENV (local | lambda). Uso: make update-bruno-url-base ENV=local
	@if [ -z "$(ENV)" ]; then \
		echo "$(ERROR)❌ Debes pasar ENV=local o ENV=lambda$(RESET)"; \
		exit 1; \
	fi; \
	BRUNO_ENV_FILE="bruno/environments/$(ENV).bru"; \
	if [ ! -f "$$BRUNO_ENV_FILE" ]; then \
		echo "$(ERROR)❌ No existe $$BRUNO_ENV_FILE$(RESET)"; \
		exit 1; \
	fi; \
	if [ "$(ENV)" = "lambda" ]; then \
		if [ ! -f .api_base_tmp ]; then echo "❌ Faltan .api_base_tmp"; exit 1; fi; \
		API_BASE=$$(cat .api_base_tmp); \
	else \
		API_BASE="http://127.0.0.1:$(SERVER_PORT)/"; \
	fi; \
	if [ "$$(uname)" = "Darwin" ]; then \
		sed -i '' -E "s|urlBase: .*|urlBase: $$API_BASE|" "$$BRUNO_ENV_FILE"; \
	else \
		sed -i -E "s|urlBase: .*|urlBase: $$API_BASE|" "$$BRUNO_ENV_FILE"; \
	fi; \
	echo "$(SUCCESS)✅ urlBase actualizada a [$$API_BASE] en Bruno ($(ENV))$(RESET)"


.PHONY: update-url-all
update-url-all: ## ✏️✏️ Sincroniza la URL en .env y Bruno. Uso: make update-url-all ENV=lambda
	@$(MAKE) update-env-url-base
	@$(MAKE) update-bruno-url-base

.PHONY: watch-lambda
watch-lambda: check-env infra-up ## 🚀👀 Actualiza código, infraestructura y bruno (sin bloquear). Uso: make watch-lambda
	@$(MAKE) fast-deploy-all
	@$(MAKE) infra-deploy MODE=lambda
	@$(MAKE) update-bruno-lambda
	@echo "$(INFO)📺 Ejecuta 'make logs-all' si quieres Observar los logs de las funciones$(RESET)"

.PHONY: update-bruno-lambda
update-bruno-lambda: ## 🦁 Actualiza la URL de la API en Bruno (Lambda). Uso: make update-bruno-lambda
	@echo "$(INFO)🔄 Actualizando URL en Bruno (Lambda)...$(RESET)"
	@API_ID=$$(awslocal apigateway get-rest-apis --query "items[0].id" --output text); \
	if [ -z "$$API_ID" ] || [ "$$API_ID" = "None" ]; then \
		echo "$(ERROR)❌ No se encontró API Gateway ID.$(RESET)"; \
	else \
		sed -i '' "s|urlBase: http://localhost:4566/restapis/[a-z0-9]*/Prod/_user_request_/|urlBase: http://localhost:4566/restapis/$$API_ID/Prod/_user_request_/|g" bruno/environments/lambda.bru; \
		echo "$(SUCCESS)✅ Bruno actualizado con API ID: $$API_ID$(RESET)"; \
	fi

.PHONY: update-bruno
update-bruno: update-bruno-lambda ## 🦁 Alias para update-bruno-lambda.



.PHONY: write-api-base
write-api-base: ## 📝 Extrae la URL desde el output de Terraform y la guarda en .api_base_tmp. Uso: make write-api-base
	@echo "$(INFO)🔗 Obteniendo URL desde Terraform...$(RESET)"
	@API_BASE=$$(cd terraform && tflocal output -raw api_base_url 2>/dev/null); \
	if [ -z "$$API_BASE" ] || [ "$$API_BASE" = "None" ]; then \
		echo "$(ERROR)❌ No se pudo obtener la URL. Revisa el archivo outputs.tf$(RESET)"; \
		exit 1; \
	fi; \
	echo "$$API_BASE" > .api_base_tmp
	@echo "$(SUCCESS)✅ URL guardada en .api_base_tmp$(RESET)"


.PHONY: update-function
update-function: ## ⚙️ Recompila, construye con SAM y DESPLIEGA automáticamente. Uso: make update-function FOLDER=api
	@if [ -z "$(FOLDER)" ]; then \
		echo "$(ERROR)❌ Debes indicar la función: make update-function FOLDER=nombre-carpeta$(RESET)"; \
		exit 1; \
	fi
	@echo "$(INFO)🏗️ Iniciando actualización completa para [$(FOLDER)]...$(RESET)"

	@# 1. Compilar el binario fresco
	@$(MAKE) compile-fn FOLDER=$(FOLDER)

	@# 2. Ejecutar SAM Build y, si tiene éxito, ejecutar SAM Deploy inmediatamente
	@sam build --template master-template.yml && $(MAKE) sam-deploy

	@echo "$(SUCCESS)✅ Proceso de actualización y despliegue finalizado para $(FOLDER).$(RESET)"


.PHONY: compile-fn
compile-fn: ## 🏗️ Compila el binario y genera el ZIP para Terraform. Uso: make compile-fn FOLDER=api
	@echo "$(INFO)🏗️ Compilando [$(FOLDER)]...$(RESET)"
	$(eval OUT_DIR := $(shell pwd)/sam-compile/$(FOLDER))
	$(eval IMAGE_TAG := lambda-$(FOLDER):latest)

	# 1. Construir la imagen de Docker
	docker build --no-cache --build-arg FOLDER=$(FOLDER) --build-arg FUNC_NAME=$(FOLDER) -f dockerfiles/Dockerfile.func.lambda -t $(IMAGE_TAG) .

	# 2. Limpieza y preparación de directorios
	@rm -rf $(OUT_DIR) && mkdir -p $(OUT_DIR)

	# 3. Extraer el binario y archivos desde el contenedor
	@docker rm -f temp_$(FOLDER) 2>/dev/null || true
	@docker create --name temp_$(FOLDER) $(IMAGE_TAG)
	@docker cp temp_$(FOLDER):/app/$(FOLDER)/. $(OUT_DIR)/
	@docker rm temp_$(FOLDER) > /dev/null

	# 4. Generar Makefile para SAM (Tu lógica original)
	@$(eval FUNC_PASCAL := $(shell echo "$(FOLDER)" | awk -F '-' '{for(i=1;i<=NF;i++) printf toupper(substr($$i,1,1)) substr($$i,2)}'))
	@$(eval LOGICAL_ID := $(PROJECT_NAME_PASCAL)$(FUNC_PASCAL))
	@printf "build-$(LOGICAL_ID):\n\tcp -r * \$$(ARTIFACTS_DIR)/\n\tchmod +x \$$(ARTIFACTS_DIR)/bootstrap\n" > $(OUT_DIR)/Makefile

	# 5. Generar ZIP para Terraform
	@cd $(OUT_DIR) && zip -r ../$(FOLDER).zip .
	@echo "$(SUCCESS)📦 ZIP generado en sam-compile/$(FOLDER).zip$(RESET)"

	# 5. 📦 Generar el ZIP para Terraform/LocalStack
	@echo "$(INFO)📦 Empaquetando ZIP para Terraform...$(RESET)"
	@cd $(OUT_DIR) && \
		chmod +x bootstrap && \
		zip -q -r ../$(FOLDER).zip .

	@echo "$(SUCCESS)🚀 ZIP listo en: sam-compile/$(FOLDER).zip$(RESET)"


###############################################################################
## TERRAFORM + LOCALSTACK
###############################################################################

# 1. Compilar y Desplegar TODO el stack (Infrastructure + All Functions)
.PHONY: deploy-all
deploy-all: ## 🌎 Compila todas las funciones y despliega toda la infraestructura. Uso: make deploy-all
	@echo "$(INFO)🚀 Desplegando stack completo...$(RESET)"
	@$(MAKE) compile-fn FOLDER=api
	@$(MAKE) compile-fn FOLDER=sqs-consumer
	@$(MAKE) compile-fn FOLDER=dlq-consumer
	@$(MAKE) compile-fn FOLDER=every-1min-cron
	@$(MAKE) compile-fn FOLDER=daily-24-cron
	@$(MAKE) infra-deploy

# 2. Compilar y Desplegar una SOLA función (Hot-reload)
.PHONY: deploy
deploy: infra-init ## ⚡ Compila y actualiza una sola función (Uso: make deploy FOLDER=api)
	@echo "$(INFO)🔄 Actualizando componente: [$(FOLDER)]...$(RESET)"
	@$(MAKE) compile-fn FOLDER=$(FOLDER)
	@$(MAKE) infra-deploy

.PHONY: infra-init
infra-init: infra-up ## 🏁 Inicializa Terraform/LocalStack. Uso: make infra-init
	@echo "$(INFO)🚀 Inicializando Terraform con Backend S3 (LocalStack)...$(RESET)"
	@cd $(TF_DIR) && tflocal init -backend-config=backend.local.conf -reconfigure

.PHONY: infra-deploy
infra-deploy: infra-up generate-tfvars ## 🚀 Despliega toda la infraestructura en LocalStack. Uso: make infra-deploy
	@if [ ! -d "$(TF_DIR)/.terraform" ]; then $(MAKE) infra-init; fi
	@cd $(TF_DIR) && tflocal apply -var-file=$(TF_VARS) -auto-approve

.PHONY: infra-destroy
infra-destroy: ## 💣 Destruye la infraestructura en LocalStack. Uso: make infra-destroy
	@cd $(TF_DIR) && tflocal destroy -var-file=$(TF_VARS) -auto-approve

.PHONY: deploy-full
deploy-full: compile-all infra-deploy ## ⚡ Compila y despliega en un solo paso. Uso: make deploy-full
	@echo "$(SUCCESS)🔥 Todo el stack ha sido actualizado en LocalStack$(RESET)"

.PHONY: infra-logs
infra-logs: ## 📜 Muestra logs de una función específica. Uso: make infra-logs FOLDER=api
	@$(eval FUNC_NAME := $(shell echo $(FOLDER) | sed 's/every-1min-cron/1min-cron/' | sed 's/daily-24-cron/daily-cron/'))
	@echo "🔍 Siguiendo logs de: gofibercore-local-$(FUNC_NAME)..."
	@awslocal logs tail /aws/lambda/gofibercore-local-$(FUNC_NAME) --follow

.PHONY: logs-lambdas
logs-lambdas: ## 📊 Sigue logs de las funciones Lambda (LocalStack).
	@echo "📺 Observando logs de las funciones (Lambda)... (Ctrl+C para detener)"
	@sh -c '\
		trap "kill 0 2>/dev/null || true; exit 0" INT TERM; \
		awslocal logs tail /aws/lambda/gofibercore-local-api --follow & \
		awslocal logs tail /aws/lambda/gofibercore-local-sqs-consumer --follow & \
		awslocal logs tail /aws/lambda/gofibercore-local-1min-cron --follow & \
		awslocal logs tail /aws/lambda/gofibercore-local-daily-cron --follow & \
		wait \
	'

.PHONY: logs-docker
logs-docker: ## 🐳 Sigue logs de Docker Compose.
	@$(DC_BASE) logs -f

.PHONY: logs-all-lambda
logs-all-lambda: logs-lambdas ## 📊 Sigue logs de Lambda (LocalStack). Uso: make logs-all-lambda

.PHONY: logs-all-k8s
logs-all-k8s: ## 📊 Sigue logs de los pods en K8s (API + Consumers). Uso: make logs-all-k8s
	@echo "📺 Observando logs de pods K8s (api, sqs-consumer)..."
	@sh -c '\
		trap "kill 0" INT TERM; \
		kubectl logs -l app=api -f --prefix=true & \
		kubectl logs -l app=sqs-consumer -f --prefix=true & \
		wait \
	'

.PHONY: logs-all
logs-all: ## 📊 Logs unificados (auto-detecta: k8s | lambda | docker). Uso: make logs-all MODE=eks|lambda|docker
	@MODE_IN="$(MODE)"; \
	if [ -z "$$MODE_IN" ]; then \
		if command -v kubectl >/dev/null 2>&1 && kubectl get pods >/dev/null 2>&1; then \
			MODE_IN="eks"; \
		elif command -v awslocal >/dev/null 2>&1; then \
			MODE_IN="lambda"; \
		else \
			MODE_IN="docker"; \
		fi; \
	fi; \
	case "$$MODE_IN" in \
		eks) $(MAKE) logs-all-k8s ;; \
		lambda) $(MAKE) logs-all-lambda ;; \
		docker|local) $(MAKE) logs-all-docker ;; \
		*) echo "$(ERROR)❌ MODE inválido: $$MODE_IN (usa eks|lambda|docker)$(RESET)"; exit 1 ;; \
	esac


.PHONY: package-lambda
package-lambda: ## 📦 Empaqueta una función Lambda en ZIP (Nativo). Uso: make package-lambda FOLDER=api
	@echo "$(INFO)📦 Empaquetando Lambda [$(FOLDER)]...$(RESET)"
	@# 1. Definir rutas
	$(eval OUT_DIR := sam-compile/$(FOLDER))
	@mkdir -p $(OUT_DIR)

	@# 2. Compilar el binario específico
	@echo "$(INFO)🔨 Compilando binario (Go nativo)...$(RESET)"
	@cd cmd/$(FOLDER) && \
		GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -mod=mod -tags lambda.norpc -o bootstrap main.go

	@# 3. Mover el binario
	@mv cmd/$(FOLDER)/bootstrap $(OUT_DIR)/bootstrap

	@# 3.1 Copiar recursos
	@mkdir -p $(OUT_DIR)/internal/appconfig
	@cp internal/appconfig/config.yml $(OUT_DIR)/internal/appconfig/
	@mkdir -p $(OUT_DIR)/internal/services/email/templates
	@cp -r internal/services/email/templates/* $(OUT_DIR)/internal/services/email/templates/ 2>/dev/null || :

	@# 4. Generar Makefile para SAM (Compatibilidad)
	@$(eval FUNC_PASCAL := $(shell echo "$(FOLDER)" | awk -F '-' '{for(i=1;i<=NF;i++) printf toupper(substr($$i,1,1)) substr($$i,2)}'))
	@$(eval LOGICAL_ID := $(PROJECT_NAME_PASCAL)$(FUNC_PASCAL))
	@printf "build-$(LOGICAL_ID):\n\tcp -r * \$$(ARTIFACTS_DIR)/\n\tchmod +x \$$(ARTIFACTS_DIR)/bootstrap\n" > $(OUT_DIR)/Makefile

	@# 5. Generar ZIP
	@echo "$(INFO)🤐 Creando archivo ZIP...$(RESET)"
	@cd $(OUT_DIR) && \
		chmod +x bootstrap && \
		zip -q -r ../$(FOLDER).zip .
	@echo "$(SUCCESS)✅ ZIP generado: sam-compile/$(FOLDER).zip$(RESET)"

.PHONY: update-fn
update-fn: package-lambda ## 🔄 Actualización rápida (alias de package-lambda).

.PHONY: package-all
package-all: ## 📦 Empaqueta TODAS las funciones Lambda.
	@echo "$(INFO)📦 Iniciando empaquetado masivo...$(RESET)"
	@for folder in $(FOLDERS); do \
		$(MAKE) package-lambda FOLDER=$$folder; \
	done
	@echo "$(SUCCESS)✅ Todas las funciones empaquetadas.$(RESET)"

.PHONY: ci-test
ci-test: ## 🧪 Ejecuta tests y linter para CI/CD.
	@echo "$(INFO)🧪 Ejecutando tests unitarios...$(RESET)"
	@REDIS_HOST=localhost go test -mod=mod ./... -v
	@echo "$(SUCCESS)✅ Tests completados.$(RESET)"


.PHONY: ci-build-lambda
ci-build-lambda: package-all ## 🏗️📦 CI: Construye artefactos para Lambda (ZIPs).
	@echo "$(SUCCESS)✅ Artefactos Lambda listos para despliegue.$(RESET)"

.PHONY: ci-build-eks
ci-build-eks: build-all-images ## 🏗️🐳 CI: Construye imágenes Docker para EKS.
	@echo "$(SUCCESS)✅ Imágenes EKS listas para despliegue.$(RESET)"

.PHONY: ci-build
ci-build: ci-build-lambda ci-build-eks ## 🏗️🌍 CI: Construye TODO (Lambda + EKS).
	@echo "$(SUCCESS)✅ Todos los artefactos construidos.$(RESET)"

.PHONY: fast-deploy
fast-deploy: ## ⚡🚀 Compilación nativa + actualización directa (Sin Terraform/Docker). Uso: make fast-deploy FOLDER=api
	@if [ -z "$(FOLDER)" ]; then echo "$(ERROR)❌ Debes especificar FOLDER (ej: api, sqs-consumer)$(RESET)"; exit 1; fi
	@$(MAKE) update-fn FOLDER=$(FOLDER)
	@echo "$(INFO)🔄 Actualizando código en LocalStack...$(RESET)"
	@# Mapeo de nombres de carpeta a nombres de función en LocalStack
	@FUNC_NAME="gofibercore-local-$(FOLDER)"; \
	if [ "$(FOLDER)" = "every-1min-cron" ]; then FUNC_NAME="gofibercore-local-1min-cron"; fi; \
	if [ "$(FOLDER)" = "daily-24-cron" ]; then FUNC_NAME="gofibercore-local-daily-cron"; fi; \
	echo "🎯 Función destino: $$FUNC_NAME"; \
	if awslocal lambda get-function --function-name $$FUNC_NAME >/dev/null 2>&1; then \
		awslocal lambda update-function-code --function-name $$FUNC_NAME --zip-file fileb://sam-compile/$(FOLDER).zip >/dev/null; \
		echo "$(SUCCESS)✅ Función actualizada exitosamente.$(RESET)"; \
	else \
		echo "$(WARNING)⚠️  Función $$FUNC_NAME no encontrada. Se omitirá actualización directa (Terraform la creará).$(RESET)"; \
	fi

.PHONY: fast-deploy-all
fast-deploy-all: ## ⚡🚀⚡ Actualiza TODAS las funciones rápidamente (Ideal si cambiaste internal/...). Uso: make fast-deploy-all
	@echo "$(INFO)🚀 Iniciando actualización masiva rápida...$(RESET)"
	@for folder in $(FOLDERS); do \
		echo "$(INFO)⏩ Procesando $$folder...$(RESET)"; \
		$(MAKE) fast-deploy FOLDER=$$folder; \
	done
	@echo "$(SUCCESS)🔥 Todo el stack (código) ha sido actualizado.$(RESET)"

.PHONY: compile-all
compile-all: ## 🏗️🏗️ Compila todas las funciones del proyecto. Uso: make compile-all
	@for folder in $(FOLDERS); do $(MAKE) compile-fn FOLDER=$$folder || exit 1; done
	@echo "$(SUCCESS)✅ Todas las funciones compiladas.$(RESET)"


.PHONY: sam-deploy
sam-deploy: ## 🚀 Despliega el stack SAM en LocalStack. Uso: make sam-deploy
	@echo "$(INFO)🚀 Desplegando stack con SAM...$(RESET)"
	@sam deploy --profile $(AWS_PROFILE_NAME) --template master-template.yml --stack-name $(STACK_NAME) --s3-bucket $(S3_BUCKET) --region $(AWS_DEFAULT_REGION) --no-confirm-changeset --capabilities CAPABILITY_IAM --disable-rollback --force-upload


.PHONY: deploy-staging
deploy-staging: ## 🚀☁️ Despliegue en STAGING (AWS Real). Uso: make deploy-staging
	@echo "$(WARNING)🚀 Iniciando despliegue en STAGING (AWS Real)...$(RESET)"
	@# 1. Generar variables usando .env.staging
	@$(MAKE) generate-tfvars MODE=lambda ENV_FILE=.env.staging ENVIRONMENT=staging
	@# 2. Compilar funciones
	@$(MAKE) compile-all
	@# 3. Desplegar con Terraform
	@echo "$(INFO)terraform apply...$(RESET)"
	@cd $(TF_DIR) && terraform init && terraform apply -var-file=$(TF_VARS)

.PHONY: deploy-prod
deploy-prod: ## 🚀🌍 Despliegue REAL en AWS Producción. Uso: make deploy-prod
	@echo "$(WARNING)🚀 Iniciando despliegue en PRODUCCIÓN (AWS Real)...$(RESET)"
	@# 1. Generar variables usando .env.prod
	@$(MAKE) generate-tfvars MODE=lambda ENV_FILE=.env.prod ENVIRONMENT=prod
	@# 2. Compilar funciones
	@$(MAKE) compile-all
	@# 3. Desplegar con Terraform
	@echo "$(INFO)terraform apply...$(RESET)"
	@cd $(TF_DIR) && terraform init && terraform apply -var-file=$(TF_VARS)


.PHONY: watch
watch: check-env ## 🏎️ Inicia API con live-reload (Air). Uso: make watch
	@$(MAKE) set-env ENV=local
	@$(MAKE) update-bruno-url-base ENV=local
	@echo "$(SUCCESS)🏎️ Iniciando modo watch...$(RESET)"
	$(DC_BASE) -p $(PROJECT_SLUG)-local up --build --remove-orphans --force-recreate


.PHONY: aws-down
aws-down: ## 💥🧹 Elimina contenedores e imágenes Docker (requiere confirmación explícita). Uso: make aws-down
	@sh -c ' \
		project_lower=$(PROJECT_NAME_LOWERCASE); \
		CONTAINERS=$$(docker ps -a --format "{{.Names}}" | grep "$$project_lower" || true); \
		IMAGES=$$(docker images --format "{{.Repository}}:{{.Tag}}" | grep "$$project_lower" || true); \
		if [ -z "$$CONTAINERS" ] && [ -z "$$IMAGES" ]; then \
			printf "$(INFO)🚫 No se encontraron contenedores ni imágenes que coincidan con el filtro: $$project_lower$(RESET)\n"; \
			exit 0; \
		fi; \
		printf "$(ERROR)🚨 ATENCIÓN: Eliminación masiva de contenedores e imágenes Docker iniciada para: $$project_lower$(RESET)\n"; \
		printf "$(INFO)📦 Contenedores a eliminar:\n$$CONTAINERS$(RESET)\n"; \
		printf "$(INFO)📦 Imágenes a eliminar:\n$$IMAGES$(RESET)\n"; \
		printf "$(WARNING)❓ ¿Estás ABSOLUTAMENTE seguro de que querés ELIMINAR estos recursos? (y/N)$(RESET) "; \
		read confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			printf "$(INFO)❌ Operación CANCELADA por el usuario.$(RESET)\n"; \
			exit 0; \
		fi; \
		printf "$(SUCCESS)✅ Confirmación recibida. Iniciando limpieza...$(RESET)\n"; \
		for container in $$CONTAINERS; do \
			printf "$(INFO)🧹 Eliminando contenedor: $$container...$(RESET)\n"; \
			image_id=$$(docker inspect --format="{{.Image}}" $$container 2>/dev/null); \
			docker rm -f $$container >/dev/null 2>&1 || true; \
			if [ -n "$$image_id" ]; then \
				printf "$(INFO)🗑️  Eliminando imagen asociada ($$image_id)...$(RESET)\n"; \
				docker rmi -f $$image_id >/dev/null 2>&1 || true; \
			fi; \
		done; \
		for image in $$IMAGES; do \
			printf "$(INFO)🗑️  Eliminando imagen: $$image...$(RESET)\n"; \
			docker rmi -f $$image >/dev/null 2>&1 || true; \
		done; \
		printf "$(SUCCESS)✅ Limpieza finalizada con éxito.$(RESET)\n"; \
	'


###############################################################################
## Cobra CLI
###############################################################################

.PHONY: run-cli
run-cli: ## ▶️ Ejecuta un comando CLI personalizado. Uso: make run-cli c="comando --flag=valor"
	@echo "▶️ Ejecutando comando CLI: $(c)..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go $(c)

.PHONY: cli-help
cli-help: ## 🧾 Lista los comandos CLI disponibles y su uso. Uso: make cli-help
	@echo "🧾 Comandos CLI disponibles:"
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go --help

.PHONY: create-command
create-command: ## ✨ Crea un nuevo comando Cobra. Uso: make create-command name=nuevoComando
	@if [ -z "$(name)" ]; then \
        echo "❌ Por favor, especifique el nombre del comando."; \
        exit 1; \
    fi
	@echo "✨ Creando comando Cobra: $(name)..."
	@$(DC_RUN) sh -c '\
        set -e; \
        echo "--> Ejecutando cobra-cli..."; \
        cobra-cli add $(name) -p "rootCmd" || true; \
        echo "--> Moviendo archivo..."; \
        mv "./cmd/$(name).go" "./cmd/cmd-cli/cmd/"; \
        echo "✅ ¡Comando creado"; \
    '

.PHONY: create-bulk-job-config
create-bulk-job-config: ## 🧾 Crea un bulk_job_config con ref_code consecutivo de 5 en 5. Uso: make create-bulk-job-config process_type_id=13 [sede_id=0 override_process_version_id=0 roadmap=0]
	@if [ -z "$(process_type_id)" ]; then \
		echo "❌ Debes especificar process_type_id: make create-bulk-job-config process_type_id=13"; \
		exit 1; \
	fi
	@echo "$(INFO)🧾 Creando bulk_job_config con ref_code consecutivo...$(RESET)"
	@$(DC_RUN) go run ./cmd/tools/create-bulk-job-config \
		-config "$(or $(config),internal/appconfig/config.yml)" \
		-process-type-id "$(process_type_id)" \
		-sede-id "$(or $(sede_id),0)" \
		-override-process-version-id "$(or $(override_process_version_id),0)" \
		-roadmap "$(or $(roadmap),0)"

.PHONY: cancel-process-run
cancel-process-run: ## 🛑 Cancela una corrida batch por run_key o bulk_job_id. Uso: make cancel-process-run bulk_job_id=2 [reason=manual_cancel]
	@if [ -z "$(run_key)" ] && [ -z "$(bulk_job_id)" ]; then \
		echo "❌ Debes especificar run_key o bulk_job_id: make cancel-process-run bulk_job_id=2"; \
		exit 1; \
	fi
	@echo "$(INFO)🛑 Cancelando corrida batch...$(RESET)"
	@$(DC_RUN) go run ./cmd/tools/cancel-process-run \
		-config "$(or $(config),internal/appconfig/config.yml)" \
		-run-key "$(run_key)" \
		-bulk-job-id "$(or $(bulk_job_id),0)" \
		-reason "$(or $(reason),manual_cancel)"

## --------------------------------------------------------------------------
## Gestión de Base de Datos
## --------------------------------------------------------------------------

.PHONY: create-migration
create-migration: ## 🧱 Crea un nuevo archivo de migración SQL. Uso: make create-migration name=create_users_table
	@if [ -z "$(name)" ]; then \
		echo "❌ Por favor, especifique el nombre. Uso: make create-migration name=create_users_table"; \
		exit 1; \
	fi
	@echo "🌱 Creando migración: $(name)..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations create $(name)

.PHONY: migrate-up
migrate-up: ## 🚀 Aplica todas las migraciones pendientes. Uso: make migrate-up
	@echo "🚀 Aplicando migraciones..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations up

.PHONY: migrate-down
migrate-down: ## ⏪ Revierte la última migración aplicada. Uso: make migrate-down
	@echo "⏪ Revertiendo la última migración..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations down

.PHONY: migrate-status
migrate-status: ## ℹ️ Muestra el estado de todas las migraciones. Uso: make migrate-status
	@echo "ℹ️  Estado de las migraciones:"
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations status

.PHONY: migrate-reset
migrate-reset: ## ♻️ Revierte y reaplica todas las migraciones (reset completo). Uso: make migrate-reset
	@echo "ℹ️  Reviendo todas las migraciones..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations reset

.PHONY: seed
seed: ## 🌱 Ejecuta todos los seeders registrados. Uso: make seed
	@echo "Ejecutando todos los seeders..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go seed

.PHONY: seed-one
seed-one: ## 🎯 Ejecuta un seeder específico. Uso: make seed-one name=catalog_items
	@if [ -z "$(name)" ]; then \
		echo "Debes pasar name=nombre_seeder, por ejemplo name=catalog_items"; \
		exit 1; \
	fi
	@echo "Ejecutando seeder: $(name)"
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go seed --only "$(name)"

.PHONY: seed-list
seed-list: ## 📋 Muestra la lista de seeders disponibles y cómo ejecutarlos. Uso: make seed-list
	@echo "📋 Seeders disponibles:"
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go seed --list

.PHONY: terraform-deploy-eks
terraform-deploy-eks: ## 🚀☸️ Despliega la infraestructura en modo EKS (usando Helm + Terraform). Uso: make terraform-deploy-eks
	@echo "$(INFO)🚀 Desplegando infraestructura en modo EKS (Híbrido)...$(RESET)"
	@cd $(TF_DIR) && \
		tflocal init && \
		tflocal apply -var-file=$(TF_VARS) -var="deploy_mode=eks" -auto-approve
	@echo "$(SUCCESS)✅ Despliegue en modo EKS completado.$(RESET)"

.PHONY: terraform-deploy-lambda
terraform-deploy-lambda: ## 🚀⚡ Despliega la infraestructura en modo Lambda (Clásico). Uso: make terraform-deploy-lambda
	@echo "$(INFO)🚀 Desplegando infraestructura en modo Lambda...$(RESET)"
	@cd $(TF_DIR) && \
		tflocal init && \
		tflocal apply -var-file=$(TF_VARS) -var="deploy_mode=lambda" -auto-approve
	@echo "$(SUCCESS)✅ Despliegue en modo Lambda completado.$(RESET)"

# ==============================================================================
# NIVEL 1: Desarrollo Local (Air + Docker Compose)
# ==============================================================================
.PHONY: infra-up
infra-up: ## 🐳 Verifica si LocalStack está corriendo (Infraestructura Compartida).
	@echo "$(INFO)🔍 Verificando LocalStack en $(LOCALSTACK_ENDPOINT_BASE)...$(RESET)"
	@curl -s -f "$(LOCALSTACK_ENDPOINT_BASE)/_localstack/health" >/dev/null 2>&1 || ( \
		echo "$(WARNING)⚠️ LocalStack NO respondió en $(LOCALSTACK_ENDPOINT_BASE). Intentando levantarlo...$(RESET)"; \
		$(MAKE) localstack-up; \
	)
	@curl -s -f "$(LOCALSTACK_ENDPOINT_BASE)/_localstack/health" >/dev/null 2>&1 || ( \
		echo "$(ERROR)❌ LocalStack NO respondió en $(LOCALSTACK_ENDPOINT_BASE).$(RESET)"; \
		exit 1; \
	)
	@echo "$(SUCCESS)✅ LocalStack OK (infraestructura compartida).$(RESET)"
	@./tools/init-localstack.sh
	@echo "$(WARNING)⚠️  Asegúrate de tener Postgres y Redis corriendo (host: 5432/6379).$(RESET)"

.PHONY: infra-down
infra-down: ## 🛑 Detiene dependencias
	docker compose -p localstack -f $(LOCALSTACK_DOCKER_FILE) down

.PHONY: dev-local
dev-local: ## ⚡ Nivel 1: Corre la API localmente con Air (Hot Reload)
	@echo "$(INFO)🚀 Iniciando API con Air...$(RESET)"
	air -c air.api.toml

# ==============================================================================
# NIVEL 2: Desarrollo Local (Kubernetes - Skaffold)
# ==============================================================================
.PHONY: dev-k8s
dev-k8s: ## ☸️  Nivel 2: Desarrollo en Kubernetes Local (Skaffold). Simula EKS.
	@echo "$(INFO)🚀 Iniciando entorno Kubernetes (Skaffold)...$(RESET)"
	skaffold dev --port-forward

# ==============================================================================
# NIVEL 3: Desarrollo GitOps (ArgoCD)
# ==============================================================================
.PHONY: argocd-pass
argocd-pass: ## 🔐 Muestra la contraseña de admin de ArgoCD
	@echo "$(INFO)🔐 Contraseña de ArgoCD (admin):$(RESET)"
	@kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d && echo ""

.PHONY: argocd-ui
argocd-ui: ## 🌐 Abre acceso a la UI de ArgoCD (localhost:8080)
	@echo "$(INFO)🔌 Abriendo túnel a ArgoCD... Accede en https://localhost:8080 (user: admin)$(RESET)"
	kubectl port-forward svc/argocd-server -n argocd 8080:443

.PHONY: check-k8s
check-k8s: ## 🕵️ Verifica la conexión al cluster de Kubernetes.
	@echo "$(INFO)🔍 Verificando conexión a Kubernetes (Contexto: $$(kubectl config current-context))...$(RESET)"
	@kubectl cluster-info > /dev/null 2>&1 || (echo "$(ERROR)❌ No se puede conectar al cluster de Kubernetes. Ejecuta 'make k8s-up' para iniciarlo.$(RESET)"; exit 1)
	@echo "$(SUCCESS)✅ Conexión a Kubernetes exitosa.$(RESET)"

.PHONY: k8s-up
k8s-up: ## ☸️🚀 Levanta el cluster de Kubernetes (Soporta OrbStack, Minikube, Docker Desktop).
	@echo "$(INFO)🚀 Intentando levantar Kubernetes...$(RESET)"
	@CONTEXT=$$(kubectl config current-context); \
	if [ "$$CONTEXT" = "orbstack" ]; then \
		echo "$(INFO)🛠️ Detectado entorno OrbStack. Verificando configuración...$(RESET)"; \
		if orbctl config show | grep -q "k8s.enable: false"; then \
			echo "$(INFO)☸️ Kubernetes está deshabilitado en OrbStack. Habilitando...$(RESET)"; \
			orbctl config set k8s.enable true; \
			echo "$(INFO)🔄 Reiniciando OrbStack para aplicar cambios...$(RESET)"; \
			orb stop && orb start; \
		else \
			echo "$(INFO)✅ Kubernetes ya está habilitado en OrbStack. Iniciando...$(RESET)"; \
			orb start; \
		fi; \
	elif [ "$$CONTEXT" = "minikube" ]; then \
		echo "$(INFO)🛠️ Detectado entorno Minikube. Iniciando...$(RESET)"; \
		minikube start; \
	elif [ "$$CONTEXT" = "kind-kind" ]; then \
		echo "$(INFO)🛠️ Detectado entorno Kind. Iniciando...$(RESET)"; \
		kind create cluster 2>/dev/null || echo "Cluster ya existe"; \
	elif [ "$$CONTEXT" = "docker-desktop" ]; then \
		echo "$(INFO)🛠️ Detectado entorno Docker Desktop. Iniciando...$(RESET)"; \
		open -a Docker; \
	else \
		echo "$(WARNING)⚠️ Contexto desconocido: $$CONTEXT. Intentando comando genérico...$(RESET)"; \
		echo "$(INFO)🛠️ Ejecutando 'orb start' por defecto...$(RESET)"; \
		orb start || echo "$(ERROR)❌ No se pudo iniciar OrbStack.$(RESET)"; \
	fi
	@echo "$(INFO)⏳ Esperando a que Kubernetes esté listo (máx 120s)...$(RESET)"
	@count=0; \
	while ! kubectl cluster-info >/dev/null 2>&1; do \
		if [ $$count -ge 60 ]; then \
			echo ""; \
			echo "$(ERROR)❌ Kubernetes no respondió después de 120 segundos. Verifica que Kubernetes esté habilitado en OrbStack/Docker.$(RESET)"; \
			exit 1; \
		fi; \
		printf "."; \
		sleep 2; \
		count=$$((count+1)); \
	done; \
	echo ""
	@echo "$(SUCCESS)✅ Kubernetes está listo.$(RESET)"

.PHONY: check-k8s-schedulable
check-k8s-schedulable: ## 🩺 Verifica si el nodo actual puede agendar pods antes de desplegar.
	@echo "$(INFO)🩺 Verificando salud del nodo Kubernetes...$(RESET)"
	@NODE_NAME=$$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}' 2>/dev/null); \
	if [ -z "$$NODE_NAME" ]; then \
		echo "$(ERROR)❌ No se pudo detectar un nodo de Kubernetes.$(RESET)"; \
		exit 1; \
	fi; \
	DISK_PRESSURE=$$(kubectl get node $$NODE_NAME -o jsonpath='{range .status.conditions[?(@.type=="DiskPressure")]}{.status}{end}'); \
	TAINTS=$$(kubectl get node $$NODE_NAME -o jsonpath='{range .spec.taints[*]}{.key}:{.effect}{"\n"}{end}' 2>/dev/null); \
	if [ "$$DISK_PRESSURE" = "True" ] || echo "$$TAINTS" | grep -q 'node.kubernetes.io/disk-pressure:NoSchedule'; then \
		if [ "$(AUTO_FIX_K8S_DISK_PRESSURE)" = "1" ]; then \
			echo "$(WARNING)⚠️ Detectado DiskPressure en '$$NODE_NAME'. Intentando limpieza automática segura...$(RESET)"; \
			$(MAKE) recover-k8s-disk-pressure AUTO_FIX_K8S_PRUNE_VOLUMES=$(AUTO_FIX_K8S_PRUNE_VOLUMES); \
			echo "$(INFO)⏳ Esperando a que Kubernetes actualice el estado del nodo (máx $(AUTO_FIX_K8S_DISK_PRESSURE_WAIT_SECONDS)s)...$(RESET)"; \
			remaining=$(AUTO_FIX_K8S_DISK_PRESSURE_WAIT_SECONDS); \
			while [ $$remaining -gt 0 ]; do \
				DISK_PRESSURE=$$(kubectl get node $$NODE_NAME -o jsonpath='{range .status.conditions[?(@.type=="DiskPressure")]}{.status}{end}'); \
				TAINTS=$$(kubectl get node $$NODE_NAME -o jsonpath='{range .spec.taints[*]}{.key}:{.effect}{"\n"}{end}' 2>/dev/null); \
				if [ "$$DISK_PRESSURE" != "True" ] && ! echo "$$TAINTS" | grep -q 'node.kubernetes.io/disk-pressure:NoSchedule'; then \
					break; \
				fi; \
				printf "."; \
				sleep 5; \
				remaining=$$((remaining-5)); \
			done; \
			echo ""; \
		fi; \
		DISK_PRESSURE=$$(kubectl get node $$NODE_NAME -o jsonpath='{range .status.conditions[?(@.type=="DiskPressure")]}{.status}{end}'); \
		TAINTS=$$(kubectl get node $$NODE_NAME -o jsonpath='{range .spec.taints[*]}{.key}:{.effect}{"\n"}{end}' 2>/dev/null); \
		if [ "$$DISK_PRESSURE" = "True" ] || echo "$$TAINTS" | grep -q 'node.kubernetes.io/disk-pressure:NoSchedule'; then \
			echo "$(ERROR)❌ El nodo '$$NODE_NAME' sigue con DiskPressure y no puede agendar pods.$(RESET)"; \
			echo "$(WARNING)💡 Recomendado: revisa espacio en OrbStack/Docker y luego reintenta.$(RESET)"; \
			echo "   docker builder prune -af"; \
			echo "   docker image prune -af"; \
			echo "   docker container prune -f"; \
			echo "   docker volume prune -f"; \
			echo "   kubectl describe node $$NODE_NAME | grep -A5 Conditions"; \
			exit 1; \
		fi; \
	fi; \
	echo "$(SUCCESS)✅ Nodo '$$NODE_NAME' sin DiskPressure. Se pueden agendar pods.$(RESET)"

.PHONY: recover-k8s-disk-pressure
recover-k8s-disk-pressure: ## 🧹 Ejecuta limpieza local para intentar liberar DiskPressure en Kubernetes.
	@echo "$(INFO)🧹 Liberando build cache, imágenes y contenedores detenidos...$(RESET)"
	@docker builder prune -af || true
	@docker image prune -af || true
	@docker container prune -f || true
	@if [ "$(AUTO_FIX_K8S_PRUNE_VOLUMES)" = "1" ]; then \
		echo "$(WARNING)🗑️ AUTO_FIX_K8S_PRUNE_VOLUMES=1 → limpiando volúmenes Docker...$(RESET)"; \
		docker volume prune -f || true; \
	else \
		echo "$(INFO)ℹ️ Omitiendo docker volume prune (usa AUTO_FIX_K8S_PRUNE_VOLUMES=1 si lo necesitas).$(RESET)"; \
	fi
	@echo "$(INFO)📊 Estado de disco Docker después de la limpieza:$(RESET)"
	@docker system df || true

.PHONY: k8s-down
k8s-down: ## 🛑 Detiene el cluster de Kubernetes.
	@echo "$(INFO)🛑 Intentando detener Kubernetes...$(RESET)"
	@CONTEXT=$$(kubectl config current-context); \
	if [ "$$CONTEXT" = "orbstack" ]; then \
		echo "$(INFO)🛠️ Detectado entorno OrbStack. Deteniendo...$(RESET)"; \
		orb stop; \
	elif [ "$$CONTEXT" = "minikube" ]; then \
		echo "$(INFO)🛠️ Detectado entorno Minikube. Deteniendo...$(RESET)"; \
		minikube stop; \
	elif [ "$$CONTEXT" = "kind-kind" ]; then \
		echo "$(INFO)🛠️ Detectado entorno Kind. Eliminando cluster...$(RESET)"; \
		kind delete cluster; \
	elif [ "$$CONTEXT" = "docker-desktop" ]; then \
		echo "$(INFO)🛠️ Detectado entorno Docker Desktop. Deteniendo...$(RESET)"; \
		osascript -e 'quit app "Docker"'; \
	else \
		echo "$(WARNING)⚠️ Contexto desconocido: $$CONTEXT. Intentando comando genérico...$(RESET)"; \
		echo "$(INFO)🛠️ Ejecutando 'orb stop' por defecto...$(RESET)"; \
		orb stop || echo "$(ERROR)❌ No se pudo detener OrbStack.$(RESET)"; \
	fi
	@echo "$(SUCCESS)✅ Kubernetes detenido.$(RESET)"

.PHONY: watch-eks
watch-eks: check-env infra-up k8s-up check-k8s-schedulable ## ☸️ Levanta todo el entorno en EKS (LocalStack) compilando imágenes.
	@$(MAKE) generate-tfvars MODE=eks
	@echo "$(INFO)🚀 Iniciando despliegue en EKS...$(RESET)"
	@$(MAKE) build-all-images
	@$(MAKE) check-k8s-schedulable
	@$(MAKE) clean-k8s-apps
	@$(MAKE) infra-init
	@echo "$(INFO)terraform apply -var 'deploy_mode=eks'...$(RESET)"
	@cd $(TF_DIR) && tflocal apply -var-file=$(TF_VARS) -var "deploy_mode=eks" -auto-approve
	@echo "$(SUCCESS)✅ Despliegue en EKS completado.$(RESET)"
	@echo "$(INFO)🔍 Servicios expuestos:$(RESET)"
	@kubectl get svc
	@$(MAKE) update-bruno-eks

.PHONY: clean-k8s-apps
clean-k8s-apps: ## 🧹 Elimina releases de Helm previos para evitar conflictos de estado.
	@echo "$(INFO)🧹 Limpiando aplicaciones en Kubernetes (Helm)...$(RESET)"
	@helm uninstall api sqs-consumer dlq-consumer daily-cron 1min-cron 2>/dev/null || true
	@echo "$(INFO)🧹 Limpiando recursos legacy (no-Helm) que pueden chocar con LoadBalancer...$(RESET)"
	@kubectl delete svc api -n default --ignore-not-found >/dev/null 2>&1 || true
	@kubectl delete deploy api -n default --ignore-not-found >/dev/null 2>&1 || true
	@kubectl delete rs -n default -l app=api --ignore-not-found >/dev/null 2>&1 || true
	@kubectl delete pods -n default -l app=api --ignore-not-found >/dev/null 2>&1 || true
	@echo "$(SUCCESS)✅ Limpieza completada (o no había nada que limpiar).$(RESET)"

.PHONY: update-bruno-eks
update-bruno-eks: ## 📝 Actualiza la IP del entorno EKS en Bruno.
	@echo "$(INFO)🔄 Actualizando IP en Bruno (bruno/environments/eks.bru)...$(RESET)"
	@IP=$$(kubectl get svc api-gofiber-app -o jsonpath='{.status.loadBalancer.ingress[0].ip}'); \
	if [ -z "$$IP" ]; then \
		echo "$(WARNING)⚠️ No se pudo obtener la IP externa. Verifica que el servicio 'api-gofiber-app' tenga una IP asignada.$(RESET)"; \
	else \
		echo "IP detectada: $$IP"; \
		sed -i '' "s|urlBase: .*|urlBase: http://$$IP/|g" bruno/environments/eks.bru; \
		echo "$(SUCCESS)✅ IP actualizada en Bruno: http://$$IP/$(RESET)"; \
	fi

.PHONY: build-image
build-image: ## 🐳 Construye una imagen Docker (Universal). Uso: make build-image FOLDER=api
	@echo "$(INFO)🐳 Construyendo imagen para $(FOLDER)...$(RESET)"
	docker build -t $(FOLDER):local \
		--build-arg CMD_PATH=cmd/$(FOLDER) \
		--build-arg BIN_NAME=bootstrap \
		-f dockerfiles/Dockerfile.universal .

.PHONY: build-all-images
build-all-images: ##  Construye todas las imágenes Docker para EKS.
	@echo "$(INFO)🔨 Construyendo imágenes Docker...$(RESET)"
	@docker build -f dockerfiles/Dockerfile.universal --build-arg CMD_PATH=cmd/api -t api:local .
	@docker build -f dockerfiles/Dockerfile.universal --build-arg CMD_PATH=cmd/sqs-consumer -t sqs-consumer:local .
	@docker build -f dockerfiles/Dockerfile.universal --build-arg CMD_PATH=cmd/dlq-consumer -t dlq-consumer:local .
	@docker build -f dockerfiles/Dockerfile.universal --build-arg CMD_PATH=cmd/every-1min-cron -t every-1min-cron:local .
	@docker build -f dockerfiles/Dockerfile.universal --build-arg CMD_PATH=cmd/daily-24-cron -t daily-24-cron:local .
	@echo "$(SUCCESS)✅ Imágenes construidas.$(RESET)"
