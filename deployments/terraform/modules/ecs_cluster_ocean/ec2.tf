resource "spotinst_ocean_ecs" "ocean" {
  count        = var.spotinst_ocean
  name         = "${var.project}-${var.environment}-${var.family}"
  cluster_name = "${var.project}-${var.environment}-${var.family}"
  region       = "eu-central-1"
  max_size     = 1000
  min_size     = 0
  subnet_ids   = data.aws_subnet_ids.private.ids

  user_data = <<EOF
#!/bin/bash
echo ECS_INSTANCE_ATTRIBUTES='{"lifecycle":"spot"}' >> /etc/ecs/ecs.config
echo ECS_CLUSTER=${var.project}-${var.environment}-${var.family} >> /etc/ecs/ecs.config
echo ECS_ENGINE_TASK_CLEANUP_WAIT_DURATION=5m >> /etc/ecs/ecs.config
echo ECS_NUM_IMAGES_DELETE_PER_CYCLE=50 >> /etc/ecs/ecs.config
echo ECS_IMAGE_CLEANUP_INTERVAL=5m >> /etc/ecs/ecs.config
echo ECS_UPDATES_ENABLED=true >> /etc/ecs/ecs.config
echo ECS_RESERVED_MEMORY=1024 >> /etc/ecs/ecs.config
echo ECS_AVAILABLE_LOGGING_DRIVERS=[\"json-file\", \"none\", \"gelf\", \"awslogs\"] >> /etc/ecs/ecs.config
echo ECS_ENABLE_CONTAINER_METADATA=true >> /etc/ecs/ecs.config
cloud-init-per once docker_options echo 'OPTIONS="$${OPTIONS} --storage-opt dm.basesize=${var.ec2_block_device_size - 2}G"' >> /etc/sysconfig/docker

EOF

  image_id                    = var.ec2_ami
  security_group_ids          = data.aws_security_groups.private.ids
  key_pair                    = var.ec2_key_name
  iam_instance_profile        = aws_iam_instance_profile.ec2.id
  associate_public_ip_address = false
  draining_timeout            = 120
  monitoring                  = false
  ebs_optimized               = true

  autoscaler {
    is_enabled     = true
    is_auto_config = true

    down {
      max_scale_down_percentage = 25
    }

    resource_limits {
      max_vcpu       = 20000
      max_memory_gib = 100000
    }
  }

  tags {
    key   = "Component"
    value = "${var.family}_cluster"
  }

  tags {
    key   = "Name"
    value = "${var.project}-${var.environment}-${var.family}-ecs"
  }

  tags {
    key   = "Environment"
    value = var.environment
  }

  tags {
    key   = "Project"
    value = var.project
  }

  tags {
    key   = "create-alarms"
    value = "false"
  }

  update_policy {
    should_roll = true

    roll_config {
      batch_size_percentage = 25
    }
  }
}

resource "spotinst_elastigroup_aws" "main" {
  count            = var.spotinst_elastigroup
  name             = "${var.project}-${var.environment}-${var.family}"
  region           = "eu-central-1"
  max_size         = 1000
  min_size         = 1
  desired_capacity = 1
  subnet_ids       = data.aws_subnet_ids.private.ids

  user_data = <<EOF
#!/bin/bash
echo ECS_INSTANCE_ATTRIBUTES='{"lifecycle":"od"}' >> /etc/ecs/ecs.config
echo ECS_CLUSTER=${var.project}-${var.environment}-${var.family} >> /etc/ecs/ecs.config
echo ECS_ENGINE_TASK_CLEANUP_WAIT_DURATION=5m >> /etc/ecs/ecs.config
echo ECS_NUM_IMAGES_DELETE_PER_CYCLE=50 >> /etc/ecs/ecs.config
echo ECS_IMAGE_CLEANUP_INTERVAL=5m >> /etc/ecs/ecs.config
echo ECS_UPDATES_ENABLED=true >> /etc/ecs/ecs.config
echo ECS_RESERVED_MEMORY=1024 >> /etc/ecs/ecs.config
echo ECS_AVAILABLE_LOGGING_DRIVERS=[\"json-file\", \"none\", \"gelf\", \"awslogs\"] >> /etc/ecs/ecs.config
echo ECS_ENABLE_CONTAINER_METADATA=true >> /etc/ecs/ecs.config
cloud-init-per once docker_options echo 'OPTIONS="$${OPTIONS} --storage-opt dm.basesize=${var.ec2_block_device_size - 2}G"' >> /etc/sysconfig/docker

EOF

  image_id             = var.ec2_ami
  security_groups      = data.aws_security_groups.private.ids
  key_name             = var.ec2_key_name
  iam_instance_profile = aws_iam_instance_profile.ec2.id
  draining_timeout     = 120
  enable_monitoring    = false
  ebs_optimized        = true

  product                 = "Linux/UNIX"
  orientation             = "balanced"
  lifetime_period         = "days"
  spot_percentage         = 0
  fallback_to_ondemand    = true
  instance_types_ondemand = "m5.large"
  instance_types_spot     = ["m5.large"]

  network_interface {
    device_index                = 0
    associate_public_ip_address = false
    delete_on_termination       = true
  }

  integration_ecs {
    cluster_name         = "${var.project}-${var.environment}-${var.family}"
    autoscale_is_enabled = true
  }

  tags {
    key   = "Component"
    value = "${var.family}_cluster"
  }

  tags {
    key   = "Name"
    value = "${var.project}-${var.environment}-${var.family}-ecs"
  }

  tags {
    key   = "Environment"
    value = var.environment
  }

  tags {
    key   = "Project"
    value = var.project
  }

  tags {
    key   = "create-alarms"
    value = "false"
  }

  update_policy {
    should_roll            = true
    should_resume_stateful = false

    roll_config {
      batch_size_percentage = 25
    }
  }
}