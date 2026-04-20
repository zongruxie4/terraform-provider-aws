resource "aws_securityhub_configuration_policy_association" "test" {
{{- template "region" }}
  target_id = aws_organizations_organizational_unit.test.id
  policy_id = aws_securityhub_configuration_policy.test.id
}

data "aws_caller_identity" "member" {}

resource "aws_securityhub_organization_admin_account" "test" {
{{- template "region" }}
  provider = awsalternate

  admin_account_id = data.aws_caller_identity.member.account_id
}

data "aws_organizations_organization" "test" {
  provider = awsalternate
}

resource "aws_organizations_organizational_unit" "test" {
  provider = awsalternate

  name      = "${var.rName}-ou"
  parent_id = data.aws_organizations_organization.test.roots[0].id
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

data "aws_partition" "current" {}

resource "aws_securityhub_configuration_policy" "test" {
{{- template "region" }}
  name = "${var.rName}-policy"

  configuration_policy {
    service_enabled       = true
    enabled_standard_arns = ["arn:${data.aws_partition.current.partition}:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"]

    security_controls_configuration {
      disabled_control_identifiers = []
    }
  }

  depends_on = [aws_securityhub_organization_configuration.test]
}

{{ template "acctest.ConfigAlternateAccountProvider" }}
