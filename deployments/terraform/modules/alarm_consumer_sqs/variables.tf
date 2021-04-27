terraform {
  experiments = [module_variable_optional_attrs]
}

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
  description = "Required percentage of successful requests"
}

variable "period" {
  type        = number
  default     = 60
  description = "The period in seconds over which the specified statistic is applied"
}

variable "consumers" {
  type = set(object({
    application = optional(string),
    name        = string,
    queue_id    = string
  }))
  default     = null
  description = "List of consumer/queue_id pairs"
}
