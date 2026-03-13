variable "environment" {
  description = "local, staging o prod"
  type        = string
  default     = "local"
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "localstack_endpoint_base" {
  description = "Endpoint base de LocalStack para Terraform (host)"
  type        = string
  default     = "http://127.0.0.1:4566"
}

variable "localstack_s3_endpoint" {
  description = "Endpoint S3 de LocalStack (virtual-host) para Terraform (host)"
  type        = string
  default     = "http://s3.127.0.0.1.localstack.cloud:4566"
}

# --- ESTA ES LA VARIABLE QUE FALTABA ---
variable "app_env_vars" {
  description = "Mapa de variables de entorno de la aplicación (común para Lambda y EKS)"
  type        = map(string)
  default     = {}
}

# --- ESTA TAMBIÉN ES NECESARIA PARA EL PROVIDER ---
variable "project_name" {
  description = "Nombre del proyecto para etiquetas"
  type        = string
  default     = "GoFiberCore"
}

variable "log_retention_in_days" {
  description = "Días de retención para CloudWatch Logs"
  type        = number
  default     = 7
}

variable "enable_cloudwatch_in_local" {
  description = "Si es local, crear Log Groups en CloudWatch (LocalStack) en lugar de solo logs locales"
  type        = bool
  default     = false
}

# Nueva variable para modo de despliegue
variable "deploy_mode" {
  description = "Modo de despliegue: 'lambda' (por defecto) o 'eks'"
  type        = string
  default     = "lambda"
  validation {
    condition     = contains(["lambda", "eks"], var.deploy_mode)
    error_message = "El modo de despliegue debe ser 'lambda' o 'eks'."
  }
}

variable "kube_context" {
  description = "Contexto de Kubernetes a usar (por defecto 'orbstack' en local)"
  type        = string
  default     = "orbstack"
}



variable "slow_sql_alarm_enabled" {
  description = "Habilita creación de Metric Filters y Alarms para 'SLOW SQL' en CloudWatch"
  type        = bool
  default     = true
}

variable "slow_sql_alarm_threshold" {
  description = "Umbral de sumatoria de eventos 'SLOW SQL' para disparar la alarma"
  type        = number
  default     = 1
}

variable "slow_sql_alarm_period" {
  description = "Periodo de evaluación (segundos) para la alarma de 'SLOW SQL'"
  type        = number
  default     = 300
}

variable "slow_sql_metric_namespace" {
  description = "Namespace de la métrica en CloudWatch para SLOW SQL"
  type        = string
  default     = "DB/SlowQueries"
}
