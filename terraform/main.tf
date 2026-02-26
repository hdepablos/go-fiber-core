# ==============================================================================
# 1. Lambdas (Usando el módulo reutilizable)
# ==============================================================================

module "lambda_api" {
  count                 = var.deploy_mode == "lambda" ? 1 : 0
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-api"
  zip_path              = "${local.zip_path}/api.zip"
  memory_size           = 1769
  architectures         = ["arm64"]
  project_name          = var.project_name
  environment           = var.environment
  retention_in_days     = var.log_retention_in_days
  enable_cw_in_local    = var.enable_cloudwatch_in_local
  environment_variables = merge(var.lambda_env_vars, {
    SQS_QUEUE_URL = aws_sqs_queue.main_queue.url
  })
}

module "lambda_sqs_consumer" {
  count                 = var.deploy_mode == "lambda" ? 1 : 0
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-sqs-consumer"
  zip_path              = "${local.zip_path}/sqs-consumer.zip"
  memory_size           = 1769
  architectures         = ["arm64"]
  project_name          = var.project_name
  environment           = var.environment
  retention_in_days     = var.log_retention_in_days
  enable_cw_in_local    = var.enable_cloudwatch_in_local
  environment_variables = merge(var.lambda_env_vars, {
    SQS_QUEUE_URL = aws_sqs_queue.main_queue.url
  })
}

module "lambda_dlq_consumer" {
  count                 = var.deploy_mode == "lambda" ? 1 : 0
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-dlq-consumer"
  zip_path              = "${local.zip_path}/dlq-consumer.zip"
  architectures         = ["arm64"]
  project_name          = var.project_name
  environment           = var.environment
  retention_in_days     = var.log_retention_in_days
  enable_cw_in_local    = var.enable_cloudwatch_in_local
  environment_variables = var.lambda_env_vars
}

module "lambda_daily_cron" {
  count                 = var.deploy_mode == "lambda" ? 1 : 0
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-daily-cron"
  zip_path              = "${local.zip_path}/daily-24-cron.zip"
  architectures         = ["arm64"]
  project_name          = var.project_name
  environment           = var.environment
  retention_in_days     = var.log_retention_in_days
  enable_cw_in_local    = var.enable_cloudwatch_in_local
  environment_variables = merge(var.lambda_env_vars, {
    SQS_QUEUE_URL = aws_sqs_queue.main_queue.url
  })
}

module "lambda_every_1min_cron" {
  count                 = var.deploy_mode == "lambda" ? 1 : 0
  source                = "./modules/lambda_function"
  function_name         = "${local.name_prefix}-1min-cron"
  zip_path              = "${local.zip_path}/every-1min-cron.zip"
  architectures         = ["arm64"]
  project_name          = var.project_name
  environment           = var.environment
  retention_in_days     = var.log_retention_in_days
  enable_cw_in_local    = var.enable_cloudwatch_in_local
  environment_variables = merge(var.lambda_env_vars, {
    SQS_QUEUE_URL = aws_sqs_queue.main_queue.url
  })
}

# ==============================================================================
# 2. Kubernetes / Helm Deployments (Solo si deploy_mode == "eks")
# ==============================================================================

resource "helm_release" "sqs_consumer" {
  count      = var.deploy_mode == "eks" ? 1 : 0
  name       = "sqs-consumer"
  chart      = "${path.module}/charts/gofiber-app"
  namespace  = "default"
  version    = "0.1.0"

  values = [
    yamlencode({
      image = {
        repository = "sqs-consumer"
        tag        = "local"
        pullPolicy = "Never" # Usa la imagen local de OrbStack
      }
      env = merge(var.lambda_env_vars, {
        SQS_QUEUE_URL = aws_sqs_queue.main_queue.url
      })
      autoscaling = {
        enabled         = true
        minReplicaCount = 1
        maxReplicaCount = 5
        triggers = [
          {
            type = "aws-sqs-queue"
            metadata = {
              queueURL      = aws_sqs_queue.main_queue.url
              queueLength   = "5"
              awsRegion     = var.aws_region
              identityOwner = "operator"
            }
          }
        ]
      }
    })
  ]
}

# ==============================================================================
# 3. Otros Recursos (SQS, API Gateway, Events)
# ==============================================================================

# Aquí Terraform buscará los archivos api_gateway.tf, sqs.tf y events.tf
# en la misma carpeta y los combinará automáticamente al hacer el apply.

# ==============================================================================
# 4. CloudWatch Log Metric Filters & Alarms for Slow SQL
#    (Solo se crean si las Lambdas existen, es decir en modo lambda)
# ==============================================================================

