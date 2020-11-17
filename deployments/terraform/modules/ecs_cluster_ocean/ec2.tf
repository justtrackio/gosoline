resource "spotinst_ocean_ecs" "ocean" {
  name         = "${var.project}-${var.environment}-${var.family}"
  cluster_name = "${var.project}-${var.environment}-${var.family}"
  region       = "eu-central-1"
  max_size     = var.max_size
  min_size     = var.min_size
  subnet_ids   = data.aws_subnet_ids.private.ids

  user_data = <<EOF
#!/bin/bash
echo ECS_INSTANCE_ATTRIBUTES='{"lifecycle":"spot"}' >> /etc/ecs/ecs.config
echo ECS_CLUSTER=${var.project}-${var.environment}-${var.family} >> /etc/ecs/ecs.config
echo ECS_AVAILABLE_LOGGING_DRIVERS=[\"none\", \"awslogs\", \"fluentd\"] >> /etc/ecs/ecs.config
EOF

  image_id             = data.aws_ssm_parameter.ami.value
  security_group_ids   = data.aws_security_groups.private.ids
  key_pair             = var.ec2_key_name
  iam_instance_profile = aws_iam_instance_profile.ec2.id
  ebs_optimized        = true
  draining_timeout     = 300

  autoscaler {
    is_enabled     = true
    is_auto_config = true

    headroom {}

    down {
      max_scale_down_percentage = 10
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

  update_policy {
    should_roll = true

    roll_config {
      batch_size_percentage = 33
    }
  }

  scheduled_task {
    tasks {
      is_enabled      = true
      cron_expression = "0 3 * * *"
      task_type       = "clusterRoll"
    }
  }
}

resource "spotinst_ocean_ecs_launch_spec" "ocean" {
  name      = "${var.project}-${var.environment}-${var.family}"
  ocean_id  = spotinst_ocean_ecs.ocean.id
  image_id  = data.aws_ssm_parameter.ami.value
  user_data = <<EOF
#!/bin/bash
echo ECS_INSTANCE_ATTRIBUTES='{"lifecycle":"spot"}' >> /etc/ecs/ecs.config
echo ECS_CLUSTER=${var.project}-${var.environment}-${var.family} >> /etc/ecs/ecs.config
echo ECS_AVAILABLE_LOGGING_DRIVERS=[\"none\", \"awslogs\", \"fluentd\"] >> /etc/ecs/ecs.config
EOF

  iam_instance_profile = aws_iam_instance_profile.ec2.id
  security_group_ids   = data.aws_security_groups.private.ids

  attributes {
    key   = "lifecycle"
    value = "spot"
  }
}

resource "spotinst_elastigroup_aws" "main" {
  count            = var.spotinst_elastigroup
  name             = "${var.project}-${var.environment}-${var.family}"
  region           = "eu-central-1"
  max_size         = var.max_size
  min_size         = var.min_size
  desired_capacity = 1
  subnet_ids       = data.aws_subnet_ids.private.ids

  user_data = <<EOF
#!/bin/bash
echo ECS_INSTANCE_ATTRIBUTES='{"lifecycle":"spot"}' >> /etc/ecs/ecs.config
echo ECS_CLUSTER=${var.project}-${var.environment}-${var.family} >> /etc/ecs/ecs.config
echo ECS_AVAILABLE_LOGGING_DRIVERS=[\"none\", \"awslogs\", \"fluentd\"] >> /etc/ecs/ecs.config
EOF

  image_id             = data.aws_ssm_parameter.ami.value
  security_groups      = data.aws_security_groups.private.ids
  key_name             = var.ec2_key_name
  iam_instance_profile = aws_iam_instance_profile.ec2.id
  draining_timeout     = 300
  ebs_optimized        = true

  product                 = "Linux/UNIX"
  orientation             = "balanced"
  lifetime_period         = "days"
  spot_percentage         = 0
  fallback_to_ondemand    = true
  instance_types_ondemand = var.ec2_instance_type
  instance_types_spot     = [var.ec2_instance_type]

  network_interface {
    device_index                = 0
    associate_public_ip_address = false
    delete_on_termination       = true
  }

  integration_ecs {
    cluster_name             = "${var.project}-${var.environment}-${var.family}"
    autoscale_is_enabled     = true
    autoscale_cooldown       = 300
    autoscale_is_auto_config = true
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

  update_policy {
    should_roll            = false
    should_resume_stateful = false
  }
}
