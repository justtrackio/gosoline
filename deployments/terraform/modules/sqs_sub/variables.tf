variable "project" {}
variable "environment" {}
variable "family" {}
variable "application" {}
variable "model" {
  default = ""
}
variable "targetFamily" {}
variable "targetApplication" {}
variable "targetModel" {}

variable "filterPolicy" {
  default = ""
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

variable "alarm_main_backlog_evaluation_periods" {
  default = 3
}

variable "alarm_main_backlog_datapoints_to_alarm" {
  default = 3
}

variable "alarm_main_backlog_treshold" {
  default = 0
}

variable "alarm_main_backlog_minutes" {
  default = 3
}

variable "alarm_main_backlog_period" {
  default = 60
}

variable "alarm_main_backlog_create" {
  default = 1
}

variable "alarm_dead_backlog_create" {
  default = 0
}
