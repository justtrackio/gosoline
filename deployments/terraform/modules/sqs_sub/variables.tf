variable "project" {}
variable "environment" {}
variable "family" {}
variable "application" {}
variable "targetFamily" {}
variable "targetApplication" {}
variable "targetModel" {}

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

variable "alarm_main_create" {
  default = 1
}

variable "alarm_main_period" {
  default = 300
}

variable "alarm_main_threshold" {
  default = 200
}

variable "alarm_main_evaluation_periods" {
  default = 1
}

variable "alarm_main_datapoints_to_alarm" {
  default = 1
}

variable "alarm_dead_create" {
  default = 1
}

variable "alarm_dead_period" {
  default = 300
}

variable "alarm_dead_threshold" {
  default = 200
}

variable "alarm_dead_evaluation_periods" {
  default = 1
}

variable "alarm_dead_datapoints_to_alarm" {
  default = 1
}
