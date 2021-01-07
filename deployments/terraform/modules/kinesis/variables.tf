variable "project" {}
variable "environment" {}
variable "family" {}
variable "application" {}

variable "model" {}

variable "alarm_create" {}
variable "alarm_period_seconds" {}
variable "alarm_limit_threshold_percentage" {}
variable "alarm_iterator_threshold_age_milliseconds" {
  default = 60000
}

variable "shard_count" {
  default = 1
}

