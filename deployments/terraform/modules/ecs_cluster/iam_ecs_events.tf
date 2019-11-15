resource "aws_iam_role" "ecs-events" {
  name = "${var.project}-${var.environment}-${var.family}-ecs-events"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
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
}

resource "aws_iam_policy" "ecs-events-pass-policy" {
  name = "${var.project}-${var.environment}-${var.family}-ecs-execution-policy"

  policy = <<EOF
{
  "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "iam:PassRole",
            "Resource": [
                "${aws_iam_role.ecs-container.arn}"
            ]
        }
    ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "ecs-events" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceEventsRole"
  role       = aws_iam_role.ecs-events.name
}

resource "aws_iam_role_policy_attachment" "ecs-events-pass-role-attachment" {
  policy_arn = aws_iam_policy.ecs-events-pass-policy.arn
  role       = aws_iam_role.ecs-events.name
}
