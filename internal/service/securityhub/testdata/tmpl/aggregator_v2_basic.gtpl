resource "aws_securityhub_account_v2" "test" {}

resource "aws_securityhub_aggregator_v2" "test" {
  region_linking_mode = "SPECIFIED_REGIONS"
  linked_regions      = ["us-east-1"]

  depends_on = [aws_securityhub_account_v2.test]
{{- template "region" }}
{{- template "tags" . }}
}
