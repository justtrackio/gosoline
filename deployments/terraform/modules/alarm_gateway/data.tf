module "lb_label" {
  source  = "applike/label/aws"
  version = "1.1.0"

  environment = var.environment_short
  application = var.application_short

  context = module.this.context
}

data "aws_lb" "default" {
  name = module.lb_label.id
}