# Copyright IBM Corp. 2014, 2026
# SPDX-License-Identifier: MPL-2.0

list "aws_s3files_file_system_policy" "test" {
  provider = awsalternate

  config {
    file_system_id = aws_s3files_file_system.test[0].id
    region         = data.aws_region.alternate.name
  }
}
