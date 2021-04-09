resource "aws_cloudwatch_metric_alarm" "firehose-read-bytes-high" {
  alarm_name          = "${var.family}-${var.application}-${var.family}-${var.model}-read-bytes-high"
  count               = var.alarm_create
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  metric_name         = "GetRecords.Bytes"
  namespace           = "AWS/Kinesis"
  period              = var.alarm_period_seconds
  statistic           = "Sum"
  threshold           = var.shard_count * var.alarm_period_seconds * var.alarm_limit_threshold_percentage / 100 * 1024 * 1024 * 2
  datapoints_to_alarm = var.alarm_datapoints_to_alarm
  treat_missing_data  = "breaching"

  alarm_description = "This metric monitors kinesis read bytes utilization"

  dimensions = {
    StreamName = aws_kinesis_stream.main.name
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}

resource "aws_cloudwatch_metric_alarm" "firehose-write-bytes-high" {
  alarm_name          = "${var.family}-${var.application}-${var.family}-${var.model}-write-bytes-high"
  count               = var.alarm_create
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  metric_name         = "IncomingBytes"
  namespace           = "AWS/Kinesis"
  period              = var.alarm_period_seconds
  statistic           = "Sum"
  threshold           = var.shard_count * var.alarm_period_seconds * var.alarm_limit_threshold_percentage / 100 * 1024 * 1024
  datapoints_to_alarm = var.alarm_datapoints_to_alarm
  treat_missing_data  = "notBreaching"

  alarm_description = "This metric monitors kinesis write bytes utilization"

  dimensions = {
    StreamName = aws_kinesis_stream.main.name
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}

resource "aws_cloudwatch_metric_alarm" "firehose-write-records-high" {
  alarm_name          = "${var.family}-${var.application}-${var.family}-${var.model}-write-records-high"
  count               = var.alarm_create
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  metric_name         = "IncomingRecords"
  namespace           = "AWS/Kinesis"
  period              = var.alarm_period_seconds
  statistic           = "Sum"
  threshold           = var.shard_count * var.alarm_period_seconds * var.alarm_limit_threshold_percentage / 100 * 1000
  datapoints_to_alarm = var.alarm_datapoints_to_alarm
  treat_missing_data  = "notBreaching"

  alarm_description = "This metric monitors kinesis write records utilization"

  dimensions = {
    StreamName = aws_kinesis_stream.main.name
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}

resource "aws_cloudwatch_metric_alarm" "firehose-iterator-age-high" {
  alarm_name          = "${var.family}-${var.application}-${var.family}-${var.model}-iterator-age-high"
  count               = var.alarm_create
  comparison_operator = "GreaterThanOrEqualToThreshold"
  evaluation_periods  = var.alarm_evaluation_periods
  metric_name         = "GetRecords.IteratorAgeMilliseconds"
  namespace           = "AWS/Kinesis"
  period              = var.alarm_period_seconds
  statistic           = "Average"
  threshold           = var.alarm_iterator_threshold_age_milliseconds
  datapoints_to_alarm = var.alarm_datapoints_to_alarm
  treat_missing_data  = "breaching"

  alarm_description = "This metric monitors kinesis iterator age"

  dimensions = {
    StreamName = aws_kinesis_stream.main.name
  }

  alarm_actions = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]
  ok_actions    = ["arn:aws:sns:eu-central-1:164105964448:${var.project}-${var.environment}-${var.family}-alarm"]

  tags = {
    Environment = var.environment
    Project     = var.project
    Family      = var.family
    Application = var.application
  }
}
