provider "spotinst" {
  token   = var.spotinst_token
  account = var.spotinst_account
}

provider "kubernetes" {
  load_config_file       = false
  host                   = data.aws_eks_cluster.cluster.endpoint
  token                  = data.aws_eks_cluster_auth.cluster.token
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
}

resource "spotinst_ocean_aws" "this" {
  depends_on = [module.eks]

  name                        = local.cluster_name
  controller_id               = local.cluster_identifier
  region                      = data.aws_region.current.id
  max_size                    = var.max_size
  min_size                    = var.min_size
  subnet_ids                  = local.subnets
  image_id                    = local.ami_id
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

module "ocean-controller" {
  source            = "spotinst/ocean-controller/spotinst"
  version           = ">=0.9.0"
  module_depends_on = [module.eks] # maintains backward compatibility with terraform v0.12

  # Credentials.
  spotinst_token   = var.spotinst_token
  spotinst_account = var.spotinst_account

  # Configuration.
  cluster_identifier = spotinst_ocean_aws.this.controller_id
}
