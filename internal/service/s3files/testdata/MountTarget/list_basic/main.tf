# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "test" {
  count             = var.resource_count
  vpc_id            = aws_vpc.test.id
  cidr_block        = cidrsubnet(aws_vpc.test.cidr_block, 8, count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]
}

data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_s3_bucket" "test" {
  bucket = "s3files-private-beta-2025-${var.rName}"
}

resource "aws_s3_bucket_versioning" "test" {
  bucket = aws_s3_bucket.test.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_iam_role" "test" {
  name = var.rName

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "elasticfilesystem.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "test" {
  name = var.rName
  role = aws_iam_role.test.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.test.arn,
          "${aws_s3_bucket.test.arn}/*"
        ]
      }
    ]
  })
}

resource "aws_s3files_file_system" "test" {
  bucket   = aws_s3_bucket.test.arn
  role_arn = aws_iam_role.test.arn

  depends_on = [aws_s3_bucket_versioning.test]
}

resource "aws_s3files_mount_target" "test" {
  count = var.resource_count

  file_system_id = aws_s3files_file_system.test.id
  subnet_id      = aws_subnet.test[count.index].id
}

variable "rName" {
  description = "Name for resource"
  type        = string
  nullable    = false
}

variable "resource_count" {
  description = "Number of resources to create"
  type        = number
  nullable    = false
}
