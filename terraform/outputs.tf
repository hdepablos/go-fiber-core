output "api_base_url" {
  description = "URL base de la API Gateway"
  # Usamos 'api' para el ID y 'prod' para el nombre del Stage
  value       = "http://localhost:4566/restapis/${aws_api_gateway_rest_api.api.id}/${aws_api_gateway_stage.prod.stage_name}/_user_request_/"
}
