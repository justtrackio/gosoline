terraform {
  experiments = [module_variable_optional_attrs]
}

variable "project" {}
variable "environment" {}
variable "family" {}
variable "application" {}

variable "create" {
  type        = bool
  default     = true
  description = "Defines if alarm should be created"
}

variable "datapoints_to_alarm" {
  type        = number
  default     = 3
  description = "The number of datapoints that must be breaching to trigger the alarm"
}

variable "evaluation_periods" {
  type        = number
  default     = 3
  description = "The number of periods over which data is compared to the specified threshold"
}

variable "success_rate_threshold" {
  type        = number
  default     = 99
  description = "Required percentage of successful messages"
}

variable "period" {
  type        = number
  default     = 60
  description = "The period in seconds over which the specified statistic is applied"
}

variable "topics" {
  type = set(object({
    application = optional(string),
    topic_id    = string
  }))
  default     = null
  description = "List of topic ids"
}