resource "aws_cloudwatch_log_metric_filter" "slow_sql_api" {
  count          = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  name           = "${local.name_prefix}-slow-sql-api"
  log_group_name = module.lambda_api[0].log_group_name
  pattern        = "SLOW SQL"

  metric_transformation {
    name      = "SlowSQLCount"
    namespace = var.slow_sql_metric_namespace
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "slow_sql_api" {
  count               = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  alarm_name          = "${local.name_prefix}-slow-sql-api"
  alarm_description   = "Detecta consultas lentas (SLOW SQL) en la API"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "SlowSQLCount"
  namespace           = var.slow_sql_metric_namespace
  period              = var.slow_sql_alarm_period
  statistic           = "Sum"
  threshold           = var.slow_sql_alarm_threshold
  treat_missing_data  = "notBreaching"
}

resource "aws_cloudwatch_log_metric_filter" "slow_sql_sqs" {
  count          = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  name           = "${local.name_prefix}-slow-sql-sqs-consumer"
  log_group_name = module.lambda_sqs_consumer[0].log_group_name
  pattern        = "SLOW SQL"

  metric_transformation {
    name      = "SlowSQLCount"
    namespace = var.slow_sql_metric_namespace
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "slow_sql_sqs" {
  count               = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  alarm_name          = "${local.name_prefix}-slow-sql-sqs-consumer"
  alarm_description   = "Detecta consultas lentas (SLOW SQL) en el SQS Consumer"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "SlowSQLCount"
  namespace           = var.slow_sql_metric_namespace
  period              = var.slow_sql_alarm_period
  statistic           = "Sum"
  threshold           = var.slow_sql_alarm_threshold
  treat_missing_data  = "notBreaching"
}

resource "aws_cloudwatch_log_metric_filter" "slow_sql_dlq" {
  count          = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  name           = "${local.name_prefix}-slow-sql-dlq-consumer"
  log_group_name = module.lambda_dlq_consumer[0].log_group_name
  pattern        = "SLOW SQL"

  metric_transformation {
    name      = "SlowSQLCount"
    namespace = var.slow_sql_metric_namespace
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "slow_sql_dlq" {
  count               = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  alarm_name          = "${local.name_prefix}-slow-sql-dlq-consumer"
  alarm_description   = "Detecta consultas lentas (SLOW SQL) en el DLQ Consumer"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "SlowSQLCount"
  namespace           = var.slow_sql_metric_namespace
  period              = var.slow_sql_alarm_period
  statistic           = "Sum"
  threshold           = var.slow_sql_alarm_threshold
  treat_missing_data  = "notBreaching"
}

resource "aws_cloudwatch_log_metric_filter" "slow_sql_daily" {
  count          = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  name           = "${local.name_prefix}-slow-sql-daily-cron"
  log_group_name = module.lambda_daily_cron[0].log_group_name
  pattern        = "SLOW SQL"

  metric_transformation {
    name      = "SlowSQLCount"
    namespace = var.slow_sql_metric_namespace
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "slow_sql_daily" {
  count               = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  alarm_name          = "${local.name_prefix}-slow-sql-daily-cron"
  alarm_description   = "Detecta consultas lentas (SLOW SQL) en el Daily Cron"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "SlowSQLCount"
  namespace           = var.slow_sql_metric_namespace
  period              = var.slow_sql_alarm_period
  statistic           = "Sum"
  threshold           = var.slow_sql_alarm_threshold
  treat_missing_data  = "notBreaching"
}

resource "aws_cloudwatch_log_metric_filter" "slow_sql_1min" {
  count          = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  name           = "${local.name_prefix}-slow-sql-1min-cron"
  log_group_name = module.lambda_every_1min_cron[0].log_group_name
  pattern        = "SLOW SQL"

  metric_transformation {
    name      = "SlowSQLCount"
    namespace = var.slow_sql_metric_namespace
    value     = "1"
  }
}

resource "aws_cloudwatch_metric_alarm" "slow_sql_1min" {
  count               = (var.slow_sql_alarm_enabled && var.deploy_mode == "lambda") ? 1 : 0
  alarm_name          = "${local.name_prefix}-slow-sql-1min-cron"
  alarm_description   = "Detecta consultas lentas (SLOW SQL) en el 1min Cron"
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = 1
  metric_name         = "SlowSQLCount"
  namespace           = var.slow_sql_metric_namespace
  period              = var.slow_sql_alarm_period
  statistic           = "Sum"
  threshold           = var.slow_sql_alarm_threshold
  treat_missing_data  = "notBreaching"
}
