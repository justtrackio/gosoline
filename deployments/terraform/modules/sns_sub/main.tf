locals {
  model = var.model != "" ? var.model : var.targetModel
}

module "sns" {
  source = "./sns"

  project     = var.project
  environment = var.environment
  family      = var.targetFamily
  application = var.targetApplication
  topicName   = local.model
}

module "sqs_sub" {
  source = "./../sqs_sub"

  project           = var.project
  environment       = var.environment
  family            = var.family
  application       = var.application
  targetFamily      = var.targetFamily
  targetApplication = var.targetApplication
  targetModel       = local.model

  messageDeliveryDelay    = var.messageDeliveryDelay
  visibilityTimeout       = var.visibilityTimeout
  messageRetentionSeconds = var.messageRetentionSeconds
  maxReceiveCount         = var.maxReceiveCount

  alarm_main_create              = var.alarm_main_create
  alarm_main_period              = var.alarm_main_period
  alarm_main_threshold           = var.alarm_main_threshold
  alarm_main_evaluation_periods  = var.alarm_main_evaluation_periods
  alarm_main_datapoints_to_alarm = var.alarm_main_datapoints_to_alarm

  alarm_dead_create              = var.alarm_dead_create
  alarm_dead_period              = var.alarm_dead_period
  alarm_dead_threshold           = var.alarm_dead_threshold
  alarm_dead_evaluation_periods  = var.alarm_dead_evaluation_periods
  alarm_dead_datapoints_to_alarm = var.alarm_dead_datapoints_to_alarm
}
