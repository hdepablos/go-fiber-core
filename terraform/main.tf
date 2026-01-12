# ==============================================================================
# 1. Lambdas (Usando el módulo reutilizable)
# ==============================================================================

module "lambda_api" {
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-api"
  zip_path              = "${local.zip_path}/api.zip"
  environment_variables = var.lambda_env_vars # Inyecta las variables de tu .tfvars
}

module "lambda_sqs_consumer" {
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-sqs-consumer"
  zip_path              = "${local.zip_path}/sqs-consumer.zip"
  environment_variables = var.lambda_env_vars
}

module "lambda_dlq_consumer" {
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-dlq-consumer"
  zip_path              = "${local.zip_path}/dlq-consumer.zip"
  environment_variables = var.lambda_env_vars
}

module "lambda_daily_cron" {
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-daily-cron"
  zip_path              = "${local.zip_path}/daily-24-cron.zip"
  environment_variables = var.lambda_env_vars
}

module "lambda_every_1min_cron" {
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-1min-cron"
  zip_path              = "${local.zip_path}/every-1min-cron.zip"
  environment_variables = var.lambda_env_vars
}

# ==============================================================================
# 2. Otros Recursos (SQS, API Gateway, Events)
# ==============================================================================

# Aquí Terraform buscará los archivos api_gateway.tf, sqs.tf y events.tf
# en la misma carpeta y los combinará automáticamente al hacer el apply.
