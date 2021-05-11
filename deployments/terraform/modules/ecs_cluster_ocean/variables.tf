variable "project" {}
variable "environment" {}
variable "family" {}

variable "ec2_key_name" {
  default = "ecs-temp"
}

variable "ec2_instance_type" {
  default = "m5.large"
}

variable "spotinst_elastigroup" {
  default = 0
}
variable "max_size" {
  default = 1000
}

variable "min_size" {
  default = 1
}

variable "tags" {
  type        = "map"
  default     = {}
  description = "Additional tags for ECS Service"
}
