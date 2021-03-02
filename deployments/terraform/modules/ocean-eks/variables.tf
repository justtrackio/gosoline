variable "project" {
  type        = string
  default     = null
  description = "Project, which could be your organization name or abbreviation, e.g. 'eg' or 'cp'"
}

variable "environment" {
  type        = string
  default     = null
  description = "Environment, e.g. 'uw2', 'us-west-2', OR 'prod', 'staging', 'dev', 'UAT'"
}

variable "application" {
  type        = string
  default     = null
  description = "Solution application, e.g. 'app' or 'jenkins'"
}

variable "spotinst_token" {
  type        = string
  description = "Spot Personal Access token"
}

variable "spotinst_account" {
  type        = string
  description = "Spot account ID"
}

variable "region" {
  type        = string
  description = "The region the EKS cluster will be located"
  default     = "eu-central-1"
}

variable "cluster_identifier" {
  type        = string
  description = "Cluster identifier"
  default     = null
}

variable "cluster_version" {
  type        = string
  description = "Kubernetes supported version"
  default     = "1.18"
}

variable "subnets" {
  type        = list(string)
  description = "A list of subnets to place the EKS cluster and workers within"
  default     = null
}

variable "vpc_id" {
  type        = string
  description = "VPC where the cluster and workers will be deployed"
  default     = null
}

variable "ami_id" {
  type        = string
  description = "The image ID for the EKS worker nodes. If none is provided, Terraform will search for the latest version of their EKS optimized worker AMI based on platform"
  default     = null
}

variable "min_size" {
  type        = number
  description = "The lower limit of worker nodes the Ocean cluster can scale down to"
  default     = 1
}

variable "max_size" {
  type        = number
  description = "The upper limit of worker nodes the Ocean cluster can scale up to"
  default     = 1000
}

variable "key_name" {
  type        = string
  description = "The key pair to attach to the worker nodes launched by Ocean"
  default     = "admin"
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

variable "private_subnet_ids" {
  description = "A ID's of private subnets inside the VPC (only needed if no vpc is created)"
  type        = list(string)
  default     = []
}
