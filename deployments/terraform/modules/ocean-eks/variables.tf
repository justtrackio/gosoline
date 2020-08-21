variable "spotinst_token" {
  type        = string
  description = "Spotinst Personal Access token"
}

variable "spotinst_account" {
  type        = string
  description = "Spotinst account ID"
}

variable "cluster_identifier" {
  type        = string
  description = "Cluster identifier"
  default     = null
}

variable "cluster_name" {
  type        = string
  description = "Cluster name"
}

variable "cluster_version" {
  type        = string
  description = "Kubernetes supported version"
  default     = "1.17"
}

variable "region" {
  type        = string
  description = "The region the EKS cluster will be located"
}

variable "ami_id" {
  type        = string
  description = "The image ID for the EKS worker nodes. If none is provided, Terraform will search for the latest version of their EKS optimized worker AMI based on platform"
  default     = null
}

variable "min_size" {
  type        = number
  description = "The lower limit of worker nodes the Ocean cluster can scale down to"
}

variable "max_size" {
  type        = number
  description = "The upper limit of worker nodes the Ocean cluster can scale up to"
}

variable "key_name" {
  type        = string
  description = "The key pair to attach to the worker nodes launched by Ocean"
  default     = null
}

variable "associate_public_ip_address" {
  type        = bool
  description = "Associate a public IP address to worker nodes"
  default     = false
}

variable "create_vpc" {
  description = "Controls if VPC should be created (it affects almost all resources)"
  type        = bool
  default     = true
}

variable "cidr" {
  description = "The CIDR block for the VPC. Default value is a valid CIDR, but not acceptable by AWS and should be overridden (only needed if new vpc is created)"
  type        = string
  default     = "0.0.0.0/0"
}

variable "private_subnets" {
  description = "A list of private subnets inside the VPC (only needed if new vpc is created)"
  type        = list(string)
  default     = []
}

variable "public_subnets" {
  description = "A list of public subnets inside the VPC (only needed if new vpc is created)"
  type        = list(string)
  default     = []
}

variable "vpc_id" {
  description = "The ID of the VPC (only needed if no vpc is created)"
  type        = string
  default     = null
}

variable "private_subnet_ids" {
  description = "A ID's of private subnets inside the VPC (only needed if no vpc is created)"
  type        = list(string)
  default     = []
}
