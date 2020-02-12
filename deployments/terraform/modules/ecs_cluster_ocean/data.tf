data "aws_ssm_parameter" "ami" {
  name = "/mcoins/packer/ecs/ami"
}
