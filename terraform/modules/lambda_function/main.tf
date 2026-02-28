variable "function_name" {}
variable "zip_path" {}

variable "handler" {
  default = "bootstrap"
}

variable "runtime" {
  default = "provided.al2023"
}

variable "memory_size" {
  default = 128
}

variable "project_name" {
  type    = string
  default = "GoFiberCore"
}

variable "environment" {
  type    = string
  default = "local"
}

variable "retention_in_days" {
  type    = number
  default = 7
}

variable "enable_cw_in_local" {
  type    = bool
  default = false
}
variable "architectures" {
  type    = list(string)
  default = ["x86_64"]
}
variable "environment_variables" {
  type    = map(string)
  default = {}
}

resource "aws_lambda_function" "this" {
  function_name    = var.function_name
  filename         = var.zip_path
  source_code_hash = filebase64sha256(var.zip_path)
  handler          = var.handler
  runtime          = var.runtime
  architectures    = var.architectures
  role             = aws_iam_role.lambda_exec.arn
  timeout          = 30 # Aumentado a 30s para conexiones a DB/Redis
  memory_size      = var.memory_size

  environment {
    variables = var.environment_variables
  }
  
  publish = true
}

resource "aws_lambda_alias" "prod" {
  name             = "prod"
  description      = "Alias de Producción"
  function_name    = aws_lambda_function.this.function_name
  function_version = aws_lambda_function.this.version
}

resource "aws_cloudwatch_log_group" "this" {
  # Para Lambda, el grupo de logs oficial es /aws/lambda/<function_name>.
  # Esto garantiza compatibilidad total con el runtime de AWS Lambda,
  # CloudWatch y herramientas como aws logs tail.
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = var.retention_in_days
}

# --- IAM ROLE ---
resource "aws_iam_role" "lambda_exec" {
  name = "${var.function_name}_role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_sqs" {
  role       = aws_iam_role.lambda_exec.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaSQSQueueExecutionRole"
}

# --- OUTPUTS ---
output "function_arn" {
  description = "ARN de la función"
  value       = aws_lambda_function.this.arn
}

output "function_name" {
  description = "Nombre de la función"
  value       = aws_lambda_function.this.function_name
}

output "invoke_arn" {
  description = "ARN de invocación para API Gateway"
  value       = aws_lambda_function.this.invoke_arn
}

output "role_name" {
  description = "Nombre del rol IAM de la función"
  value       = aws_iam_role.lambda_exec.name
}

output "log_group_name" {
  description = "Nombre del Log Group asociado al servicio"
  value       = aws_cloudwatch_log_group.this.name
}

output "log_group_arn" {
  description = "ARN del Log Group (puede ser null en local si está deshabilitado)"
  value       = aws_cloudwatch_log_group.this.arn
}
