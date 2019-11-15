resource "spotinst_ocean_ecs" "ocean" {
  count        = var.spotinst_ocean_ecs
  name         = "${var.project}-${var.environment}-${var.family}"
  cluster_name = "${var.project}-${var.environment}-${var.family}"
  region       = "eu-central-1"
  max_size     = 1000
  min_size     = 0
  subnet_ids   = data.aws_subnet_ids.private.ids

  user_data = <<EOF
#!/bin/bash
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

resource "aws_instance" "ecs_instance" {
  ami                         = var.ec2_ami
  count                       = var.ec2_count
  iam_instance_profile        = aws_iam_instance_profile.ec2.id
  instance_type               = var.ec2_instance_type
  key_name                    = var.ec2_key_name
  subnet_id                   = reverse(tolist(data.aws_subnet_ids.private.ids))[count.index % 2]
  vpc_security_group_ids      = data.aws_security_groups.private.ids
  associate_public_ip_address = false

  user_data = <<EOF
#!/bin/bash
echo ECS_CLUSTER=${var.project}-${var.environment}-${var.family} >> /etc/ecs/ecs.config
echo ECS_ENGINE_TASK_CLEANUP_WAIT_DURATION=10m >> /etc/ecs/ecs.config
echo ECS_NUM_IMAGES_DELETE_PER_CYCLE=50 >> /etc/ecs/ecs.config
echo ECS_IMAGE_CLEANUP_INTERVAL=5m >> /etc/ecs/ecs.config
echo ECS_UPDATES_ENABLED=true >> /etc/ecs/ecs.config
echo ECS_RESERVED_MEMORY=768 >> /etc/ecs/ecs.config
echo ECS_AVAILABLE_LOGGING_DRIVERS=[\"json-file\", \"none\", \"gelf\", \"awslogs\"] >> /etc/ecs/ecs.config
echo ECS_ENABLE_CONTAINER_METADATA=true >> /etc/ecs/ecs.config
cloud-init-per once docker_options echo 'OPTIONS="$${OPTIONS} --storage-opt dm.basesize=${var.ec2_block_device_size - 2}G"' >> /etc/sysconfig/docker

EOF

  ebs_block_device {
    device_name           = "/dev/xvdcz"
    volume_size           = var.ec2_block_device_size
    delete_on_termination = true
  }

  tags = {
    Name        = "${var.project}-${var.environment}-${var.family}-ecs-${count.index}"
    Environment = var.environment
    Project     = var.project
    Component   = "${var.family}_cluster"
  }

  lifecycle {
    create_before_destroy = true
  }
}
