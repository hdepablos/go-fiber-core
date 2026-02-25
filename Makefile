# Carga las variables desde el archivo .env y las exporta.
include .env
export

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
S3_BUCKET_NAME=${PROJECT_NAME_LOWERCASE}-app-data
S3_BUCKET=${PROJECT_NAME_LOWERCASE}-bucket
SQS_QUEUE_NAME=${PROJECT_NAME_LOWERCASE}queue
SQS_DLQ_NAME=${PROJECT_NAME_LOWERCASE}dlq
SQS_QUEUE_URL=${LOCALSTACK_ENDPOINT_BASE}/000000000000/${SQS_QUEUE_NAME}
SQS_DLQ_URL=${LOCALSTACK_ENDPOINT_BASE}/000000000000/${SQS_DLQ_NAME}
FUNCTION_NAME_SQS_CONSUMER=${PROJECT_NAME_PASCAL}SqsConsumer
# Variables de Terraform
TF_DIR := ./terraform
TF_VARS := local.tfvars

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
DC_RUN  = $(DC_BASE) run --rm $(SERVICE_NAME)

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
	@$(MAKE) install-pkg pkg=github.com/golang-jwt/jwt/v5
	@$(MAKE) install-pkg pkg=golang.org/x/crypto/bcrypt
	@$(MAKE) install-pkg pkg=github.com/redis/go-redis/v9
	@$(MAKE) install-pkg pkg=gorm.io/gorm
	@$(MAKE) install-pkg pkg=gorm.io/driver/postgres
	@$(MAKE) install-pkg pkg=github.com/jackc/pgx/v5
	@$(MAKE) install-pkg pkg=github.com/spf13/viper
	@$(MAKE) install-pkg pkg=github.com/gofiber/fiber/v2
	@$(MAKE) install-pkg pkg=github.com/gofiber/fiber/v2/middleware/limiter
	@$(MAKE) install-pkg pkg=github.com/gofiber/fiber/v2/middleware/cors
	@$(MAKE) install-pkg pkg=github.com/spf13/cobra
	@$(MAKE) install-pkg pkg=github.com/robfig/cron/v3
	@$(MAKE) install-pkg pkg=gopkg.in/gomail.v2
	@$(MAKE) install-pkg pkg=github.com/natefinch/lumberjack
	@$(MAKE) install-pkg pkg=github.com/russross/blackfriday/v2
	@$(MAKE) install-pkg pkg=github.com/go-resty/resty/v2
	@$(MAKE) install-pkg pkg=github.com/mitchellh/mapstructure
	@$(MAKE) install-pkg pkg=github.com/go-playground/locales
	@$(MAKE) install-pkg pkg=github.com/go-playground/universal-translator
	@$(MAKE) install-pkg pkg=github.com/alicebob/miniredis/v2
	@$(MAKE) install-pkg pkg=github.com/DATA-DOG/go-sqlmock
	@$(MAKE) install-pkg pkg=github.com/stretchr/testify/mock
	@$(MAKE) install-pkg pkg=github.com/go-playground/locales/es
	@$(MAKE) install-pkg pkg=github.com/go-playground/validator/v10
	@$(MAKE) install-pkg pkg=github.com/go-playground/validator/v10/translations/es
	@$(MAKE) install-pkg pkg=github.com/aws/aws-sdk-go-v2/aws
	@$(MAKE) install-pkg pkg=github.com/aws/aws-sdk-go-v2/service/sns
	@$(MAKE) install-pkg pkg=github.com/aws/aws-sdk-go-v2/service/sqs
	@$(MAKE) install-pkg pkg=github.com/aws/aws-lambda-go/events
	@$(MAKE) install-pkg pkg=github.com/aws/aws-lambda-go/lambda
	@$(MAKE) install-pkg pkg=github.com/aws/aws-sdk-go-v2/config
	@$(MAKE) vendor

.PHONY: wire
wire: ## 🧬 Genera el código de inyección de dependencias con Google Wire. Uso: make wire
	@echo "$(SUCCESS)🧬 Generando inyección de dependencias con Wire...$(RESET)"
	@$(DC_RUN) wire gen -tags wireinject ./cmd/api/di

.PHONY: wire-sync
wire-sync: ## 🧬📦 Genera código de Wire y actualiza vendor. Uso: make wire-sync
	@$(MAKE) wire
	@$(MAKE) vendor
	@echo "$(SUCCESS)✅ Proceso de Wire y vendor completado.$(RESET)"

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

