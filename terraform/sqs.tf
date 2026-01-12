# --- COLA DEAD LETTER (DLQ) ---
resource "aws_sqs_queue" "dlq" {
  name = "gofibercoredlq" # Nombre exacto según tu lógica de variables
}

# --- COLA PRINCIPAL CON REDRIVE POLICY ---
resource "aws_sqs_queue" "main_queue" {
  name                       = "gofibercorequeue" # El nombre que esperas
  visibility_timeout_seconds = 30

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 3 # Al 4to intento fallido, salta a la DLQ
  })
}

# --- TRIGGER: SQS PRINCIPAL -> SQS CONSUMER ---
resource "aws_lambda_event_source_mapping" "sqs_trigger" {
  event_source_arn = aws_sqs_queue.main_queue.arn
  function_name    = module.lambda_sqs_consumer.function_name
  batch_size       = 1
}

# --- TRIGGER: DLQ -> DLQ CONSUMER ---
resource "aws_lambda_event_source_mapping" "dlq_trigger" {
  event_source_arn = aws_sqs_queue.dlq.arn
  function_name    = module.lambda_dlq_consumer.function_name
  batch_size       = 1
}
