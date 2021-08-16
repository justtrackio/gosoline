locals {
  model = var.model != "" ? var.model : var.targetModel
}

data "aws_sns_topic" "main" {
  name = "${var.project}-${var.environment}-${var.targetFamily}-${var.targetApplication}-${var.targetModel}"
}

module "queue" {
  source = "./../sqs"

  project     = var.project
  environment = var.environment
  family      = var.family
  application = var.application
  queueName   = local.model

  messageDeliveryDelay    = var.messageDeliveryDelay
  visibilityTimeout       = var.visibilityTimeout
  messageRetentionSeconds = var.messageRetentionSeconds
  maxReceiveCount         = var.maxReceiveCount

  alarm_main_backlog_minutes             = var.alarm_main_backlog_minutes
  alarm_main_backlog_period              = var.alarm_main_backlog_period
  alarm_main_backlog_create              = var.alarm_main_backlog_create
  alarm_main_backlog_evaluation_periods  = var.alarm_main_backlog_evaluation_periods
  alarm_main_backlog_datapoints_to_alarm = var.alarm_main_backlog_datapoints_to_alarm
  alarm_main_backlog_treshold            = var.alarm_main_backlog_treshold

  alarm_dead_backlog_create = var.alarm_dead_backlog_create
}

resource "aws_sns_topic_subscription" "main" {
  topic_arn                       = data.aws_sns_topic.main.arn
  confirmation_timeout_in_minutes = "1"
  endpoint_auto_confirms          = "false"
  protocol                        = "sqs"
  endpoint                        = module.queue.queue_arn
  filter_policy                   = var.filterPolicy
}

resource "aws_sqs_queue_policy" "main" {
  queue_url = module.queue.queue_id

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "MyQueuePolicy",
  "Statement": [{
     "Sid":"MySQSPolicy001",
     "Effect":"Allow",
     "Principal":"*",
     "Action":"sqs:SendMessage",
     "Resource":"${module.queue.queue_arn}",
     "Condition":{
       "ArnEquals":{
         "aws:SourceArn":"${data.aws_sns_topic.main.arn}"
       }
     }
  }]
}
POLICY
}
