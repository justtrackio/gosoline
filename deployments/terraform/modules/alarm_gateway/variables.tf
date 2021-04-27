variable "environment_short" {
  type        = string
  default     = ""
  description = "Environment, e.g. 'uw2', 'us-west-2', OR 'prod', 'staging', 'dev', 'UAT' used for loadbalancers"
}

variable "application_short" {
  type        = string
  default     = ""
  description = "Solution application, e.g. 'app' or 'jenkins'"
}

variable "create" {
  type        = bool
  default     = true
  description = "Defines if alarm should be created"
}

variable "elb_datapoints_to_alarm" {
  type        = number
  default     = 3
  description = "The number of datapoints that must be breaching to trigger the alarm"
}

variable "elb_evaluation_periods" {
  type        = number
  default     = 3
  description = "The number of periods over which data is compared to the specified threshold"
}

variable "elb_period" {
  type        = number
  default     = 300
  description = "The period in seconds over which the specified statistic is applied"
}

variable "elb_success_rate_threshold" {
  type        = number
  default     = 99
  description = "Required percentage of successful requests"
}

variable "paths" {
  type        = set(string)
  default     = []
  description = "List of paths for which success rate alarm should be created"
}

variable "path_datapoints_to_alarm" {
  type        = number
  default     = 3
  description = "The number of datapoints that must be breaching to trigger the path oriented alarm"
}

variable "path_evaluation_periods" {
  type        = number
  default     = 3
  description = "The number of periods over which data is compared to the specified threshold"
}

variable "path_period" {
  type        = number
  default     = 300
  description = "The period in seconds over which the specified statistic is applied"
}

variable "path_success_rate_threshold" {
  type        = number
  default     = 99
  description = "Required percentage of successful requests"
}