data "aws_sns_topic" "main" {
  name = "${var.project}-${var.environment}-${var.targetFamily}-${var.targetApplication}-${var.targetModel}"
}

module "queue" {
  source = "../sqs"

  project     = var.project
  environment = var.environment
  family      = var.family
  application = var.application
  queueName   = var.targetModel

  messageDeliveryDelay    = var.messageDeliveryDelay
  visibilityTimeout       = var.visibilityTimeout
  messageRetentionSeconds = var.messageRetentionSeconds
  maxReceiveCount         = var.maxReceiveCount

  alarm_main_create    = var.alarm_main_create
  alarm_main_period    = var.alarm_main_period
  alarm_main_threshold = var.alarm_main_threshold

  alarm_dead_create    = var.alarm_dead_create
  alarm_dead_period    = var.alarm_dead_period
  alarm_dead_threshold = var.alarm_dead_threshold
}

resource "aws_sns_topic_subscription" "main" {
  topic_arn                       = data.aws_sns_topic.main.arn
  confirmation_timeout_in_minutes = "1"
  endpoint_auto_confirms          = "false"
  protocol                        = "sqs"
  endpoint                        = module.queue.queue_arn
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
