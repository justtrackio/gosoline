locals {
  redrivePolicy = "{\"deadLetterTargetArn\":\"${var.deadLetterArn}\",\"maxReceiveCount\":${var.maxReceiveCount}}"
}
resource "aws_sqs_queue" "main" {
  name = "${var.project}-${var.environment}-${var.family}-${var.application}-${var.queueName}"

  fifo_queue                 = var.fifo
  delay_seconds              = var.messageDeliveryDelay
  visibility_timeout_seconds = var.visibilityTimeout
  message_retention_seconds  = var.messageRetentionSeconds
  redrive_policy             = var.maxReceiveCount > 0 ? local.redrivePolicy : ""

  tags = {
    Project     = var.project
    Environment = var.environment
    Family      = var.family
    Application = var.application
  }
}

resource "aws_cloudwatch_metric_alarm" "backlog" {
  alarm_name = "${var.family}-${var.application}-${var.queueName}-backlog"
  count      = var.environment == "prod" && var.alarm_backlog_create == 1 ? 1 : 0

  datapoints_to_alarm = var.alarm_backlog_datapoints_to_alarm
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = var.alarm_backlog_evaluation_periods
  threshold           = var.alarm_backlog_threshold
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "visible"
    return_data = false

    metric {
      dimensions = {
        QueueName = "${var.project}-${var.environment}-${var.family}-${var.application}-${var.queueName}"
      }
      metric_name = "ApproximateNumberOfMessagesVisible"
      namespace   = "AWS/SQS"
      period      = 60
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "incoming"
    return_data = false

    metric {
      dimensions = {
        QueueName = "${var.project}-${var.environment}-${var.family}-${var.application}-${var.queueName}"
      }
      metric_name = "NumberOfMessagesSent"
      namespace   = "AWS/SQS"
      period      = 60
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "deleted"
    return_data = false

    metric {
      dimensions = {
        QueueName = "${var.project}-${var.environment}-${var.family}-${var.application}-${var.queueName}"
      }
      metric_name = "NumberOfMessagesDeleted"
      namespace   = "AWS/SQS"
      period      = 60
      stat        = "Sum"
    }
  }

  metric_query {
    expression  = "visible + incoming - (deleted * ${var.alarm_backlog_minutes})"
    id          = "backlog"
    label       = "visible + incoming - (deleted * ${var.alarm_backlog_minutes})"
    return_data = true
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
}