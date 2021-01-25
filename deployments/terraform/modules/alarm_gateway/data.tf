module "lb_label" {
  source      = "applike/label/aws"
  version     = "1.0.2"
  context     = module.label.context
  environment = "pr"
  application = var.application_short
}

data "aws_lb" "default" {
  name = module.lb_label.id
}