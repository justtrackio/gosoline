variable "project" {}
variable "family" {}
variable "application" {}

variable "primary_tag" {
  type    = string
  default = "master"
}

variable "use_default_lifecycle_policy" {
  default = 1
}
