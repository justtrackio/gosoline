resource "aws_iam_role" "ecs-ec2" {
  name = "${var.project}-${var.environment}-${var.family}-ec2"

  assume_role_policy = <<EOF
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_policy" "ec2ssm" {
  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ssm:DescribeParameters"
            ],
            "Resource": "*"
        },
        {
            "Sid": "GetParametersRule",
            "Effect": "Allow",
            "Action": [
                "ssm:GetParameter"
            ],
            "Resource": [
                "arn:aws:ssm:eu-central-1:164105964448:parameter/${var.project}/${var.environment}/${var.family}/*"
            ]
        },
        {
            "Sid": "DecryptSecretsRule",
            "Effect": "Allow",
            "Action": [
                "kms:Decrypt"
            ],
            "Resource": [
                "arn:aws:kms:eu-central-1:164105964448:key/alias/aws/ssm"
            ]
        }
    ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "ec2container" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
  role       = aws_iam_role.ecs-ec2.name
}

resource "aws_iam_role_policy_attachment" "ec2ssm" {
  policy_arn = aws_iam_policy.ec2ssm.arn
  role       = aws_iam_role.ecs-ec2.name
}

resource "aws_iam_instance_profile" "ec2" {
  name = "${var.project}-${var.environment}-${var.family}-ecs-ec2"
  role = aws_iam_role.ecs-ec2.name
}
