resource "aws_cloudwatch_metric_alarm" "sns-success-rate" {
  for_each            = var.create ? { for topic in var.topics : topic.topic_id => topic } : {}
  alarm_name          = "${var.family}-${var.application}-${each.value.topic_id}-sns-success-rate"
  datapoints_to_alarm = var.datapoints_to_alarm
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.evaluation_periods
  threshold           = var.success_rate_threshold
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "published"
    return_data = false

    metric {
      dimensions = {
        TopicName = each.value.application != null ? "${var.project}-${var.environment}-${var.family}-${each.value.application}-${each.value.topic_id}" : "${var.project}-${var.environment}-${var.family}-${var.application}-${each.value.topic_id}"
      }
      metric_name = "NumberOfMessagesPublished"
      namespace   = "AWS/SNS"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "failed"
    return_data = false

    metric {
      dimensions = {
        TopicName = each.value.application != null ? "${var.project}-${var.environment}-${var.family}-${each.value.application}-${each.value.topic_id}" : "${var.project}-${var.environment}-${var.family}-${var.application}-${each.value.topic_id}"
      }
      metric_name = "NumberOfNotificationsFailed"
      namespace   = "AWS/SNS"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    expression  = "100-100*(failed/published)"
    id          = "e1"
    label       = "100-100*(failed/published)"
    return_data = true
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}
