module "namespace_label" {
  source  = "applike/label/aws"
  version = "1.1.0"

  delimiter = "/"

  context = module.this.context
}

resource "aws_cloudwatch_metric_alarm" "success-rate" {
  for_each            = var.create ? { for consumer in var.consumers : consumer.name => consumer } : {}
  alarm_name          = "${module.this.family}-${module.this.application}-${each.value.name}-success-rate"
  datapoints_to_alarm = var.datapoints_to_alarm
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = var.evaluation_periods
  threshold           = var.success_rate_threshold
  treat_missing_data  = "notBreaching"

  metric_query {
    id          = "messages"
    return_data = false

    metric {
      dimensions = {
        QueueName = each.value.application != null ? "${module.this.project}-${module.this.environment}-${module.this.family}-${each.value.application}-${each.value.queue_id}" : "${module.this.id}-${each.value.queue_id}"
      }
      metric_name = "NumberOfMessagesReceived"
      namespace   = "AWS/SQS"
      period      = var.period
      stat        = "Sum"
    }
  }

  metric_query {
    id          = "errors"
    return_data = false

    metric {
      metric_name = "Error"
      dimensions = {
        Consumer = each.value.name
      }
      namespace = module.namespace_label.id
      period    = var.period
      stat      = "Sum"
    }
  }

  metric_query {
    expression  = "100-100*(errors/messages)"
    id          = "e1"
    label       = "100-100*(errors/messages)"
    return_data = true
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${module.this.project}-${module.this.environment}-${module.this.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${module.this.project}-${module.this.environment}-${module.this.family}-alarm"]

  tags = module.this.tags
}
