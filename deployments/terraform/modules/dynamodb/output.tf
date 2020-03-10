output "table_name" {
  value       = join("", aws_dynamodb_table.default.*.name)
  description = "DynamoDB table name"
}

output "table_id" {
  value       = join("", aws_dynamodb_table.default.*.id)
  description = "DynamoDB table ID"
}

output "table_arn" {
  value       = join("", aws_dynamodb_table.default.*.arn)
  description = "DynamoDB table ARN"
}

output "global_secondary_index_names" {
  value       = null_resource.global_secondary_index_names.*.triggers.name
  description = "DynamoDB secondary index names"
}

output "local_secondary_index_names" {
  value       = null_resource.local_secondary_index_names.*.triggers.name
  description = "DynamoDB local index names"
}

output "table_stream_arn" {
  value       = join("", aws_dynamodb_table.default.*.stream_arn)
  description = "DynamoDB table stream ARN"
}

output "table_stream_label" {
  value       = join("", aws_dynamodb_table.default.*.stream_label)
  description = "DynamoDB table stream label"
}