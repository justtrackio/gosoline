locals {
  queueName           = var.fifo ? "${var.queueName}.fifo" : var.queueName
  deadLetterQueueName = var.fifo ? "${var.queueName}-dead.fifo" : "${var.queueName}-dead"
}

module "main" {
  source = "./queue"

  application = var.application
  environment = var.environment
  family      = var.family
  project     = var.project
  queueName   = local.queueName
  fifo        = var.fifo

  maxReceiveCount         = var.maxReceiveCount
  messageDeliveryDelay    = var.messageDeliveryDelay
  deadLetterArn           = module.dead.arn
  visibilityTimeout       = var.visibilityTimeout
  messageRetentionSeconds = var.messageRetentionSeconds

  alarm_backlog_create              = var.alarm_main_backlog_create
  alarm_backlog_minutes             = var.alarm_main_backlog_minutes
  alarm_backlog_period              = var.alarm_main_backlog_period
  alarm_backlog_evaluation_periods  = var.alarm_main_backlog_evaluation_periods
  alarm_backlog_datapoints_to_alarm = var.alarm_main_backlog_datapoints_to_alarm
  alarm_backlog_threshold           = var.alarm_main_backlog_treshold
}

module "dead" {
  source = "./queue"

  application = var.application
  environment = var.environment
  family      = var.family
  project     = var.project
  queueName   = local.deadLetterQueueName
  fifo        = var.fifo

  messageRetentionSeconds = var.messageRetentionSeconds
  alarm_backlog_create    = var.alarm_dead_backlog_create
}
