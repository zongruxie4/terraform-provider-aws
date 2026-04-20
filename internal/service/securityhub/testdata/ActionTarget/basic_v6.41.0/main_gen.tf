# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

resource "aws_securityhub_account" "test" {
}

resource "aws_securityhub_action_target" "test" {
  depends_on  = [aws_securityhub_account.test]
  description = "description1"
  identifier  = "testaction"
  name        = "Test action"
}

variable "rName" {
  description = "Name for resource"
  type        = string
  nullable    = false
}
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.41.0"
    }
  }
}

provider "aws" {}
