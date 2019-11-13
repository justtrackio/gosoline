# Environment variables are composed into the container definition at output generation time. See outputs.tf for more information.
locals {
  container_definition = {
    name             = var.container_name
    image            = var.container_image
    memory           = "memory_sentinel_value"
    cpu              = "cpu_sentinel_value"
    essential        = var.essential
    workingDirectory = var.working_directory
    links            = var.links
    portMappings     = var.port_mappings
    healthCheck      = var.healthcheck
    dockerLabels     = var.dockerLabels

    logConfiguration = {
      logDriver = var.log_driver
      options   = var.log_options
    }

    environment = "environment_sentinel_value"
    secrets     = "secrets_sentinel_value"
  }

  environment = var.environment
  secrets     = var.secrets
}
