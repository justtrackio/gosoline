resource "aws_ecs_cluster" "main" {
  name = "${var.project}-${var.environment}-${var.family}"

  tags = merge({
    Name        = "${var.project}-${var.environment}-${var.family}"
    Environment = var.environment
    Project     = var.project
    Family      = var.family
  }, var.tags)
}
