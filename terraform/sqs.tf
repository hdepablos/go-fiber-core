# --- COLA DEAD LETTER (DLQ) ---
resource "aws_sqs_queue" "dlq" {
  name = "gofibercoredlq" # Nombre exacto según tu lógica de variables
}

# --- COLA PRINCIPAL CON REDRIVE POLICY ---
resource "aws_sqs_queue" "main_queue" {
  name                       = "gofibercorequeue"
  visibility_timeout_seconds = 30

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.dlq.arn
    maxReceiveCount     = 3
  })
}

# --- TRIGGER: SQS PRINCIPAL -> SQS CONSUMER ---
resource "aws_lambda_event_source_mapping" "sqs_trigger" {
  event_source_arn        = aws_sqs_queue.main_queue.arn
  function_name           = module.lambda_sqs_consumer.function_name
  batch_size              = 1
  function_response_types = ["ReportBatchItemFailures"]
}

# --- TRIGGER: DLQ -> DLQ CONSUMER ---
resource "aws_lambda_event_source_mapping" "dlq_trigger" {
  event_source_arn = aws_sqs_queue.dlq.arn
  function_name    = module.lambda_dlq_consumer.function_name
  batch_size       = 1
}

# --- PERMISOS DE ENVÍO (SEND MESSAGE) ---

# 1. API: Necesita enviar mensajes a la cola principal para iniciar procesos
resource "aws_iam_role_policy" "api_send_msg" {
  name = "api_send_msg_policy"
  role = module.lambda_api.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "sqs:SendMessage"
        Resource = aws_sqs_queue.main_queue.arn
      }
    ]
  })
}

# 2. SQS Consumer: Necesita auto-invocarse (re-encolar)
resource "aws_iam_role_policy" "sqs_consumer_send_msg" {
  name = "sqs_consumer_send_msg_policy"
  role = module.lambda_sqs_consumer.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "sqs:SendMessage"
        Resource = aws_sqs_queue.main_queue.arn
      }
    ]
  })
}

# 3. Daily Cron: Tarea programada diaria
resource "aws_iam_role_policy" "daily_cron_send_msg" {
  name = "daily_cron_send_msg_policy"
  role = module.lambda_daily_cron.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "sqs:SendMessage"
        Resource = aws_sqs_queue.main_queue.arn
      }
    ]
  })
}

# 4. 1min Cron: Tarea programada cada minuto
resource "aws_iam_role_policy" "every_1min_cron_send_msg" {
  name = "every_1min_cron_send_msg_policy"
  role = module.lambda_every_1min_cron.role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "sqs:SendMessage"
        Resource = aws_sqs_queue.main_queue.arn
      }
    ]
  })
}
