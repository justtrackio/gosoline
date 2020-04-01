module "main" {
  source = "./queue"

  application = var.application
  environment = var.environment
  family      = var.family
  project     = var.project
  queueName   = var.queueName
  fifoQueue   = var.fifoQueue
  maxReceiveCount         = var.maxReceiveCount
  messageDeliveryDelay    = var.messageDeliveryDelay
  deadLetterArn           = module.dead.arn
  visibilityTimeout       = var.visibilityTimeout
  messageRetentionSeconds = var.messageRetentionSeconds

  alarm_create              = var.alarm_main_create
  alarm_period              = var.alarm_main_period
  alarm_threshold           = var.alarm_main_threshold
  alarm_evaluation_periods  = var.alarm_main_evaluation_periods
  alarm_datapoints_to_alarm = var.alarm_main_datapoints_to_alarm
}

module "dead" {
  source = "./queue"

  application = var.application
  environment = var.environment
  family      = var.family
  project     = var.project
  queueName   = "${var.queueName}-dead"
  fifoQueue   = var.fifoQueue
  messageRetentionSeconds = var.messageRetentionSeconds

  alarm_create              = var.alarm_dead_create
  alarm_period              = var.alarm_dead_period
  alarm_threshold           = var.alarm_dead_threshold
  alarm_evaluation_periods  = var.alarm_dead_evaluation_periods
  alarm_datapoints_to_alarm = var.alarm_dead_datapoints_to_alarm
}
