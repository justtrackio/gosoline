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

  alarm_create    = var.alarm_main_create
  alarm_period    = var.alarm_main_period
  alarm_threshold = var.alarm_main_threshold
}

module "dead" {
  source = "./queue"

  application = var.application
  environment = var.environment
  family      = var.family
  project     = var.project
  queueName   = "${var.queueName}-dead"

  messageRetentionSeconds = var.messageRetentionSeconds

  alarm_create    = var.alarm_dead_create
  alarm_period    = var.alarm_dead_period
  alarm_threshold = var.alarm_dead_threshold
}