.PHONY: logs-groups
logs-groups: ## 📚 Lista los log groups del proyecto en CloudWatch. Uso: make logs-groups
	@PREFIX="/app/$(PROJECT_SLUG)"; \
	echo "$(INFO)📚 Listando log groups con prefijo $$PREFIX$(RESET)"; \
	aws logs describe-log-groups $(AWS_ENDPOINT_ARG) $(AWS_PROFILE_ARG) --log-group-name-prefix "$$PREFIX" --query 'logGroups[].logGroupName' --output table

.PHONY: send-message
send-message: ## ✉️ Envía un mensaje de prueba a la cola SQS. Uso: make send-message
	@echo "$(INFO)✉️ Enviando mensaje a la cola '$(SQS_QUEUE_NAME)'...$(RESET)"
	awslocal sqs send-message \
		--queue-url $(SQS_QUEUE_URL) \
		--message-body '{"action":"process_data","id":"123","value":"test_payload"}'
	@echo "$(SUCCESS)✅ Mensaje enviado correctamente.$(RESET)"

.PHONY: send-message-error
send-message-error: ## ✉️⚠️ Envía un mensaje de error a la cola SQS. Uso: make send-message-error
	@echo "$(INFO)✉️ Enviando mensaje de error a la cola '$(SQS_QUEUE_NAME)'...$(RESET)"
	awslocal sqs send-message \
		--queue-url $(SQS_QUEUE_URL) \
		--message-body '{"action":"process_data","id":"999","value":"test_payload"}'
	@echo "$(SUCCESS)✅ Mensaje de error enviado.$(RESET)"

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
	@docker compose -f docker-compose-local-lint.yml build --no-cache lint && docker compose -f docker-compose-local-lint.yml run --rm lint run --timeout=2m

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
	@docker compose -f docker-compose-local-lint.yml run --rm lint run -v --timeout=2m

.PHONY: lint-test
lint-test: ## 🧪 Prueba si wire_gen.go está siendo ignorado. Uso: make lint-test
	@echo "🧪 Listando archivos que el linter va a analizar..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint run --issues-exit-code=0 2>&1 | grep -i "wire_gen" || echo "✅ wire_gen.go NO aparece en la salida (está siendo ignorado)"

.PHONY: localstack-up
localstack-up: ## 🛠️ Levanta LocalStack en segundo plano. Uso: make localstack-up
	@echo "$(SUCCESS)🛠️ Iniciando LocalStack...$(RESET)"
	@docker-compose -p localstack -f docker-composes/docker-compose.localstack.yml up -d --build --force-recreate
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
watch-lambda: fast-deploy-all infra-deploy update-bruno ## 🚀👀 Actualiza código, infraestructura y bruno (sin bloquear). Uso: make watch-lambda
	@echo "$(INFO)📺 Ejecuta 'make logs-all' si quieres Observar los logs de las funciones$(RESET)"

.PHONY: update-bruno
update-bruno: ## 🦁 Actualiza la URL de la API en Bruno. Uso: make update-bruno
	@echo "$(INFO)🔄 Actualizando URL en Bruno...$(RESET)"
	@API_ID=$$(awslocal apigateway get-rest-apis --query "items[0].id" --output text); \
	if [ -z "$$API_ID" ] || [ "$$API_ID" = "None" ]; then \
		echo "$(ERROR)❌ No se encontró API Gateway ID.$(RESET)"; \
	else \
		sed -i '' "s|urlBase: http://localhost:4566/restapis/[a-z0-9]*/Prod/_user_request_/|urlBase: http://localhost:4566/restapis/$$API_ID/Prod/_user_request_/|g" bruno/environments/lambda.bru; \
		echo "$(SUCCESS)✅ Bruno actualizado con API ID: $$API_ID$(RESET)"; \
	fi


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
infra-init: ## 🏁 Inicializa Terraform/LocalStack. Uso: make infra-init
	@cd $(TF_DIR) && tflocal init

.PHONY: infra-deploy
infra-deploy: ## 🚀 Despliega toda la infraestructura en LocalStack. Uso: make infra-deploy
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

.PHONY: logs-all
logs-all: ## 📊 Sigue logs de TODAS las lambdas en vivo (Ctrl+C para salir). Uso: make logs-all
	@echo "📺 Observando logs de las funciones... (Ctrl+C para detener)"
	@sh -c '\
		trap "kill 0 2>/dev/null || true; exit 0" INT TERM; \
		awslocal logs tail /aws/lambda/gofibercore-local-api --follow & \
		awslocal logs tail /aws/lambda/gofibercore-local-sqs-consumer --follow & \
		awslocal logs tail /aws/lambda/gofibercore-local-1min-cron --follow & \
		awslocal logs tail /aws/lambda/gofibercore-local-daily-cron --follow & \
		wait \
	'


