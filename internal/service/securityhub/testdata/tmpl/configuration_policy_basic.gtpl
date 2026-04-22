resource "aws_securityhub_configuration_policy" "test" {
{{- template "region" }}
  name = "${var.rName}-policy"

  configuration_policy {
    service_enabled = false
  }

  depends_on = [aws_securityhub_organization_configuration.test]
}

resource "aws_securityhub_finding_aggregator" "test" {
{{- template "region" }}
  linking_mode = "ALL_REGIONS"

  depends_on = [aws_securityhub_organization_admin_account.test]
}

resource "aws_securityhub_organization_configuration" "test" {
{{- template "region" }}
  auto_enable           = false
  auto_enable_standards = "NONE"
  organization_configuration {
    configuration_type = "CENTRAL"
  }

  depends_on = [aws_securityhub_finding_aggregator.test]
}

data "aws_caller_identity" "member" {}

resource "aws_securityhub_organization_admin_account" "test" {
{{- template "region" }}
  provider = awsalternate

  admin_account_id = data.aws_caller_identity.member.account_id
}

{{ template "acctest.ConfigAlternateAccountProvider" }}
