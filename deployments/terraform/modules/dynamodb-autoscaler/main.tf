resource "aws_appautoscaling_target" "read_target" {
  count              = var.enabled ? 1 : 0
  max_capacity       = var.autoscale_max_read_capacity
  min_capacity       = var.autoscale_min_read_capacity
  resource_id        = "table/${var.dynamodb_table_name}"
  scalable_dimension = "dynamodb:table:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_target" "read_target_index" {
  count              = var.enabled_global_secondary_index ? length(var.dynamodb_indexes) : 0
  max_capacity       = var.autoscale_max_read_capacity
  min_capacity       = var.autoscale_min_read_capacity
  resource_id        = "table/${var.dynamodb_table_name}/index/${element(var.dynamodb_indexes, count.index)}"
  scalable_dimension = "dynamodb:index:ReadCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "read_policy" {
  count       = var.enabled ? 1 : 0
  name        = "DynamoDBReadCapacityUtilization:${join("", aws_appautoscaling_target.read_target.*.resource_id)}"
  policy_type = "TargetTrackingScaling"
  resource_id = join("", aws_appautoscaling_target.read_target.*.resource_id)

  scalable_dimension = join("", aws_appautoscaling_target.read_target.*.scalable_dimension)
  service_namespace  = join("", aws_appautoscaling_target.read_target.*.service_namespace)

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBReadCapacityUtilization"
    }

    target_value = var.autoscale_read_target
  }
}

resource "aws_appautoscaling_policy" "read_policy_index" {
  count = var.enabled_global_secondary_index ? length(var.dynamodb_indexes) : 0

  name = "DynamoDBReadCapacityUtilization:${element(
    aws_appautoscaling_target.read_target_index.*.resource_id,
    count.index
  )}"

  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.read_target_index.*.resource_id[count.index]
  scalable_dimension = aws_appautoscaling_target.read_target_index.*.scalable_dimension[count.index]
  service_namespace  = aws_appautoscaling_target.read_target_index.*.service_namespace[count.index]

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBReadCapacityUtilization"
    }

    target_value = var.autoscale_read_target
  }
}

resource "aws_appautoscaling_target" "write_target" {
  count              = var.enabled ? 1 : 0
  max_capacity       = var.autoscale_max_write_capacity
  min_capacity       = var.autoscale_min_write_capacity
  resource_id        = "table/${var.dynamodb_table_name}"
  scalable_dimension = "dynamodb:table:WriteCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_target" "write_target_index" {
  count              = var.enabled_global_secondary_index ? length(var.dynamodb_indexes) : 0
  max_capacity       = var.autoscale_max_write_capacity
  min_capacity       = var.autoscale_min_write_capacity
  resource_id        = "table/${var.dynamodb_table_name}/index/${element(var.dynamodb_indexes, count.index)}"
  scalable_dimension = "dynamodb:index:WriteCapacityUnits"
  service_namespace  = "dynamodb"
}

resource "aws_appautoscaling_policy" "write_policy" {
  count       = var.enabled ? 1 : 0
  name        = "DynamoDBWriteCapacityUtilization:${join("", aws_appautoscaling_target.write_target.*.resource_id)}"
  policy_type = "TargetTrackingScaling"
  resource_id = join("", aws_appautoscaling_target.write_target.*.resource_id)

  scalable_dimension = join("", aws_appautoscaling_target.write_target.*.scalable_dimension)
  service_namespace  = join("", aws_appautoscaling_target.write_target.*.service_namespace)

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBWriteCapacityUtilization"
    }

    target_value = var.autoscale_write_target
  }
}

resource "aws_appautoscaling_policy" "write_policy_index" {
  count = var.enabled_global_secondary_index ? length(var.dynamodb_indexes) : 0

  name = "DynamoDBWriteCapacityUtilization:${element(
    aws_appautoscaling_target.write_target_index.*.resource_id,
    count.index
  )}"

  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.write_target_index.*.resource_id[count.index]
  scalable_dimension = aws_appautoscaling_target.write_target_index.*.scalable_dimension[count.index]
  service_namespace  = aws_appautoscaling_target.write_target_index.*.service_namespace[count.index]

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "DynamoDBWriteCapacityUtilization"
    }

    target_value = var.autoscale_write_target
  }
}