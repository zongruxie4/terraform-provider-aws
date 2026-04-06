resource "aws_opensearchserverless_collection_group" "test" {
{{- template "region" }}
  name             = var.rName
  standby_replicas = "ENABLED"

{{- template "tags" . }}
}
