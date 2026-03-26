locals {
  name_prefix = "gofibercore-${var.environment}"
  zip_path    = "../sam-compile"
  
  # Lógica Híbrida de Secretos:
  # En LOCAL: Usamos las variables pasadas por archivo (var.app_env_vars)
  # En PROD/STAGING: Deberíamos obtenerlas de SSM (o usarlas vacías y que la app las lea)
  # Por simplicidad y compatibilidad con el flujo actual del usuario,
  # mantendremos var.app_env_vars como fuente de verdad inyectada por el CI/CD o Makefile.
  # 
  # NOTA: Para implementar lectura directa de SSM en tiempo de ejecución de Lambda,
  # se requeriría cambiar el código Go para usar AWS SDK SSM.
  #
  # Sin embargo, podemos hacer que Terraform busque los valores en SSM y los inyecte.
  # Pero eso requiere que las claves sean conocidas de antemano.
  #
  # La solución más alineada al flujo actual del usuario ("crear en .env y que funcione")
  # es la que ya tiene: `env-manager` inyecta valores.
  # 
  # Para PRODUCCIÓN REAL con SSM, el cambio es en CÓMO se llena `var.app_env_vars`.
  # No necesitamos cambiar el código de Terraform aquí, sino la fuente de los datos.
}


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
  environment_variables = merge(var.app_env_vars, {
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
  environment_variables = merge(var.app_env_vars, {
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
  environment_variables = var.app_env_vars
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
  environment_variables = merge(var.app_env_vars, {
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
  environment_variables = merge(var.app_env_vars, {
    SQS_QUEUE_URL = aws_sqs_queue.main_queue.url
  })
}

# ==============================================================================
# 2. Kubernetes / Helm Deployments (Solo si deploy_mode == "eks")
# ==============================================================================

resource "helm_release" "sqs_consumer" {
  count            = var.deploy_mode == "eks" ? 1 : 0
  name             = "sqs-consumer"
  chart            = "${path.module}/charts/gofiber-app"
  namespace        = "default"
  version          = "0.1.0"
  create_namespace = true
  force_update     = true
  recreate_pods    = true

  values = [
    yamlencode({
      image = {
        repository = "sqs-consumer"
        tag        = "local"
        pullPolicy = "Never"
      }
      env = merge(var.app_env_vars, {
        SQS_QUEUE_URL = replace(aws_sqs_queue.main_queue.url, "localhost", "host.docker.internal")
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

resource "helm_release" "api" {
  count            = var.deploy_mode == "eks" ? 1 : 0
  name             = "api"
  chart            = "${path.module}/charts/gofiber-app"
  namespace        = "default"
  version          = "0.1.0"
  create_namespace = true
  force_update     = true
  recreate_pods    = true

  values = [
    yamlencode({
      image = {
        repository = "api"
        tag        = "local"
        pullPolicy = "Never"
      }
      service = {
        enabled    = true
        type       = "LoadBalancer"
        port       = 80
        targetPort = 9009
      }
      env = merge(var.app_env_vars, {
        SQS_QUEUE_URL = replace(aws_sqs_queue.main_queue.url, "localhost", "host.docker.internal")
      })
    })
  ]
}

resource "helm_release" "dlq_consumer" {
  count            = var.deploy_mode == "eks" ? 1 : 0
  name             = "dlq-consumer"
  chart            = "${path.module}/charts/gofiber-app"
  namespace        = "default"
  version          = "0.1.0"
  create_namespace = true
  force_update     = true
  recreate_pods    = true

  values = [
    yamlencode({
      image = {
        repository = "dlq-consumer"
        tag        = "local"
        pullPolicy = "Never"
      }
      env = merge(var.app_env_vars, {
        SQS_QUEUE_URL = replace(aws_sqs_queue.dlq.url, "localhost", "host.docker.internal")
      })
      autoscaling = {
        enabled         = true
        minReplicaCount = 1
        maxReplicaCount = 5
        triggers = [
          {
            type = "aws-sqs-queue"
            metadata = {
              queueURL      = aws_sqs_queue.dlq.url
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

resource "helm_release" "every_1min_cron" {
  count            = var.deploy_mode == "eks" ? 1 : 0
  name             = "1min-cron"
  chart            = "${path.module}/charts/gofiber-app"
  namespace        = "default"
  version          = "0.1.0"
  create_namespace = true
  force_update     = true
  recreate_pods    = true

  values = [
    yamlencode({
      image = {
        repository = "every-1min-cron"
        tag        = "local"
        pullPolicy = "Never"
      }
      env = merge(var.app_env_vars, {
        SQS_QUEUE_URL = replace(aws_sqs_queue.main_queue.url, "localhost", "host.docker.internal")
      })
      cronjob = {
        enabled  = true
        schedule = "*/1 * * * *"
      }
    })
  ]
}

resource "helm_release" "daily_cron" {
  count            = var.deploy_mode == "eks" ? 1 : 0
  name             = "daily-cron"
  chart            = "${path.module}/charts/gofiber-app"
  namespace        = "default"
  version          = "0.1.0"
  create_namespace = true
  force_update     = true
  recreate_pods    = true

  values = [
    yamlencode({
      image = {
        repository = "daily-24-cron"
        tag        = "local"
        pullPolicy = "Never"
      }
      env = merge(var.app_env_vars, {
        SQS_QUEUE_URL = replace(aws_sqs_queue.main_queue.url, "localhost", "host.docker.internal")
      })
      cronjob = {
        enabled  = true
        schedule = "0 0 * * *"
      }
    })
  ]
}
