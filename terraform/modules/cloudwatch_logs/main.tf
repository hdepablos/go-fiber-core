variable "project_name" {
  description = "Nombre del proyecto para prefijo del Log Group"
  type        = string
}

variable "service_name" {
  description = "Nombre del servicio (api, sqs-consumer, cron, etc.)"
  type        = string
}

variable "environment" {
  description = "Entorno actual (local|staging|prod)"
  type        = string
  default     = "local"
}

variable "retention_in_days" {
  description = "Retención de logs en días"
  type        = number
  default     = 7
}

variable "enable_in_local" {
  description = "Si es local, crear el Log Group (útil para usar CloudWatch en LocalStack)"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags adicionales opcionales"
  type        = map(string)
  default     = {}
}

variable "create_writer_policy" {
  description = "Crear una IAM Policy con permisos mínimos para escribir logs (para servicios NO Lambda)"
  type        = bool
  default     = false
}

locals {
  log_group_name = "/app/${var.project_name}/${var.service_name}"
}

resource "aws_cloudwatch_log_group" "this" {
  count             = var.environment == "local" && var.enable_in_local == false ? 0 : 1
  name              = local.log_group_name
  retention_in_days = var.retention_in_days

  tags = var.tags
}

data "aws_iam_policy_document" "logs_basic_write" {
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = [
      "arn:aws:logs:*:*:log-group:${local.log_group_name}:*"
    ]
  }
}

resource "aws_iam_policy" "logs_basic_write" {
  count       = var.create_writer_policy ? 1 : 0
  name        = "${var.project_name}-${var.service_name}-cwlogs-write"
  description = "Permisos mínimos para escribir en ${local.log_group_name}"
  policy      = data.aws_iam_policy_document.logs_basic_write.json
}

output "log_group_name" {
  value       = local.log_group_name
  description = "Nombre del Log Group"
}

output "log_group_arn" {
  value       = try(aws_cloudwatch_log_group.this[0].arn, null)
  description = "ARN del Log Group (null si no se creó en local)"
}

output "logs_basic_write_policy_arn" {
  value       = try(aws_iam_policy.logs_basic_write[0].arn, null)
  description = "ARN de la policy de escritura básica (si create_writer_policy=true)"
}
