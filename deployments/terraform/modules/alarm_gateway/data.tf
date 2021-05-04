module "lb_label" {
  source      = "applike/label/aws"
  version     = "1.0.2"
  context     = module.label.context
  environment = var.environment_short
  application = var.application_short
  family      = length(var.family_short) != 0 ? var.family_short : var.family
}

data "aws_lb" "default" {
  name = module.lb_label.id
}