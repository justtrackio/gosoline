module "default_label" {
  source      = "applike/label/aws"
  version     = "1.0.1"
  project     = var.project
  environment = var.environment
}

module "application_label" {
  source      = "applike/label/aws"
  version     = "1.0.1"
  context     = module.default_label.context
  application = var.application
}

provider "aws" {
  region = var.region
}

resource "aws_security_group" "all_worker_mgmt" {
  name   = "${module.application_label.id}-all-worker-management"
  vpc_id = local.vpc_id
  tags   = module.application_label.tags

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"

    cidr_blocks = [
      "10.0.0.0/8",
    ]
  }
}

resource "aws_iam_role" "workers" {
  name                  = module.application_label.id
  assume_role_policy    = data.aws_iam_policy_document.workers_assume_role_policy.json
  force_detach_policies = true
  tags                  = module.application_label.tags
}

resource "aws_iam_instance_profile" "workers" {
  name = module.application_label.id
  role = aws_iam_role.workers.name
}

resource "aws_iam_role_policy_attachment" "workers_AmazonEKSWorkerNodePolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.workers.name
}

resource "aws_iam_role_policy_attachment" "workers_AmazonEKS_CNI_Policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.workers.name
}

resource "aws_iam_role_policy_attachment" "workers_AmazonEC2ContainerRegistryReadOnly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.workers.name
}

resource "aws_iam_role_policy_attachment" "workers_AdministratorAccess" {
  policy_arn = "arn:aws:iam::aws:policy/AdministratorAccess"
  role       = aws_iam_role.workers.name
}

data "aws_region" "current" {}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.64.0"

  create_vpc         = var.vpc_id == null
  name               = module.default_label.id
  azs                = [data.aws_availability_zones.available.names[0], data.aws_availability_zones.available.names[1], data.aws_availability_zones.available.names[2]]
  cidr               = var.cidr
  private_subnets    = var.private_subnets
  public_subnets     = var.public_subnets
  enable_nat_gateway = true
  single_nat_gateway = true
  tags = {
    "Project"                                          = module.default_label.project
    "Environment"                                      = module.default_label.environment
    "kubernetes.io/cluster/${module.default_label.id}" = "shared"
  }
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "12.2.0"

  cluster_version = var.cluster_version
  cluster_name    = module.default_label.id
  vpc_id          = local.vpc_id
  subnets         = local.subnets
  tags            = module.application_label.tags
  map_roles = [
    {
      rolearn  = aws_iam_role.workers.arn
      username = "system:node:{{EC2PrivateDNSName}}"
      groups   = ["system:nodes"]
    },
  ]

  map_users = [
    {
      userarn  = "arn:aws:iam::164105964448:user/marco"
      username = "marco"
      groups   = ["system:masters"]
    },
    {
      userarn  = "arn:aws:iam::164105964448:user/jan"
      username = "jan"
      groups   = ["system:masters"]
    },
  ]

  worker_additional_security_group_ids = [aws_security_group.all_worker_mgmt.id]
}
