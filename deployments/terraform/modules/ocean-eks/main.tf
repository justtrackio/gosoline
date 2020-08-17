provider "aws" {
  region = var.region
}

provider "spotinst" {
  token   = var.spotinst_token
  account = var.spotinst_account
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  token                  = data.aws_eks_cluster_auth.cluster.token
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  load_config_file       = false
}

locals {
  cluster_name = var.cluster_name
  tags         = {}
}

resource "aws_security_group" "all_worker_mgmt" {
  name   = "${var.cluster_name}_all_worker_management"
  vpc_id = var.create_vpc == true ? module.vpc.vpc_id : var.vpc_id

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

resource "spotinst_ocean_aws" "this" {
  depends_on = [module.eks]

  name                        = var.cluster_name
  controller_id               = var.cluster_identifier != null ? var.cluster_identifier : module.eks.cluster_id
  region                      = var.region
  max_size                    = var.max_size
  min_size                    = var.min_size
  desired_capacity            = var.desired_capacity
  subnet_ids                  = var.create_vpc == true ? module.vpc.private_subnets : var.private_subnet_ids
  image_id                    = var.ami_id != null ? var.ami_id : module.eks.workers_default_ami_id
  security_groups             = [aws_security_group.all_worker_mgmt.id, module.eks.worker_security_group_id]
  key_name                    = var.key_name
  associate_public_ip_address = var.associate_public_ip_address
  iam_instance_profile        = aws_iam_instance_profile.workers.arn

  user_data = <<-EOF
    #!/bin/bash
    set -o xtrace
    /etc/eks/bootstrap.sh ${local.cluster_name}
EOF

  tags {
    key   = "Name"
    value = "${local.cluster_name}-node"
  }
  tags {
    key   = "kubernetes.io/cluster/${local.cluster_name}"
    value = "owned"
  }
  tags {
    key   = "create-alarms"
    value = "false"
  }

  autoscaler {
    autoscale_is_enabled     = true
    autoscale_is_auto_config = true
  }
}

module "vpc" {
  source     = "terraform-aws-modules/vpc/aws"
  version    = "2.47.0"
  create_vpc = var.create_vpc

  name               = local.cluster_name
  cidr               = var.cidr
  azs                = [data.aws_availability_zones.available.names[0], data.aws_availability_zones.available.names[1], data.aws_availability_zones.available.names[2]]
  private_subnets    = var.private_subnets
  public_subnets     = var.public_subnets
  enable_nat_gateway = true
  single_nat_gateway = true
  tags = merge(
    local.tags,
    {
      "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    },
  )
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "12.2.0"

  cluster_name    = local.cluster_name
  cluster_version = var.cluster_version
  subnets         = var.create_vpc == true ? module.vpc.private_subnets : var.private_subnet_ids
  tags            = local.tags
  vpc_id          = var.create_vpc == true ? module.vpc.vpc_id : var.vpc_id
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
  ]

  worker_additional_security_group_ids = [aws_security_group.all_worker_mgmt.id]
}

module "ocean-controller" {
  depends_on         = [module.eks, spotinst_ocean_aws.this]
  source             = "spotinst/ocean-controller/spotinst"
  version            = "0.5.0"
  cluster_identifier = spotinst_ocean_aws.this.controller_id
  spotinst_account   = var.spotinst_account
  spotinst_token     = var.spotinst_token
}
