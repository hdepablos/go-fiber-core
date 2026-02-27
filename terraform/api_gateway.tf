# 1. Definición de la API REST (Nombre dinámico)
resource "aws_api_gateway_rest_api" "api" {
  count       = var.deploy_mode == "lambda" ? 1 : 0
  name        = "${local.name_prefix}-api"
  description = "API Gateway para Go Fiber Core (${var.environment})"
}

# 2. Recurso Proxy (captura todas las rutas: /{proxy+})
resource "aws_api_gateway_resource" "proxy" {
  count       = var.deploy_mode == "lambda" ? 1 : 0
  rest_api_id = aws_api_gateway_rest_api.api[0].id
  parent_id   = aws_api_gateway_rest_api.api[0].root_resource_id
  path_part   = "{proxy+}"
}

# 3. Método del Proxy
resource "aws_api_gateway_method" "proxy_method" {
  count         = var.deploy_mode == "lambda" ? 1 : 0
  rest_api_id   = aws_api_gateway_rest_api.api[0].id
  resource_id   = aws_api_gateway_resource.proxy[0].id
  http_method   = "ANY"
  authorization = "NONE"
}

# 3.1 Método Raíz (para GET /)
resource "aws_api_gateway_method" "root_method" {
  count         = var.deploy_mode == "lambda" ? 1 : 0
  rest_api_id   = aws_api_gateway_rest_api.api[0].id
  resource_id   = aws_api_gateway_rest_api.api[0].root_resource_id
  http_method   = "ANY"
  authorization = "NONE"
}

# 4. Integración con la Lambda (Proxy)
resource "aws_api_gateway_integration" "lambda_integration" {
  count                   = var.deploy_mode == "lambda" ? 1 : 0
  rest_api_id             = aws_api_gateway_rest_api.api[0].id
  resource_id             = aws_api_gateway_resource.proxy[0].id
  http_method             = aws_api_gateway_method.proxy_method[0].http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = "arn:aws:apigateway:${var.aws_region}:lambda:path/2015-03-31/functions/${module.lambda_api[0].function_arn}/invocations"
}

# 4.1 Integración con la Lambda (Raíz)
resource "aws_api_gateway_integration" "root_integration" {
  count                   = var.deploy_mode == "lambda" ? 1 : 0
  rest_api_id             = aws_api_gateway_rest_api.api[0].id
  resource_id             = aws_api_gateway_rest_api.api[0].root_resource_id
  http_method             = aws_api_gateway_method.root_method[0].http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = "arn:aws:apigateway:${var.aws_region}:lambda:path/2015-03-31/functions/${module.lambda_api[0].function_arn}/invocations"
}

# 5. Permiso para API Gateway
resource "aws_lambda_permission" "apigw_lambda" {
  count         = var.deploy_mode == "lambda" ? 1 : 0
  statement_id  = "AllowExecutionFromAPIGateway-${var.environment}"
  action        = "lambda:InvokeFunction"
  function_name = module.lambda_api[0].function_arn
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.api[0].execution_arn}/*/*"
}

# 6. Deployment
resource "aws_api_gateway_deployment" "deployment" {
  count       = var.deploy_mode == "lambda" ? 1 : 0
  depends_on  = [
    aws_api_gateway_integration.lambda_integration,
    aws_api_gateway_integration.root_integration
  ]
  rest_api_id = aws_api_gateway_rest_api.api[0].id

  # Este bloque asegura que si cambias recursos o métodos, el despliegue se actualice
  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_resource.proxy,
      aws_api_gateway_method.proxy_method,
      aws_api_gateway_method.root_method,
      aws_api_gateway_integration.lambda_integration,
      aws_api_gateway_integration.root_integration,
    ]))
  }

  # LA SOLUCIÓN AL ERROR: Crea el nuevo antes de borrar el viejo vinculado al Stage
  lifecycle {
    create_before_destroy = true
  }
}

# 7. Stage
resource "aws_api_gateway_stage" "prod" {
  count         = var.deploy_mode == "lambda" ? 1 : 0
  deployment_id = aws_api_gateway_deployment.deployment[0].id
  rest_api_id   = aws_api_gateway_rest_api.api[0].id
  stage_name    = "Prod"
}

# 8. Output de la URL (Optimizado para LocalStack)
output "base_url" {
  value = var.deploy_mode == "lambda" ? (var.environment == "local" ? "http://localhost:4566/restapis/${aws_api_gateway_rest_api.api[0].id}/${aws_api_gateway_stage.prod[0].stage_name}/_user_request_/" : aws_api_gateway_stage.prod[0].invoke_url) : ""
}
