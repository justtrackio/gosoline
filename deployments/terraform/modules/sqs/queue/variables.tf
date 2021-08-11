variable "project" {}
variable "environment" {}
variable "family" {}
variable "application" {}
variable "queueName" {}

variable "fifo" {
  type    = bool
  default = false
}

variable "messageDeliveryDelay" {
  default = 0
}

variable "visibilityTimeout" {
  default = 30
}

variable "messageRetentionSeconds" {
  default = 604800
}

variable "maxReceiveCount" {
  default = 0
}

variable "deadLetterArn" {
  default = ""
}

variable "alarm_backlog_create" {
  default = 1
}

variable "alarm_backlog_datapoints_to_alarm" {
  default = 3
}

variable "alarm_backlog_evaluation_periods" {
  default = 3
}

variable "alarm_backlog_threshold" {
  default = 0
}

variable "alarm_backlog_minutes" {
  default = 3
}

variable "alarm_backlog_period" {
  default = 60
}
