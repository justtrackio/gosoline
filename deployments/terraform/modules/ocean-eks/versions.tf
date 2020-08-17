terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    spotinst = {
      source = "terraform-providers/spotinst"
    }
  }
  required_version = ">= 0.13"
}
