resource "aws_sns_topic" "main" {
  name = "${var.project}-${var.environment}-${var.family}-${var.application}-${var.topicName}"
}

resource "aws_ssm_parameter" "topic-arn" {
  name  = "/${var.project}/${var.environment}/${var.family}/${var.application}/${var.topicName}_sns_arn"
  type  = "String"
  value = aws_sns_topic.main.arn
}
