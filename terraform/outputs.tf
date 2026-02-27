output "api_base_url" {
  description = "URL base de la API Gateway"
  # Usamos 'api' para el ID y 'prod' para el nombre del Stage
  value       = var.deploy_mode == "lambda" ? "http://localhost:4566/restapis/${aws_api_gateway_rest_api.api[0].id}/${aws_api_gateway_stage.prod[0].stage_name}/_user_request_/" : null
}

output "log_groups" {
  description = "Nombres de los Log Groups por servicio"
  value = var.deploy_mode == "lambda" ? {
    api           = module.lambda_api[0].log_group_name
    sqs_consumer  = module.lambda_sqs_consumer[0].log_group_name
    dlq_consumer  = module.lambda_dlq_consumer[0].log_group_name
    daily_cron    = module.lambda_daily_cron[0].log_group_name
    every_1min    = module.lambda_every_1min_cron[0].log_group_name
  } : {}
}
