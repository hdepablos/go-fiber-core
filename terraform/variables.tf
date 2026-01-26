variable "environment" {
  description = "local, staging o prod"
  type        = string
  default     = "local"
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

# --- ESTA ES LA VARIABLE QUE FALTABA ---
variable "lambda_env_vars" {
  description = "Mapa de variables de entorno inyectadas desde .tfvars"
  type        = map(string)
  default     = {}
}

# --- ESTA TAMBIÉN ES NECESARIA PARA EL PROVIDER ---
variable "project_name" {
  description = "Nombre del proyecto para etiquetas"
  type        = string
  default     = "GoFiberCore"
}

locals {
  # Nombre base para recursos
  name_prefix = "gofibercore-${var.environment}"

  # Ruta de los zips (relativa a la carpeta terraform/)
  zip_path = "../sam-compile"
}
