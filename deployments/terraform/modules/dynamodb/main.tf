locals {
  table_name = length(var.table_name) > 0 ? var.table_name : "${var.project}-${var.environment}-${var.family}-${var.application}-${var.model}"

  attributes = concat(
    [
      {
        name = var.range_key
        type = var.range_key_type
      },
      {
        name = var.hash_key
        type = var.hash_key_type
      }
    ],
    var.attributes
  )

  # Remove the first map from the list if no `range_key` is provided
  from_index = length(var.range_key) > 0 ? 0 : 1

  attributes_final = slice(local.attributes, local.from_index, length(local.attributes))
}

resource "null_resource" "global_secondary_index_names" {
  count = (var.enabled ? 1 : 0) * length(var.global_secondary_index)

  # Convert the multi-item `global_secondary_index_map` into a simple `map` with just one item `name` since `triggers` does not support `lists` in `maps` (which are used in `non_key_attributes`)
  # See `examples/complete`
  # https://www.terraform.io/docs/providers/aws/r/dynamodb_table.html#non_key_attributes-1
  triggers = {
    "name" = var.global_secondary_index[count.index]["name"]
  }
}

resource "null_resource" "local_secondary_index_names" {
  count = (var.enabled ? 1 : 0) * length(var.local_secondary_index)

  # Convert the multi-item `local_secondary_index_map` into a simple `map` with just one item `name` since `triggers` does not support `lists` in `maps` (which are used in `non_key_attributes`)
  # See `examples/complete`
  # https://www.terraform.io/docs/providers/aws/r/dynamodb_table.html#non_key_attributes-1
  triggers = {
    "name" = var.local_secondary_index[count.index]["name"]
  }
}

resource "aws_dynamodb_table" "default" {
  count            = var.enabled ? 1 : 0
  name             = local.table_name
  billing_mode     = var.billing_mode
  read_capacity    = var.autoscale_min_read_capacity
  write_capacity   = var.autoscale_min_write_capacity
  hash_key         = var.hash_key
  range_key        = var.range_key
  stream_enabled   = var.enable_streams
  stream_view_type = var.enable_streams ? var.stream_view_type : ""

  server_side_encryption {
    enabled = var.enable_encryption
  }

  point_in_time_recovery {
    enabled = var.enable_point_in_time_recovery
  }

  lifecycle {
    ignore_changes = [
      read_capacity,
      write_capacity
    ]
  }

  dynamic "attribute" {
    for_each = local.attributes_final
    content {
      name = attribute.value.name
      type = attribute.value.type
    }
  }

  dynamic "global_secondary_index" {
    for_each = var.global_secondary_index
    content {
      hash_key           = global_secondary_index.value.hash_key
      name               = global_secondary_index.value.name
      non_key_attributes = lookup(global_secondary_index.value, "non_key_attributes", null)
      projection_type    = global_secondary_index.value.projection_type
      range_key          = lookup(global_secondary_index.value, "range_key", null)
      read_capacity      = lookup(global_secondary_index.value, "read_capacity", null)
      write_capacity     = lookup(global_secondary_index.value, "write_capacity", null)
    }
  }

  dynamic "local_secondary_index" {
    for_each = var.local_secondary_index
    content {
      name               = local_secondary_index.value.name
      non_key_attributes = lookup(local_secondary_index.value, "non_key_attributes", null)
      projection_type    = local_secondary_index.value.projection_type
      range_key          = local_secondary_index.value.range_key
    }
  }

  ttl {
    attribute_name = var.ttl
    enabled        = var.ttl != "" && var.ttl != null ? true : false
  }

  tags = merge({
    Project     = var.project
    Environment = var.environment
    Family      = var.family
    Application = var.application
    Model       = var.model
  }, var.tags)
}

module "dynamodb_autoscaler" {
  source                         = "./../dynamodb-autoscaler"
  enabled                        = var.enabled && var.enable_autoscaler && var.billing_mode == "PROVISIONED"
  dynamodb_table_name            = join("", aws_dynamodb_table.default.*.id)
  dynamodb_indexes               = null_resource.global_secondary_index_names.*.triggers.name
  enabled_global_secondary_index = var.enabled_global_secondary_index && var.billing_mode == "PROVISIONED"
  autoscale_write_target         = var.autoscale_write_target
  autoscale_read_target          = var.autoscale_read_target
  autoscale_min_read_capacity    = var.autoscale_min_read_capacity
  autoscale_max_read_capacity    = var.autoscale_max_read_capacity
  autoscale_min_write_capacity   = var.autoscale_min_write_capacity
  autoscale_max_write_capacity   = var.autoscale_max_write_capacity

  autoscaling_schedule_table_read_start  = var.autoscaling_schedule_table_read_start
  autoscaling_schedule_table_read_stop   = var.autoscaling_schedule_table_read_stop
  autoscaling_schedule_table_write_start = var.autoscaling_schedule_table_write_start
  autoscaling_schedule_table_write_stop  = var.autoscaling_schedule_table_write_stop
}