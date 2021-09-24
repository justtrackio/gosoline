resource "aws_appautoscaling_scheduled_action" "table_read_start" {
  count = length(var.autoscaling_schedule_table_read_start)

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index}-read-schedule-start"
  service_namespace  = join("", aws_appautoscaling_target.read_target.*.service_namespace)
  resource_id        = join("", aws_appautoscaling_target.read_target.*.resource_id)
  scalable_dimension = join("", aws_appautoscaling_target.read_target.*.scalable_dimension)

  schedule = "cron(${var.autoscaling_schedule_table_read_start[count.index].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_read_start[count.index].min_capacity
    max_capacity = var.autoscaling_schedule_table_read_start[count.index].max_capacity
  }
}

resource "aws_appautoscaling_scheduled_action" "index_read_start" {
  count = var.enabled_global_secondary_index ? length(var.autoscaling_schedule_table_read_start) * length(var.dynamodb_indexes) : 0

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index % length(var.dynamodb_indexes)}-read-${element(var.dynamodb_indexes, count.index / length(var.dynamodb_indexes))}-schedule-start"
  service_namespace  = aws_appautoscaling_target.read_target_index[count.index % length(var.dynamodb_indexes)].service_namespace
  resource_id        = aws_appautoscaling_target.read_target_index[count.index % length(var.dynamodb_indexes)].resource_id
  scalable_dimension = aws_appautoscaling_target.read_target_index[count.index % length(var.dynamodb_indexes)].scalable_dimension

  schedule = "cron(${var.autoscaling_schedule_table_read_start[count.index % length(var.dynamodb_indexes)].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_read_start[count.index % length(var.dynamodb_indexes)].min_capacity
    max_capacity = var.autoscaling_schedule_table_read_start[count.index % length(var.dynamodb_indexes)].max_capacity
  }
}

resource "aws_appautoscaling_scheduled_action" "table_read_stop" {
  count = length(var.autoscaling_schedule_table_read_stop)

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index}-read-schedule-stop"
  service_namespace  = join("", aws_appautoscaling_target.read_target.*.service_namespace)
  resource_id        = join("", aws_appautoscaling_target.read_target.*.resource_id)
  scalable_dimension = join("", aws_appautoscaling_target.read_target.*.scalable_dimension)

  schedule = "cron(${var.autoscaling_schedule_table_read_stop[count.index].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_read_stop[count.index].min_capacity
    max_capacity = var.autoscaling_schedule_table_read_stop[count.index].max_capacity
  }
}

resource "aws_appautoscaling_scheduled_action" "index_read_stop" {
  count = var.enabled_global_secondary_index ? length(var.autoscaling_schedule_table_read_stop) * length(var.dynamodb_indexes) : 0

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index % length(var.dynamodb_indexes)}-read-${element(var.dynamodb_indexes, count.index / length(var.dynamodb_indexes))}-schedule-stop"
  service_namespace  = aws_appautoscaling_target.read_target_index[count.index % length(var.dynamodb_indexes)].service_namespace
  resource_id        = aws_appautoscaling_target.read_target_index[count.index % length(var.dynamodb_indexes)].resource_id
  scalable_dimension = aws_appautoscaling_target.read_target_index[count.index % length(var.dynamodb_indexes)].scalable_dimension

  schedule = "cron(${var.autoscaling_schedule_table_read_stop[count.index % length(var.dynamodb_indexes)].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_read_stop[count.index % length(var.dynamodb_indexes)].min_capacity
    max_capacity = var.autoscaling_schedule_table_read_stop[count.index % length(var.dynamodb_indexes)].max_capacity
  }
}

resource "aws_appautoscaling_scheduled_action" "table_write_start" {
  count = length(var.autoscaling_schedule_table_write_start)

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index}-write-schedule-start"
  service_namespace  = join("", aws_appautoscaling_target.write_target.*.service_namespace)
  resource_id        = join("", aws_appautoscaling_target.write_target.*.resource_id)
  scalable_dimension = join("", aws_appautoscaling_target.write_target.*.scalable_dimension)

  schedule = "cron(${var.autoscaling_schedule_table_write_start[count.index].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_write_start[count.index].min_capacity
    max_capacity = var.autoscaling_schedule_table_write_start[count.index].max_capacity
  }
}

resource "aws_appautoscaling_scheduled_action" "index_write_start" {
  count = var.enabled_global_secondary_index ? length(var.autoscaling_schedule_table_write_start) * length(var.dynamodb_indexes) : 0

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index % length(var.dynamodb_indexes)}-write-${element(var.dynamodb_indexes, count.index / length(var.dynamodb_indexes))}-schedule-start"
  service_namespace  = aws_appautoscaling_target.write_target_index[count.index % length(var.dynamodb_indexes)].service_namespace
  resource_id        = aws_appautoscaling_target.write_target_index[count.index % length(var.dynamodb_indexes)].resource_id
  scalable_dimension = aws_appautoscaling_target.write_target_index[count.index % length(var.dynamodb_indexes)].scalable_dimension

  schedule = "cron(${var.autoscaling_schedule_table_write_start[count.index % length(var.dynamodb_indexes)].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_write_start[count.index % length(var.dynamodb_indexes)].min_capacity
    max_capacity = var.autoscaling_schedule_table_write_start[count.index % length(var.dynamodb_indexes)].max_capacity
  }
}

resource "aws_appautoscaling_scheduled_action" "table_write_stop" {
  count = length(var.autoscaling_schedule_table_write_stop)

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index}-write-schedule-stop"
  service_namespace  = join("", aws_appautoscaling_target.write_target.*.service_namespace)
  resource_id        = join("", aws_appautoscaling_target.write_target.*.resource_id)
  scalable_dimension = join("", aws_appautoscaling_target.write_target.*.scalable_dimension)

  schedule = "cron(${var.autoscaling_schedule_table_write_stop[count.index].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_write_stop[count.index].min_capacity
    max_capacity = var.autoscaling_schedule_table_write_stop[count.index].max_capacity
  }
}

resource "aws_appautoscaling_scheduled_action" "index_write_stop" {
  count = var.enabled_global_secondary_index ? length(var.autoscaling_schedule_table_write_stop) * length(var.dynamodb_indexes) : 0

  name               = "dynamodb-${var.dynamodb_table_name}-${count.index % length(var.dynamodb_indexes)}-write-${element(var.dynamodb_indexes, count.index / length(var.dynamodb_indexes))}-schedule-stop"
  service_namespace  = aws_appautoscaling_target.write_target_index[count.index % length(var.dynamodb_indexes)].service_namespace
  resource_id        = aws_appautoscaling_target.write_target_index[count.index % length(var.dynamodb_indexes)].resource_id
  scalable_dimension = aws_appautoscaling_target.write_target_index[count.index % length(var.dynamodb_indexes)].scalable_dimension

  schedule = "cron(${var.autoscaling_schedule_table_write_stop[count.index % length(var.dynamodb_indexes)].cron})"

  scalable_target_action {
    min_capacity = var.autoscaling_schedule_table_write_stop[count.index % length(var.dynamodb_indexes)].min_capacity
    max_capacity = var.autoscaling_schedule_table_write_stop[count.index % length(var.dynamodb_indexes)].max_capacity
  }
}
