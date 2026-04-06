# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

resource "aws_opensearchserverless_collection_group" "test" {
  name             = var.rName
  standby_replicas = "ENABLED"
}

variable "rName" {
  description = "Name for resource"
  type        = string
  nullable    = false
}
