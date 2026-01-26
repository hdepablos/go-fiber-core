variable "function_name" {}
variable "zip_path" {}
variable "handler" { default = "bootstrap" }
variable "runtime" { default = "provided.al2023" }
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
  role             = aws_iam_role.lambda_exec.arn
  timeout          = 30 # Aumentado a 30s para conexiones a DB/Redis
  memory_size      = 128

  environment {
    variables = var.environment_variables
  }
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
