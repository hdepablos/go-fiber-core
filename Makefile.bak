# Makefile

# Carga las variables desde el archivo .env y las exporta.
include .env
export

# .DEFAULT_GOAL define el comando que se ejecuta si solo escribís "make".
.DEFAULT_GOAL := help

# --- Variables y Configuración ------------------------------------------------
# Define colores para una salida más amigable en la terminal.
GREEN  := \033[0;32m
YELLOW := \033[1;33m
NC     := \033[0m # Sin Color

# Configuración del entorno y stack (local por defecto).
APP_ENV ?= local
STACK   ?= watch

# Construye el nombre del archivo docker-compose a usar.
DOCKER_FILE := docker-compose-$(APP_ENV).yml
ifeq ($(STACK),traefik)
    DOCKER_FILE = docker-compose-traefik-$(APP_ENV).yml
endif

# Define los comandos base de Docker Compose.
DC_BASE = docker compose -f docker-compose-base.yml -f $(DOCKER_FILE)
DC_RUN  = $(DC_BASE) run --rm $(SERVICE_NAME)

# --- Ayuda --------------------------------------------------------------------
help: ## ℹ️ Muestra todos los comandos disponibles con su descripción.
	@awk -F ':|##' '/^[a-zA-Z0-9_-]+:.*?##/ {printf "\033[36m%-20s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST)

# --- Verifica que existan las variables de entorno necesarias para el Makefile
check-env:
	@echo "$(GREEN)Verificando variables de entorno en el .env indispensables para el Makefile$(NC)"
	@if [ -z "$(APP_NAME)" ]; then echo "❌ APP_NAME no está definido en .env"; exit 1; fi
	@if [ -z "$(SERVICE_NAME)" ]; then echo "❌ SERVICE_NAME no está definido en .env"; exit 1; fi
	@if [ -z "$(DOMAIN)" ]; then echo "❌ SERVICE_NAME no está definido en .env"; exit 1; fi
	@if [ -z "$(STACK)" ]; then echo "❌ STACK no está definido en .env"; exit 1; fi
	@echo "✅ Todas las variables de entorno están definidas en .env"

# --- Ciclo de Vida de la Aplicación -------------------------------------------
watch: ## 🚀 Inicia la aplicación en modo desarrollo con live-reload (Air).
	@if [ "$(STACK)" != "watch" ]; then \
		echo "❌ La variable STACK en el .env debe ser 'watch'"; \
		exit 1; \
	fi
	@echo "$(GREEN)Iniciando en modo watch...$(NC)"
	$(DC_BASE) -p $(APP_NAME)-$(APP_ENV) up --remove-orphans --force-recreate

build: ## 🛠️ Compila la aplicación Go para producción (Linux amd64).
	@echo "$(GREEN)Compilando la aplicación...$(NC)"
	@$(DC_RUN) go build -o bin/app cmd/api/main.go

build-dev: build ## 🚀 Inicia la aplicación en modo desarrollo.
	@echo "$(GREEN)Compilando el docker compose$(NC)"
	docker compose -f docker-compose-base.yml -f docker-compose-local.yml build --no-cache

prod: build ## 🚢 Despliega la aplicación en modo producción.
	@echo "$(GREEN)Desplegando en modo producción...$(NC)"
	@docker compose -p $(APP_NAME)-prod -f docker-compose-base.yml -f docker-compose-prod.yml up --remove-orphans -d

# --- Gestión de Dependencias ------------------------------------------------
vendor: ## 📦 Actualiza el archivo go.mod y la carpeta vendor.
	@echo "$(GREEN)Ordenando y vendoring dependencias...$(NC)"
	@$(DC_RUN) go mod tidy
	@$(DC_RUN) go mod vendor

install-pkg: ## 📥 Instala o actualiza un paquete Go específico. Uso: make install-pkg pkg=...
	@echo "$(GREEN)Instalando/actualizando paquete: $(pkg)...$(NC)"
	@$(DC_RUN) go get -u $(pkg)
	@make vendor

# *****************************************************************************
# Instala todas las dependencias básicas
# *****************************************************************************
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

	make vendor

clean-cache: ## 🧹 Limpia la caché de módulos de Go.
	@echo "$(GREEN)Limpiando caché de módulos...$(NC)"
	@$(DC_RUN) go clean -modcache

# --- Pruebas y Calidad de Código --------------------------------------------
test-clean: ## 🧼 Limpia el caché de los tests de Go.
	@echo "🧼 Limpiando el caché de los tests de Go..."
	@$(DC_RUN) go clean -testcache

test: ## 🧪 Ejecuta todos los tests unitarios con formato amigable.
	@chmod +x ./scripts/run_tests.sh
	@$(DC_RUN) bash ./scripts/run_tests.sh

test-pkg: ## 🧪 Ejecuta tests unitarios de un paquete. Uso: make test-pkg PKG=./...
	@echo "🧪 Ejecutando tests unitarios para el paquete: $(PKG)"
	@$(DC_RUN) go test -v $(PKG)

# 	docker compose -f docker-compose-base.yml -f docker-compose-local.yml run --build --rm go-fiber-core go test -v ./internal/services/pagination


test-func: ## 🔬 Ejecuta un test unitario específico. Uso: make test-func PKG=./... FUNC=Test...
	@echo "🔬 Ejecutando test unitario: $(FUNC) en el paquete: $(PKG)"
	@$(DC_RUN) go test -v -run $(FUNC) $(PKG)

test-pkg-int: ## 🔗 Ejecuta tests de INTEGRACIÓN de un paquete. Uso: make test-pkg-int PKG=./...
	@echo "🔗 Ejecutando tests de INTEGRACIÓN para el paquete: $(PKG)"
	@$(DC_RUN) go test -v -tags=integration $(PKG)

test-func-int: ## 🔬🔗 Ejecuta un test de INTEGRACIÓN específico. Uso: make test-func-int PKG=./... FUNC=Test...
	@echo "🔬🔗 Ejecutando test de INTEGRACIÓN: $(FUNC) en el paquete: $(PKG)"
	@$(DC_RUN) go test -v -tags=integration -run $(FUNC) $(PKG)

coverage: ## 📊 Genera reporte de cobertura COMPLETO (unitarios + integración).
	@chmod +x ./scripts/generate_coverage_report.sh
	@echo "📊 Generando reporte de cobertura COMPLETO..."
	@$(DC_RUN) go test -tags=integration -coverprofile=coverage.out ./...
	@$(DC_RUN) bash ./scripts/generate_coverage_report.sh

coverage-unit: ## 📊 Genera reporte de cobertura RÁPIDO (solo unitarios).
	@chmod +x ./scripts/generate_coverage_report.sh
	@echo "📊 Generando reporte de cobertura para tests UNITARIOS..."
	@$(DC_RUN) go test -coverprofile=coverage.out ./...
	@$(DC_RUN) bash ./scripts/generate_coverage_report.sh

lint: ##  lint: 🎨 Analiza el código en busca de errores y malas prácticas con golangci-lint.
	@echo "🧹 Limpiando la caché de golangci-lint..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint cache clean
	@echo "Limpiando contenedores huérfanos..."
	@docker compose -f docker-compose-local-lint.yml down --remove-orphans
	@echo "Ejecutando linter..."
	@docker compose -f docker-compose-local-lint.yml build --no-cache lint && docker compose -f docker-compose-local-lint.yml run --rm lint run --timeout=2m

lint-check-config: ## 🔍 Verifica qué archivos está usando golangci-lint
	@echo "🔍 Verificando configuración de golangci-lint..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint config path
	@echo ""
	@echo "📄 Mostrando configuración cargada:"
	@docker compose -f docker-compose-local-lint.yml run --rm lint config dump


lint-verbose: ## 🔍 Ejecuta el linter en modo verbose para ver qué archivos analiza
	@echo "🔍 Ejecutando linter en modo verbose..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint run -v --timeout=2m


lint-test: ## 🧪 Prueba si wire_gen.go está siendo ignorado
	@echo "🧪 Listando archivos que el linter va a analizar..."
	@docker compose -f docker-compose-local-lint.yml run --rm lint run --issues-exit-code=0 2>&1 | grep -i "wire_gen" || echo "✅ wire_gen.go NO aparece en la salida (está siendo ignorado)"


## --------------------------------------------------------------------------
## Gestión de Base de Datos 🚀
## --------------------------------------------------------------------------

# Crea un nuevo archivo de migración SQL.
# Uso: make create-migration name=nombre_descriptivo_de_la_migracion
create-migration:
	@if [ -z "$(name)" ]; then \
		echo "❌ Por favor, especifique el nombre. Uso: make create-migration name=create_users_table"; \
		exit 1; \
	fi
	@echo "🌱 Creando migración: $(name)..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations create $(name)

# Aplica todas las migraciones pendientes.
migrate-up:
	@echo "🚀 Aplicando migraciones..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations up

# Revierte la última o X migraciones aplicadas.
migrate-down:
	@echo "⏪ Revertiendo migración(es)..."
	@step=$(word 2,$(MAKECMDGOALS)); \
	if [ "$$step" = "" ]; then step=1; fi; \
	$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations down --step=$$step
# Captura args como '2' o '3' etc... y evita errores
%:
	@:

# Revierte todas las migraciones.
migrate-down-all:
	@echo "🧹 Revertiendo TODAS las migraciones..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations reset

# Refresca todas las migraciones: primero baja todo, luego aplica todas.
migrate-refresh:
	@echo "🔄 Refrescando migraciones: bajando todo y aplicando nuevamente..."
	@$(MAKE) migrate-down-all
	@$(MAKE) migrate-up

# Muestra el estado de todas las migraciones.
migrate-status:
	@echo "📊 Estado actual de las migraciones:"
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go migrations status


# --- Herramientas y Comandos CLI --------------------------------------------
run-cli: ## ▶️ Ejecuta un comando CLI personalizado. Uso: make run-cli c="comando --flag=valor"
	@echo "▶️ Ejecutando comando CLI: $(c)..."
	@$(DC_RUN) go run ./cmd/cmd-cli/main.go $(c)


create-command: ## ✨ Crea un nuevo comando Cobra. Uso: make create-command name=...
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


local-ssl: ## 🔐 Genera certificados SSL locales para desarrollo.
	@./scripts/generate-local-cert.sh $(DOMAIN) $(SERVICE_NAME) certs

clean-certs: ## 🧹 Elimina los certificados SSL locales.
	@rm -rf certs
	@echo "🧹 Certificados eliminados."

go-version: ## 🐹 Muestra la versión de Go utilizada en el contenedor.
	@$(DC_RUN) go version

create-host: # Verifica si existe de lo contrario lo crea
	@./scripts/create-host.sh $(DOMAIN)

to-container: ## 💻 Abre una terminal (shell) dentro del contenedor de la aplicación.
	@$(DC_RUN) sh


# --- Inyección de Dependencias (Wire) ---------------------------------------
wire: ## 🧬 Genera el código de inyección de dependencias con Google Wire.
	@echo "$(GREEN)Generando inyección de dependencias con Wire...$(NC)"
	@$(DC_RUN) wire gen -tags wireinject ./cmd/api/di

wire-sync: wire vendor ## 🧬+📦 Genera código de Wire y actualiza go.mod/vendor después.
	@echo "$(GREEN)Proceso de Wire y vendor completado.$(NC)"

# .PHONY define los comandos que no producen un archivo con su mismo nombre.
# Es una buena práctica para evitar conflictos y mejorar el rendimiento.
.PHONY: help watch build prod vendor install-pkg clean-cache test-clean test test-pkg test-func test-pkg-int test-func-int coverage coverage-unit lint create-migration migrate-up migrate-down migrate-down-all migrate-status run-cli create-command local-ssl clean-certs go-version to-container wire wire-sync
