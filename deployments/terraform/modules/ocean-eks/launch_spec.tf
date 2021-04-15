data "template_file" "default" {
  template = file("${path.module}/scripts/bootstrap.sh")
  vars = {
    default_label = module.default_label.id
    nvme          = file("${path.module}/scripts/nvme.sh")
  }
}

resource "spotinst_ocean_aws_launch_spec" "default" {
  name                 = "nvme"
  ocean_id             = spotinst_ocean_aws.this.id
  image_id             = local.ami_id
  iam_instance_profile = aws_iam_instance_profile.workers.arn
  security_groups      = [aws_security_group.all_worker_mgmt.id, module.eks.worker_security_group_id]
  subnet_ids           = local.subnets
  instance_types       = ["i3.large", "i3.xlarge", "i3.2xlarge"]

  user_data = data.template_file.default.rendered

  labels {
    key   = "spotinst.io/virtual-node-group"
    value = "nvme"
  }

  tags {
    key   = "Name"
    value = "${module.application_label.id}-node"
  }

  tags {
    key   = "Project"
    value = module.application_label.project
  }

  tags {
    key   = "Environment"
    value = module.application_label.environment
  }

  tags {
    key   = "Application"
    value = module.application_label.application
  }

  tags {
    key   = "kubernetes.io/cluster/${module.default_label.id}"
    value = "owned"
  }
}