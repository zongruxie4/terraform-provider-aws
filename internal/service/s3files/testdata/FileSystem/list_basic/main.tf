# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

resource "aws_s3_bucket" "test" {
  count = var.resource_count

  bucket = "s3files-private-beta-2025-${var.rName}-${count.index}"
}

resource "aws_s3_bucket_versioning" "test" {
  count = var.resource_count

  bucket = aws_s3_bucket.test[count.index].id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_iam_role" "test" {
  count = var.resource_count

  name = "${var.rName}-${count.index}"

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
  count = var.resource_count

  name = "${var.rName}-${count.index}"
  role = aws_iam_role.test[count.index].id

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
          aws_s3_bucket.test[count.index].arn,
          "${aws_s3_bucket.test[count.index].arn}/*"
        ]
      }
    ]
  })
}

resource "aws_s3files_file_system" "test" {
  count = var.resource_count

  bucket   = aws_s3_bucket.test[count.index].arn
  role_arn = aws_iam_role.test[count.index].arn

  depends_on = [aws_s3_bucket_versioning.test]
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
