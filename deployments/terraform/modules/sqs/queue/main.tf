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

resource "aws_cloudwatch_metric_alarm" "number-of-visible-messages" {
  alarm_name = "${var.family}-${var.application}-${var.queueName}-number-of-visible-messages"
  count      = var.environment == "prod" ? 1 : 0

  namespace   = "AWS/SQS"
  metric_name = "ApproximateNumberOfMessagesVisible"

  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  period              = var.alarm_period
  statistic           = "Maximum"
  threshold           = var.alarm_visible_messages_threshold
  treat_missing_data  = "notBreaching"
  datapoints_to_alarm = var.alarm_datapoints_to_alarm

  dimensions = {
    QueueName = aws_sqs_queue.main.name
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
}

resource "aws_cloudwatch_metric_alarm" "backlog" {
  alarm_name = "${var.family}-${var.application}-${var.queueName}-backlog"
  count      = var.environment == "prod" ? 1 : 0

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
}