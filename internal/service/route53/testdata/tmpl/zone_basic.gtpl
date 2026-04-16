resource "aws_route53_zone" "test" {
{{- template "region" }}
  name = "${var.zoneName}."
{{- template "tags" . }}
}