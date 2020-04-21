variable "enabled" {
  type        = bool
  default     = true
  description = "Set to false to prevent the module from creating any resources"
}

variable "enabled_global_secondary_index" {
  type        = bool
  default     = false
  description = "Set to false to prevent the module from creating any resources"
}

variable "dynamodb_table_name" {
  type        = string
  description = "DynamoDB table name"
}

variable "dynamodb_table_arn" {
  type        = string
  description = "DynamoDB table ARN"
}

variable "dynamodb_indexes" {
  type        = list(string)
  description = "List of DynamoDB indexes"
  default     = []
}

variable "autoscale_write_target" {
  type        = number
  default     = 75
  description = "The target value for DynamoDB write autoscaling"
}

variable "autoscale_read_target" {
  type        = number
  default     = 75
  description = "The target value for DynamoDB read autoscaling"
}

variable "autoscale_min_read_capacity" {
  type        = number
  default     = 1
  description = "DynamoDB autoscaling min read capacity"
}

variable "autoscale_max_read_capacity" {
  type        = number
  default     = 1000
  description = "DynamoDB autoscaling max read capacity"
}

variable "autoscale_min_write_capacity" {
  type        = number
  default     = 1
  description = "DynamoDB autoscaling min write capacity"
}

variable "autoscale_max_write_capacity" {
  type        = number
  default     = 1000
  description = "DynamoDB autoscaling max write capacity"
}

variable "autoscale_min_read_capacity_global_secondary_index" {
  type        = number
  default     = 1
  description = "DynamoDB autoscaling min read capacity"
}

variable "autoscale_max_read_capacity_global_secondary_index" {
  type        = number
  default     = 1000
  description = "DynamoDB autoscaling max read capacity"
}

variable "autoscale_min_write_capacity_global_secondary_index" {
  type        = number
  default     = 1
  description = "DynamoDB autoscaling min write capacity"
}

variable "autoscale_max_write_capacity_global_secondary_index" {
  type        = number
  default     = 1000
  description = "DynamoDB autoscaling max write capacity"
}