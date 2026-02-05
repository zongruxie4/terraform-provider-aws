# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

list "aws_ecr_repository" "test" {
  provider = aws

  config {}
}