.PHONY: update-fn
update-fn: ## 🔄 Actualización rápida de código en LocalStack. Uso: make update-fn FOLDER=api
	@echo "$(INFO)🏗️ Compilando [$(FOLDER)] de forma nativa...$(RESET)"
	@# 1. Definir rutas
	$(eval OUT_DIR := sam-compile/$(FOLDER))
	@mkdir -p $(OUT_DIR)

	@# 2. Compilar el binario específico (IMPORTANTE: Entra a la subcarpeta cmd/FOLDER)
	@echo "$(INFO)🔨 Ejecutando go build para cmd/$(FOLDER)/main.go...$(RESET)"
	@cd cmd/$(FOLDER) && \
		GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap main.go

	@# 3. Mover el binario a la carpeta de salida
	@mv cmd/$(FOLDER)/bootstrap $(OUT_DIR)/bootstrap

	@# 3.1 Copiar archivo de configuración y otros recursos necesarios
	@echo "$(INFO)📂 Copiando recursos estáticos...$(RESET)"
	@mkdir -p $(OUT_DIR)/internal/appconfig
	@cp internal/appconfig/config.yml $(OUT_DIR)/internal/appconfig/
	@mkdir -p $(OUT_DIR)/internal/services/email/templates
	@cp -r internal/services/email/templates/* $(OUT_DIR)/internal/services/email/templates/ 2>/dev/null || :

	@# 4. Generar el Makefile para SAM (compatibilidad)
	@$(eval FUNC_PASCAL := $(shell echo "$(FOLDER)" | awk -F '-' '{for(i=1;i<=NF;i++) printf toupper(substr($$i,1,1)) substr($$i,2)}'))
	@$(eval LOGICAL_ID := $(PROJECT_NAME_PASCAL)$(FUNC_PASCAL))
	@printf "build-$(LOGICAL_ID):\n\tcp -r * \$$(ARTIFACTS_DIR)/\n\tchmod +x \$$(ARTIFACTS_DIR)/bootstrap\n" > $(OUT_DIR)/Makefile

	@# 5. 📦 Generar el ZIP
	@echo "$(INFO)📦 Empaquetando ZIP: sam-compile/$(FOLDER).zip$(RESET)"
	@cd $(OUT_DIR) && \
		chmod +x bootstrap && \
		zip -q -r ../$(FOLDER).zip .

	@echo "$(SUCCESS)🚀 ZIP listo y verificado.$(RESET)"

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
	@sam deploy --profile $(AWS_PROFILE_NAME) --template master-template.yml --stack-name $(STACK_NAME) --s3-bucket $(S3_BUCKET_NAME) --region $(AWS_DEFAULT_REGION) --no-confirm-changeset --capabilities CAPABILITY_IAM --disable-rollback --force-upload


.PHONY: deploy-prod
deploy-prod: ## 🚀🌍 Despliegue REAL en AWS Producción. Uso: make deploy-prod
	@echo "$(WARNING)🚀 Iniciando despliegue en PRODUCCIÓN...$(RESET)"
	@$(MAKE) compile-all
	@sam build -t master-template.yml
	@sam deploy --stack-name $(STACK_NAME) --s3-bucket $(S3_BUCKET) --region $(AWS_DEFAULT_REGION) --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND --no-confirm-changeset --parameter-overrides S3BucketName=$(S3_BUCKET_NAME) ProjectName=$(PROJECT_NAME_LOWERCASE)


.PHONY: watch
watch: ## 🏎️ Inicia API con live-reload (Air). Uso: make watch
	@$(MAKE) set-env ENV=local
	@$(MAKE) update-bruno-url-base ENV=local
	@echo "$(SUCCESS)🏎️ Iniciando modo watch...$(RESET)"
	$(DC_BASE) -p $(PROJECT_SLUG)-local up --remove-orphans --force-recreate




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
	fi; \
	echo "Ejecutando seeder: $(name)"; \
	$(DC_RUN) go run ./cmd/cmd-cli/main.go seed --only $(name)

.PHONY: seed-list
seed-list: ## 📋 Muestra la lista de seeders disponibles y cómo ejecutarlos. Uso: make seed-list
	@echo "📋 Seeders disponibles:"
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go seed --list
