// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package s3files_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccS3FilesMountTargetDataSource_basic(t *testing.T) {
	ctx := acctest.Context(t)
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	dataSourceName := "data.aws_s3files_mount_target.test"
	resourceName := "aws_s3files_mount_target.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.S3FilesServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMountTargetDataSourceConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, "availability_zone_id", resourceName, "availability_zone_id"),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrFileSystemID, resourceName, names.AttrFileSystemID),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrID, resourceName, names.AttrID),
					resource.TestCheckResourceAttrPair(dataSourceName, "ipv4_address", resourceName, "ipv4_address"),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrNetworkInterfaceID, resourceName, names.AttrNetworkInterfaceID),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrOwnerID, resourceName, names.AttrOwnerID),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrSecurityGroups, resourceName, names.AttrSecurityGroups),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrStatus, resourceName, names.AttrStatus),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrSubnetID, resourceName, names.AttrSubnetID),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrVPCID, resourceName, names.AttrVPCID),
				),
			},
		},
	})
}

func testAccMountTargetDataSourceConfig_basic(rName string) string {
	return acctest.ConfigCompose(
		acctest.ConfigVPCWithSubnets(rName, 1),
		fmt.Sprintf(`
resource "aws_s3_bucket" "test" {
  bucket = "s3files-private-beta-2025-%[1]s"
}

resource "aws_s3_bucket_versioning" "test" {
  bucket = aws_s3_bucket.test.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_iam_role" "test" {
  name = %[1]q

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
  name = %[1]q
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
  file_system_id = aws_s3files_file_system.test.id
  subnet_id      = aws_subnet.test[0].id
}

data "aws_s3files_mount_target" "test" {
  id = aws_s3files_mount_target.test.id
}
`, rName))
}
