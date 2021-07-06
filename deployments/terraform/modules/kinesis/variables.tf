variable "project" {}
variable "environment" {}
variable "family" {}
variable "application" {}

variable "model" {}

variable "alarm_create" {}
variable "alarm_period_seconds" {
  type    = number
  default = 900
}
variable "alarm_evaluation_periods" {
  type    = number
  default = 6
}
variable "alarm_datapoints_to_alarm" {
  type    = number
  default = 4
}
variable "alarm_put_records_datapoints_to_alarm" {
  type    = number
  default = 6
}
variable "alarm_limit_threshold_percentage" {
  type    = number
  default = 80
}
variable "alarm_records_success_threshold" {
  type    = number
  default = 0.99
}
variable "alarm_iterator_threshold_age_milliseconds" {
  type    = number
  default = 60000
}

variable "shard_count" {
  type    = number
  default = 1
}

