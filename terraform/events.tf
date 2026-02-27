# --- REGLA: CADA 1 MINUTO ---
resource "aws_cloudwatch_event_rule" "every_1min" {
  count               = var.deploy_mode == "lambda" ? 1 : 0
  name                = "${local.name_prefix}-every-1min-rule"
  schedule_expression = "rate(1 minute)"
}

resource "aws_cloudwatch_event_target" "every_1min_target" {
  count     = var.deploy_mode == "lambda" ? 1 : 0
  rule      = aws_cloudwatch_event_rule.every_1min[0].name
  target_id = "Every1MinTarget"
  arn       = module.lambda_every_1min_cron[0].function_arn
}

resource "aws_lambda_permission" "allow_cloudwatch_1min" {
  count         = var.deploy_mode == "lambda" ? 1 : 0
  statement_id  = "AllowExecutionFromCloudWatch-1min"
  action        = "lambda:InvokeFunction"
  function_name = module.lambda_every_1min_cron[0].function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.every_1min[0].arn
}

# --- REGLA: DIARIA (24 HORAS) ---
resource "aws_cloudwatch_event_rule" "daily_cron" {
  count               = var.deploy_mode == "lambda" ? 1 : 0
  name                = "${local.name_prefix}-daily-cron-rule"
  schedule_expression = "rate(2 minutes)"
}

resource "aws_cloudwatch_event_target" "daily_cron_target" {
  count     = var.deploy_mode == "lambda" ? 1 : 0
  rule      = aws_cloudwatch_event_rule.daily_cron[0].name
  target_id = "DailyCronTarget"
  arn       = module.lambda_daily_cron[0].function_arn
}

resource "aws_lambda_permission" "allow_cloudwatch_daily" {
  count         = var.deploy_mode == "lambda" ? 1 : 0
  statement_id  = "AllowExecutionFromCloudWatch-daily"
  action        = "lambda:InvokeFunction"
  function_name = module.lambda_daily_cron[0].function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.daily_cron[0].arn
}
