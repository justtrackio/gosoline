module "label" {
  source      = "applike/label/aws"
  version     = "1.0.2"
  project     = var.project
  application = var.application
  family      = var.family
  environment = var.environment
}

module "metric_label" {
  source    = "applike/label/aws"
  version   = "1.0.2"
  context   = module.label.context
  delimiter = "/"
}

resource "aws_cloudwatch_metric_alarm" "success-rate" {
  alarm_name          = "${module.label.family}-${module.label.application}-success-rate"
  count               = var.create ? 1 : 0
  datapoints_to_alarm = var.datapoints_to_alarm
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.evaluation_periods
  threshold           = var.success_rate_threshold
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "requests"
    return_data = false


    metric {
      metric_name = "RequestCount"
      namespace   = "AWS/ApplicationELB"
      dimensions = {
        "LoadBalancer" = data.aws_lb.default.arn_suffix
      }
      period = var.period
      stat   = "Sum"
    }
  }

  metric_query {
    id          = "errors"
    return_data = false

    metric {
      dimensions = {
        "reason" = "Error"
      }
      metric_name = "error"
      namespace   = module.metric_label.id
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    expression  = "100-100*(errors/requests)"
    id          = "e1"
    label       = "100-100*(errors/requests)"
    return_data = true
  }
}
