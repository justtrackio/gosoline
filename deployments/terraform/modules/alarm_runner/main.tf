resource "aws_cloudwatch_metric_alarm" "success-rate" {
  for_each            = var.create
  alarm_name          = "${var.family}-${var.application}-${var.runner_name}-success-rate"
  datapoints_to_alarm = var.datapoints_to_alarm
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.evaluation_periods
  threshold           = var.success_rate_threshold
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "read"
    return_data = false

    metric {
      dimensions = {
        Operation = "Read"
      }
      metric_name = "BlobBatchRunner"
      namespace   = "${var.project}/${var.environment}/${var.family}/${var.application}"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "write"
    return_data = false

    metric {
      dimensions = {
        Operation = "Write"
      }
      metric_name = "BlobBatchRunner"
      namespace   = "${var.project}/${var.environment}/${var.family}/${var.application}"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "copy"
    return_data = false

    metric {
      dimensions = {
        Operation = "Copy"
      }
      metric_name = "BlobBatchRunner"
      namespace   = "${var.project}/${var.environment}/${var.family}/${var.application}"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "delete"
    return_data = false

    metric {
      dimensions = {
        Operation = "Delete"
      }
      metric_name = "BlobBatchRunner"
      namespace   = "${var.project}/${var.environment}/${var.family}/${var.application}"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "errors"
    return_data = false

    metric {
      dimensions = {
        Operation = "Error"
      }
      metric_name = "BlobBatchRunner"
      namespace   = "${var.project}/${var.environment}/${var.family}/${var.application}"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    expression  = "100-100*(errors/(read+write+copy+delete))"
    id          = "e1"
    label       = "100-100*(errors/(read+write+copy+delete))"
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
