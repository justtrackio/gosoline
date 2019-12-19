module "main" {
  source = "./queue"

  application = var.application
  environment = var.environment
  family      = var.family
  project     = var.project
  queueName   = var.queueName

  maxReceiveCount         = var.maxReceiveCount
  messageDeliveryDelay    = var.messageDeliveryDelay
  deadLetterArn           = module.dead.arn
  visibilityTimeout       = var.visibilityTimeout
  messageRetentionSeconds = var.messageRetentionSeconds

  alarm_create        = var.alarm_main_create
  alarm_period        = var.alarm_main_period
  alarm_threshold     = var.alarm_main_threshold
  evaluation_periods  = var.alarm_main_evaluation_periods
  datapoints_to_alarm = var.alarm_main_datapoints_to_alarm
}

module "dead" {
  source = "./queue"

  application = var.application
  environment = var.environment
  family      = var.family
  project     = var.project
  queueName   = "${var.queueName}-dead"

  messageRetentionSeconds = var.messageRetentionSeconds

  alarm_create        = var.alarm_dead_create
  alarm_period        = var.alarm_dead_period
  alarm_threshold     = var.alarm_dead_threshold
  evaluation_periods  = var.alarm_dead_evaluation_periods
  datapoints_to_alarm = var.alarm_dead_datapoints_to_alarm
}
