provider "aws" {
  region = var.region
}

resource "aws_security_group" "all_worker_mgmt" {
  name   = "${var.cluster_name}_all_worker_management"
  vpc_id = local.vpc_id

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
  name_prefix           = local.cluster_name
  assume_role_policy    = data.aws_iam_policy_document.workers_assume_role_policy.json
  force_detach_policies = true
}

resource "aws_iam_instance_profile" "workers" {
  name_prefix = local.cluster_name
  role        = aws_iam_role.workers.name
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
  name               = local.cluster_name
  azs                = [data.aws_availability_zones.available.names[0], data.aws_availability_zones.available.names[1], data.aws_availability_zones.available.names[2]]
  cidr               = var.cidr
  private_subnets    = var.private_subnets
  public_subnets     = var.public_subnets
  enable_nat_gateway = true
  single_nat_gateway = true
  tags = merge(
    var.tags,
    {
      "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    },
  )
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "13.2.0"

  cluster_version = var.cluster_version
  cluster_name    = local.cluster_name
  vpc_id          = local.vpc_id
  subnets         = local.subnets
  tags            = var.tags
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
