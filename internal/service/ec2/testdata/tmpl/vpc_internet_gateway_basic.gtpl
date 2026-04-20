resource "aws_internet_gateway" "test" {
{{- template "region" }}
{{- template "tags" . }}
}
