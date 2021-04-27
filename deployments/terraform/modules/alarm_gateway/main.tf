module "metric_label" {
  source  = "applike/label/aws"
  version = "1.1.0"

  delimiter = "/"

  context = module.this.context
}

resource "aws_cloudwatch_metric_alarm" "elb-success-rate" {
  alarm_name          = "${module.this.family}-${module.this.application}-elb-success-rate"
  count               = var.create ? 1 : 0
  datapoints_to_alarm = var.elb_datapoints_to_alarm
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.elb_evaluation_periods
  threshold           = var.elb_success_rate_threshold
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
      period = var.elb_period
      stat   = "Sum"
    }
  }

  metric_query {
    id          = "errors"
    return_data = false

    metric {
      metric_name = "error"
      namespace   = module.metric_label.id
      period      = var.elb_period
      stat        = "Sum"
    }
  }

  metric_query {
    expression  = "100-100*(errors/requests)"
    id          = "e1"
    label       = "100-100*(errors/requests)"
    return_data = true
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]

  tags = module.this.tags
}

resource "aws_cloudwatch_metric_alarm" "path-success-rate" {
  for_each            = var.create ? var.paths : []
  alarm_name          = "${module.this.family}-${module.this.application}-${each.value}-success-rate"
  datapoints_to_alarm = var.path_datapoints_to_alarm
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.path_evaluation_periods
  threshold           = var.path_success_rate_threshold
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "requests"
    return_data = false

    metric {
      metric_name = "ApiRequestCount"
      namespace   = module.metric_label.id
      dimensions = {
        "path" = each.value
      }
      period = var.path_period
      stat   = "Sum"
    }
  }

  metric_query {
    id          = "errors"
    return_data = false

    metric {
      metric_name = "ApiStatus5XX"
      namespace   = module.metric_label.id
      dimensions = {
        "path" = each.value
      }
      period = var.path_period
      stat   = "Sum"
    }
  }

  metric_query {
    expression  = "100-100*(errors/requests)"
    id          = "e1"
    label       = "100-100*(errors/requests)"
    return_data = true
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]

  tags = module.this.tags
}
