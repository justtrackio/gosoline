resource "aws_kinesis_stream" "main" {
  name        = "${var.project}-${var.environment}-${var.family}-${var.application}-firehose-${var.model}"
  shard_count = var.shard_count

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}

resource "aws_ssm_parameter" "kinesis-stream-name" {
  name  = "/${var.project}/${var.environment}/${var.family}/firehose/${var.model}/kinesis_stream_name"
  type  = "String"
  value = aws_kinesis_stream.main.name
}
