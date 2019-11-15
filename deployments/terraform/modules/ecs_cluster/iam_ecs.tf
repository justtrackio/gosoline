resource "aws_iam_instance_profile" "ecs-container" {
  name = "${var.project}-${var.environment}-${var.family}-ecs"
  role = aws_iam_role.ecs-container.name
}

resource "aws_iam_role" "ecs-container" {
  name        = "${var.project}-${var.environment}-${var.family}-ecs"
  description = "Allows ECS tasks to call AWS services on your behalf."

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    },
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "events.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

  tags = {
    Project     = var.project
    Environment = var.environment
    Family      = var.family
  }
}

resource "aws_iam_role_policy_attachment" "dynamodb" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"
  role       = aws_iam_role.ecs-container.name
}

resource "aws_iam_role_policy_attachment" "kinesis" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonKinesisFullAccess"
  role       = aws_iam_role.ecs-container.name
}

resource "aws_iam_role_policy_attachment" "ssm" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMReadOnlyAccess"
  role       = aws_iam_role.ecs-container.name
}

resource "aws_iam_role_policy_attachment" "service_discovery" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonRoute53AutoNamingReadOnlyAccess"
  role       = aws_iam_role.ecs-container.name
}

resource "aws_iam_role_policy_attachment" "cloudwatch" {
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchFullAccess"
  role       = aws_iam_role.ecs-container.name
}

resource "aws_iam_role_policy_attachment" "sns" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonSNSFullAccess"
  role       = aws_iam_role.ecs-container.name
}

resource "aws_iam_role_policy_attachment" "sqs" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonSQSFullAccess"
  role       = aws_iam_role.ecs-container.name
}

resource "aws_iam_role_policy_attachment" "ecr" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.ecs-container.name
}
