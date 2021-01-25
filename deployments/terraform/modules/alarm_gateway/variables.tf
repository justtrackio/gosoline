variable "project" {
  type        = string
  default     = ""
  description = "Project, which could be your organization name or abbreviation, e.g. 'eg' or 'cp'"
}

variable "environment" {
  type        = string
  default     = ""
  description = "Environment, e.g. 'uw2', 'us-west-2', OR 'prod', 'staging', 'dev', 'UAT'"
}

variable "family" {
  type        = string
  default     = ""
  description = "Family, e.g. 'prod', 'staging', 'dev', OR 'source', 'build', 'test', 'deploy', 'release'"
}

variable "application" {
  type        = string
  default     = ""
  description = "Solution application, e.g. 'app' or 'jenkins'"
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
  default     = 95
  description = "Required percentage of successful requests"
}

variable "period" {
  type        = number
  default     = 300
  description = "The period in seconds over which the specified statistic is applied"
}
