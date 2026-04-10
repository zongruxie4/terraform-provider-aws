resource "aws_autoscaling_traffic_source_attachment" "test" {
{{- template "region" }}
  autoscaling_group_name = aws_autoscaling_group.test.id

  traffic_source {
    identifier = aws_lb_target_group.test.arn
    type       = "elbv2"
  }
}

resource "aws_lb_target_group" "test" {
{{- template "region" }}
  name     = var.rName
  port     = 80
  protocol = "HTTP"
  vpc_id   = aws_vpc.test.id
}

resource "aws_launch_configuration" "test" {
{{- template "region" }}
  name          = var.rName
  image_id      = data.aws_ami.amzn2-ami-minimal-hvm-ebs-x86_64.id
  instance_type = "t2.micro"
}

resource "aws_autoscaling_group" "test" {
{{- template "region" }}
  vpc_zone_identifier       = aws_subnet.test[*].id
  max_size                  = 1
  min_size                  = 0
  desired_capacity          = 0
  health_check_grace_period = 300
  force_delete              = true
  name                      = var.rName
  launch_configuration      = aws_launch_configuration.test.name

  tag {
    key                 = "Name"
    value               = var.rName
    propagate_at_launch = true
  }
}

{{ template "acctest.ConfigLatestAmazonLinux2HVMEBSX8664AMI" }}

{{ template "acctest.ConfigVPCWithSubnets" 1 }}
