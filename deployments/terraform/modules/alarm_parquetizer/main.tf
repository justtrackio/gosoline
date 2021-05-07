resource "aws_cloudwatch_metric_alarm" "success-rate" {
  for_each            = var.create ? var.models : []
  alarm_name          = "${var.family}-${var.application}-${each.value}-success-rate"
  datapoints_to_alarm = var.datapoints_to_alarm
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.evaluation_periods
  threshold           = var.success_rate_threshold
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "success"
    return_data = false

    metric {
      metric_name = "EventWriteSuccess"
      dimensions = {
        ModelId = each.value
      }
      namespace = "${var.project}/${var.environment}/${var.family}/${var.application}"
      period    = var.period
      stat      = "Sum"
    }
  }

  metric_query {
    id          = "failure"
    return_data = false

    metric {
      metric_name = "EventWriteFailure"
      dimensions = {
        ModelId = each.value
      }
      namespace = "${var.project}/${var.environment}/${var.family}/${var.application}"
      period    = var.period
      stat      = "Sum"
    }
  }

  metric_query {
    expression  = "100-100*(failure/(failure+success))"
    id          = "e1"
    label       = "100-100*(failure/(failure+success))"
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
