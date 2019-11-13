variable "project" {}
variable "environment" {}
variable "family" {}
variable "application" {}
variable "queueName" {}

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

variable "alarm_create" {
  default = 1
}

variable "alarm_period" {
  default = 300
}

variable "alarm_threshold" {
  default = 200
}
