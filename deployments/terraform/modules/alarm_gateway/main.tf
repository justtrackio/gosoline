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

resource "aws_cloudwatch_metric_alarm" "elb-success-rate" {
  alarm_name          = "${module.label.family}-${module.label.application}-elb-success-rate"
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

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}

resource "aws_cloudwatch_metric_alarm" "path-success-rate" {
  for_each            = var.create ? var.paths : []
  alarm_name          = "${module.label.family}-${module.label.application}-${each.value}-success-rate"
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
      namespace   = "${var.project}/${var.environment}/${var.family}/${var.application}"
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
      namespace   = "${var.project}/${var.environment}/${var.family}/${var.application}"
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

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}
