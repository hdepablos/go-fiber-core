# Carga las variables desde el archivo .env y las exporta.
include .env
export

# .DEFAULT_GOAL define el comando que se ejecuta si solo escribís "make".
.DEFAULT_GOAL := help

###############################################################################
## Diferentes colores para mejorar la legibilidad en la terminal.
###############################################################################
RESET 		= \033[0m		# Restablece el color por defecto
INFO 		= \033[0;36m	# Cian para información general
SUCCESS 	= \033[0;32m	# Verde para operaciones exitosas
WARNING 	= \033[0;33m	# Amarillo para advertencias
ERROR 		= \033[0;31m	# Rojo para errores críticos
PROMPT 		= \033[0;35m	# Magenta para preguntas de usuario
HEADER 		= \033[1;34m	# Azul brillante para encabezados
HIGHLIGHT 	= \033[1;33m	# Amarillo brillante para destacar algo

###############################################################################
## Variables
###############################################################################
SERVICE_NAME := $(PROJECT_SLUG)
PROJECT_NAME_LOWERCASE := $(subst -, ,$(PROJECT_SLUG))
PROJECT_NAME_LOWERCASE := $(subst _, ,$(PROJECT_NAME_LOWERCASE))
PROJECT_NAME_LOWERCASE := $(strip $(PROJECT_NAME_LOWERCASE))
PROJECT_NAME_LOWERCASE := $(shell echo $(PROJECT_NAME_LOWERCASE) | tr -d ' ' | tr '[:upper:]' '[:lower:]')
PROJECT_NAME_PASCAL := $(shell echo $(PROJECT_SLUG) | awk -F '[-_]' '{for(i=1;i<=NF;i++){printf "%s", toupper(substr($$i,1,1)) tolower(substr($$i,2))}}')
STACK_NAME := $(PROJECT_NAME_LOWERCASE)-stack
FOLDERS := $(shell echo "$(FUNCTIONS)" | tr ',' ' ')
S3_BUCKET_NAME=${PROJECT_NAME_LOWERCASE}-bucket
STACK_NAME=${PROJECT_NAME_LOWERCASE}-stack

ifeq ($(APP_ENV),local)
	SAM_ENDPOINT_ARG=--endpoint-url $(LOCALSTACK_ENDPOINT)
	AWS_ENDPOINT_ARG=--endpoint-url $(LOCALSTACK_ENDPOINT)
	AWS_PROFILE_ARG=
else
	SAM_ENDPOINT_ARG=
	AWS_ENDPOINT_ARG=
	AWS_PROFILE_ARG=--profile $(AWS_PROFILE_NAME)
endif

DOCKER_FILE := docker-compose-$(APP_ENV).yml
DC_BASE = docker compose -f docker-compose-base.yml -f $(DOCKER_FILE)
DC_RUN  = $(DC_BASE) run --rm $(SERVICE_NAME)


###############################################################################
# Comandos disponibles
###############################################################################
help: ## ℹ️ Muestra todos los comandos disponibles con su descripción.
	@awk -F ':|##' '/^[a-zA-Z0-9_-]+:.*?##/ {printf "\033[36m%-20s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST)


show-all-variables: ## 🐳🚀 Muestra las variables principalales del proyecto
	@echo "PROJECT_SLUG: $(PROJECT_SLUG)"
	@echo "PROJECT_NAME_LOWERCASE: $(PROJECT_NAME_LOWERCASE)"
	@echo "PROJECT_NAME_PASCAL: $(PROJECT_NAME_PASCAL)"
	@echo "SERVICE_NAME: $(SERVICE_NAME)"
	@echo "DOCKER_FILE: $(DOCKER_FILE)"
	@echo "DC_BASE: $(DC_BASE)"
	@echo "DC_RUN: $(DC_RUN)"


###############################################################################
## Diferenetes colores para mejorar la legibilidad en la terminal.
###############################################################################
color-messages: ## 🐳🚀 Ejemplos de los diferentes colores de mensajes
	@echo "$(RESET)	RESET  🚀 Color del mensaje$(RESET)"
	@echo "$(INFO)		INFO  🚀 Color del mensaje$(RESET)"
	@echo "$(SUCCESS)		SUCCESS  🚀 Color del mensaje$(RESET)"
	@echo "$(WARNING)		WARNING  🚀 Color del mensaje$(RESET)"
	@echo "$(ERROR)		ERROR  🚀 Color del mensaje$(RESET)"
	@echo "$(PROMPT)		PROMPT  🚀 Color del mensaje$(RESET)"
	@echo "$(HEADER)		HEADER  🚀 Color del mensaje$(RESET)"
	@echo "$(HIGHLIGHT)		HIGHLIGHT  🚀 Color del mensaje$(RESET)"

###############################################################################
## Validación de variables de entorno necesarias para el Makefile
###############################################################################
check-env: ## 🚀 Verifica que existan las variables de entorno indispensables para el Makefile
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

###############################################################################
## Compilación y construcción de la aplicación
###############################################################################


###############################################################################
## Golang
###############################################################################
vendor: ## 📦 Actualiza el archivo go.mod y la carpeta vendor.
	@echo "$(SUCCESS)Ordenando y vendoring dependencias...$(RESET)"
	@$(DC_RUN) go mod tidy
	@$(DC_RUN) go mod vendor


install-pkg: ## 📥 Instala o actualiza un paquete Go específico. Uso: make install-pkg pkg=...
	@echo "$(SUCCESS)Instalando/actualizando paquete: $(pkg)...$(RESET)"
	@$(DC_RUN) go get -u $(pkg)
	@make vendor


install-all-pkg: # Install multiple Go dependencies
	@echo "Installing all dependencies..."
	make install-pkg pkg=github.com/golang-jwt/jwt/v5
	make install-pkg pkg=golang.org/x/crypto/bcrypt
	make install-pkg pkg=github.com/redis/go-redis/v9
	make install-pkg pkg=gorm.io/gorm
	make install-pkg pkg=gorm.io/driver/postgres
	make install-pkg pkg=github.com/jackc/pgx/v5
	make install-pkg pkg=github.com/spf13/viper
	make install-pkg pkg=github.com/gofiber/fiber/v2
	make install-pkg pkg=github.com/gofiber/fiber/v2/middleware/limiter
	make install-pkg pkg=github.com/gofiber/fiber/v2/middleware/cors
	make install-pkg pkg=github.com/spf13/cobra
	make install-pkg pkg=github.com/robfig/cron/v3
	make install-pkg pkg=gopkg.in/gomail.v2
	make install-pkg pkg=github.com/natefinch/lumberjack
	make install-pkg pkg=github.com/russross/blackfriday/v2
	make install-pkg pkg=github.com/go-resty/resty/v2
	make install-pkg pkg=github.com/mitchellh/mapstructure
	make install-pkg pkg=github.com/go-playground/locales
	make install-pkg pkg=github.com/go-playground/universal-translator
	make install-pkg pkg=github.com/alicebob/miniredis/v2
	make install-pkg pkg=github.com/DATA-DOG/go-sqlmock
	make install-pkg pkg=github.com/stretchr/testify/mock
	make install-pkg pkg=github.com/go-playground/locales/es
	make install-pkg pkg=github.com/go-playground/validator/v10
	make install-pkg pkg=github.com/go-playground/validator/v10/translations/es

	make install-pkg pkg=github.com/aws/aws-sdk-go-v2/aws
	make install-pkg pkg=github.com/aws/aws-sdk-go-v2/service/sns
	make install-pkg pkg=github.com/aws/aws-sdk-go-v2/service/sqs
	make install-pkg pkg=github.com/aws/aws-lambda-go/events
	make install-pkg pkg=github.com/aws/aws-lambda-go/lambda
	make install-pkg pkg=github.com/aws/aws-sdk-go-v2/config

	make vendor

wire: ## 🧬 Genera el código de inyección de dependencias con Google Wire.
	@echo "$(SUCCESS)Generando inyección de dependencias con Wire...$(RESET)"
	@$(DC_RUN) wire gen -tags wireinject ./cmd/api/di

wire-sync: wire vendor ## 🧬+📦 Genera código de Wire y actualiza go.mod/vendor después.
	@echo "$(SUCCESS)Proceso de Wire y vendor completado.$(RESET)"


###############################################################################
## Testing
###############################################################################


###############################################################################
## Gestión de Base de Datos
###############################################################################


###############################################################################
## AWS
###############################################################################
localstack-up: ## 🐳🚀 Levanta LocalStack en segundo plano y espera a que esté listo
	@echo "$(SUCCESS)🚀 Iniciando LocalStack...$(RESET)"
	@echo "$(INFO)⏳ Esperando a que LocalStack esté listo...$(RESET)"
	@docker-compose -p localstack -f docker-compose.localstack.yml up -d --build --force-recreate
	@sleep 10
	@echo "$(SUCCESS)✅ LocalStack se encuentra en funcionamiento correctamente$(RESET)"


render-template:
	@service_name=$$(echo "$(PROJECT_NAME_PASCAL)-$(folder)" | tr "-" " " | awk '{ for (i=1; i<=NF; i++) printf toupper(substr($$i,1,1)) substr($$i,2) }'); \
	if [ "$(folder)" = "api" ]; then \
		stub="stubs/api-lambda.stub"; \
	elif echo "$(folder)" | grep -q -- "-cron$$"; then \
		stub="stubs/cron-lambda.stub"; \
	else \
		stub="stubs/$(folder)-lambda.stub"; \
	fi; \
	mkdir -p templates; \
	sed \
		-e "s|__PROJECT__|$(PROJECT)|g" \
		-e "s|__SERVICE_NAME__|$$service_name|g" \
		-e "s|__PROJECT_LOWER__|$(PROJECT_NAME_LOWERCASE)|g" \
		-e "s|__FOLDER__|$(folder)|g" \
		$$stub > templates/$(folder)-template.yml; \
	echo "$(SUCCESS)✅ Template generado para $(folder)$(RESET)"


render-templates:
	@for folder in $(FOLDERS); do \
		$(MAKE) render-template folder=$$folder; \
	done
	@$(MAKE) render-template folder=sqs-queues


delete-templates:
	@rm -f templates/*.yml
	@echo "$(SUCCESS)✅ Se han eliminado los templates$(RESET)"


update-api-base:
	@echo "🔍 Obteniendo URL de la API Gateway desde LocalStack..."
	@API_URL=$$(aws --profile $(AWS_PROFILE_NAME) cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--endpoint-url=$(LOCALSTACK_ENDPOINT) \
		--query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" \
		--output text); \
	if [ -z "$$API_URL" ]; then \
		echo "❌ No se pudo obtener la URL del API Gateway."; \
		echo "🔍 Verificando si el stack existe y tiene el output 'ApiUrl'..."; \
		aws --profile $(AWS_PROFILE_NAME) cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--endpoint-url=$(LOCALSTACK_ENDPOINT) \
			--query "Stacks[0].Outputs[*].[OutputKey,OutputValue]" \
			--output table || echo "❌ El stack '$(STACK_NAME)' no existe o no tiene outputs."; \
		exit 1; \
	fi; \
	echo "✅ API_BASE: $$API_URL"; \
	echo "$$API_URL" > .api_base_tmp


# actualiza el archivo .env con la nueva URL_BASE
update-env-url-base: update-api-base
	@echo "✏️ Actualizando URL_BASE en archivo .env..."
	@API_BASE=$$(cat .api_base_tmp); \
	if [ "$$(uname)" = "Darwin" ]; then \
		sed -i '' -E "s|^URL_BASE=.*|URL_BASE=$$API_BASE|" .env; \
	else \
		sed -i -E "s|^URL_BASE=.*|URL_BASE=$$API_BASE|" .env; \
	fi && \
	echo "✅ URL_BASE actualizado a $$API_BASE en .env"

# actualiza el archivo bruno/environments/<APP_ENV>.bru con la nueva urlBase
update-bruno-url-base:
	@echo "✏️ Actualizando urlBase en archivo bruno para entorno '$(APP_ENV)'..."
	@API_BASE="$$(cat .api_base_tmp)"; \
	API_BASE="$${API_BASE%/}/"; \
	BRUNO_ENV_FILE=bruno/environments/$(APP_ENV).bru; \
	if [ ! -f "$$BRUNO_ENV_FILE" ]; then \
		echo "❌ Archivo $$BRUNO_ENV_FILE no encontrado."; \
		exit 1; \
	fi; \
	if [ "$$(uname)" = "Darwin" ]; then \
		sed -i '' -E "s|urlBase: .*|urlBase: $$API_BASE|" "$$BRUNO_ENV_FILE"; \
	else \
		sed -i -E "s|urlBase: .*|urlBase: $$API_BASE|" "$$BRUNO_ENV_FILE"; \
	fi && \
	echo "✅ urlBase actualizado en $$BRUNO_ENV_FILE"


# actualiza el archivo .env y bruno con la nueva urlBase
update-url-all: update-env-url-base update-bruno-url-base


# Compila y empaqueta una función Lambda específica y actualiza el template SAM
update-function:
	ifndef FOLDER
		$(error ❌ Debes indicar la función: make update-function FOLDER=Carpeta función)
	endif

	$(MAKE) compile-fn FOLDER=$(FOLDER);
	@sam build --template $(TEMPLATE_FILE)

###############################################################################
## Compile single function (derivado desde FUNCTIONS)
###############################################################################
compile-fn:
	@if [ -z "$(FOLDER)" ]; then \
		echo "❌ Debes indicar FOLDER=<funcion>"; \
		exit 1; \
	fi

	@echo "▶ Compilando función: $(FOLDER)"

	@out_dir="sam-compile/$(FOLDER)"; \
	rm -rf $$out_dir; \
	mkdir -p $$out_dir; \
	echo "▶ Construyendo imagen Docker"; \
	docker build \
		--build-arg FOLDER=$(FOLDER) \
		--build-arg FUNC_NAME=$(FOLDER) \
		-f Dockerfile.func.lambda \
		-t lambda-$(FOLDER):latest .; \
	echo "▶ Extrayendo artefactos"; \
	container_id=$$(docker create lambda-$(FOLDER):latest); \
	docker cp "$$container_id:/app/$(FOLDER)/." "$$out_dir/"; \
	docker rm "$$container_id" >/dev/null; \
	echo "▶ Generando Makefile SAM"; \
	project_pascal="$(PROJECT_NAME_PASCAL)"; \
	func_pascal=$$(echo "$(FOLDER)" | awk -F '-' '{ for(i=1;i<=NF;i++) printf toupper(substr($$i,1,1)) substr($$i,2) }'); \
	logical_id="$$project_pascal$$func_pascal"; \
	printf "%s\n" \
		"build-$$logical_id:" \
		"	mkdir -p \$$(ARTIFACTS_DIR)" \
		"	cp bootstrap \$$(ARTIFACTS_DIR)/bootstrap" \
		"	chmod +x \$$(ARTIFACTS_DIR)/bootstrap" \
		"	date -u +\"%Y-%m-%dT%H:%M:%SZ\" > \$$(ARTIFACTS_DIR)/build.txt" \
		"	cd \$$(ARTIFACTS_DIR) && zip -r function.zip bootstrap build.txt" \
		"" \
		".PHONY: build-$$logical_id" \
	> $$out_dir/Makefile; \
	echo "✅ $(FOLDER) compilada correctamente"


compile-all:
	@for folder in $(FOLDERS); do \
		echo ""; \
		echo "=============================="; \
		echo "▶▶ Compilando $$folder"; \
		echo "=============================="; \
		$(MAKE) compile-fn FOLDER=$$folder || exit 1; \
	done
	@echo ""
	@echo "$(SUCCESS)✅ Build completado para todas las funciones$(RESET)"

sam-deploy:
	@sam deploy \
        --profile $(AWS_PROFILE_NAME) \
        --template master-template.yml \
        --stack-name $(STACK_NAME) \
        --s3-bucket $(S3_BUCKET_NAME) \
        --s3-prefix $(S3_PREFIX) \
        --region $(AWS_DEFAULT_REGION) \
        --no-confirm-changeset \
        --capabilities CAPABILITY_IAM \
        --disable-rollback \
        --force-upload

# 	@sam deploy \
# 		$(AWS_PROFILE_ARG) \
# 		$(SAM_ENDPOINT_ARG) \
# 		--template master-template.yml \
# 		--stack-name $(STACK_NAME) \
# 		--s3-bucket $(S3_BUCKET_NAME) \
# 		--s3-prefix $(S3_PREFIX) \
# 		--region $(AWS_DEFAULT_REGION) \
# 		--no-confirm-changeset \
# 		--capabilities CAPABILITY_IAM \
# 		--disable-rollback \
# 		--force-upload

# create-bucket:
# 	@aws $(AWS_PROFILE_ARG) $(AWS_ENDPOINT_ARG) s3api head-bucket \
# 		--bucket $(S3_BUCKET_NAME) >/dev/null 2>&1 || \
# 	aws $(AWS_PROFILE_ARG) $(AWS_ENDPOINT_ARG) s3 mb s3://$(S3_BUCKET_NAME)

localstack-bucket:

	@echo "$(INFO)⚙️ --- Creando bucket S3 en LocalStack ---"
	@echo "$(INFO)⚙️ --- Bucket name: $(S3_BUCKET_NAME) ---"
	@echo "$(INFO)⚙️ --- LOCALSTACK_ENDPOINT name: $(LOCALSTACK_ENDPOINT) ---"

	@aws --endpoint-url $(LOCALSTACK_ENDPOINT) s3api head-bucket \
		--bucket $(S3_BUCKET_NAME) >/dev/null 2>&1 || \
	aws --endpoint-url $(LOCALSTACK_ENDPOINT) s3 mb s3://$(S3_BUCKET_NAME)

deploy-localstack: compile-all
	@echo "$(INFO)⚙️ --- Build LocalStack ---"
	@sam build -t master-template.yml

	@echo "$(INFO)⚙️ --- Desplegando la aplicación SAM a LocalStack ---"
	@echo ""

	@$(MAKE) localstack-bucket
	@$(MAKE) sam-deploy

# 	@echo "$(INFO)⚙️ Bucket name: $(S3_BUCKET_NAME)$(RESET)"
# 	@echo ""

# 	@$(MAKE) create-bucket

# 	@echo "$(INFO)⚙️ --- Deploy LocalStack ---$(RESET)"

# 	@aws \
# 		--endpoint-url $(LOCALSTACK_ENDPOINT) \
# 		cloudformation deploy \
# 		--template-file .aws-sam/build/template.yaml \
# 		--stack-name $(STACK_NAME) \
# 		--capabilities CAPABILITY_IAM \
# 		--no-fail-on-empty-changeset

full-setup: deploy-localstack ## 🧪⚙️ Ejecuta el flujo completo: levanta servicios, genera ZIP, configura Lambda y envía mensaje
	@echo "$(INFO)🔄 Iniciando configuración completa del entorno para el proyecto '$(PROJECT_SLUG)'...$(RESET)"
	@echo ""
	@$(MAKE) update-url-all
	@echo "$(SUCCESS)✅ Proceso de setup completo: servicios levantados, Lambda '$(LAMBDA_FUNCTION_NAME)' configurada y mensaje enviado a la cola '$(SQS_QUEUE_NAME)'.$(RESET)"



###############################################################################
## Docker
###############################################################################
watch: ## 🚀 Inicia la aplicación en modo desarrollo con live-reload (Air).
	@echo "$(SUCCESS)Iniciando en modo watch...$(RESET)"
	$(DC_BASE) -p $(PROJECT_SLUG)-$(APP_ENV) up --remove-orphans --force-recreate


full-delete: ## 💥🧹 Elimina contenedores e imágenes Docker (requiere confirmación explícita)
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
		printf "$(WARNING)❓ ¿Estás ABSOLUTAMENTE seguro de que querés ELIMINAR estos recursos? (s/N)$(RESET) "; \
		read confirm; \
		if [ "$$confirm" != "s" ] && [ "$$confirm" != "S" ]; then \
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
## Helpers
###############################################################################

# Convierte "sqs-consumer" -> "SqsConsumer"
to-pascal = $(shell echo "$(1)" | awk -F '-' '{for(i=1;i<=NF;i++) printf toupper(substr($$i,1,1)) substr($$i,2)}')

# Logical ID SAM: GoFiberCore + PascalCase(folder)
logical-id = $(PROJECT_NAME_PASCAL)$(call to-pascal,$(1))


# .PHONY define los comandos que no producen un archivo con su mismo nombre.
# Es una buena práctica para evitar conflictos y mejorar el rendimiento.
.PHONY: help show_all_variables color-messages check-env install-pkg watch vendor install-all-pkg install-pkg wire-sync compile-fn
