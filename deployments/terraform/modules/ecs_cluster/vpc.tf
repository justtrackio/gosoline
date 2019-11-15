data "aws_vpc" "main" {
  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

data "aws_subnet_ids" "private" {
  vpc_id = data.aws_vpc.main.id

  tags = {
    Project     = var.project
    Environment = var.environment
    Tier        = "private"
  }
}

data "aws_subnet_ids" "public" {
  vpc_id = data.aws_vpc.main.id

  tags = {
    Project     = var.project
    Environment = var.environment
    Tier        = "public"
  }
}

data "aws_security_groups" "public" {
  tags = {
    Project     = var.project
    Environment = var.environment
    Name        = "public"
  }
}

data "aws_security_groups" "private" {
  tags = {
    Project     = var.project
    Environment = var.environment
    Name        = "private"
  }
}

data "aws_security_groups" "rds" {
  tags = {
    Project     = var.project
    Environment = var.environment
    Name        = "rds"
  }
}
