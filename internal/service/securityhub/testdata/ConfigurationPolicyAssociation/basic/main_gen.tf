# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

resource "aws_securityhub_configuration_policy_association" "test" {
  target_id = aws_organizations_organizational_unit.test.id
  policy_id = aws_securityhub_configuration_policy.test.id
}

data "aws_caller_identity" "member" {}

resource "aws_securityhub_organization_admin_account" "test" {
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
  linking_mode = "ALL_REGIONS"

  depends_on = [aws_securityhub_organization_admin_account.test]
}

resource "aws_securityhub_organization_configuration" "test" {
  auto_enable           = false
  auto_enable_standards = "NONE"
  organization_configuration {
    configuration_type = "CENTRAL"
  }

  depends_on = [aws_securityhub_finding_aggregator.test]
}

data "aws_partition" "current" {}

resource "aws_securityhub_configuration_policy" "test" {
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

provider "awsalternate" {
  access_key = var.AWS_ALTERNATE_ACCESS_KEY_ID
  profile    = var.AWS_ALTERNATE_PROFILE
  secret_key = var.AWS_ALTERNATE_SECRET_ACCESS_KEY
}

variable "AWS_ALTERNATE_ACCESS_KEY_ID" {
  type     = string
  nullable = true
  default  = null
}

variable "AWS_ALTERNATE_PROFILE" {
  type     = string
  nullable = true
  default  = null
}

variable "AWS_ALTERNATE_SECRET_ACCESS_KEY" {
  type     = string
  nullable = true
  default  = null
}

variable "rName" {
  description = "Name for resource"
  type        = string
  nullable    = false
}
