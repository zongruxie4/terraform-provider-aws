resource "aws_ebs_volume_copy" "test" {
{{- template "region" }}
  source_volume_id = aws_ebs_volume.test.id

{{- template "tags" . }}
}

resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size              = 1
  encrypted         = true
}

data "aws_availability_zones" "available" {
  state = "available"
}
