// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package glue_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/YakDriver/smarterr"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfglue "github.com/hashicorp/terraform-provider-aws/internal/service/glue"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func testAccCatalog_basic(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var catalog glue.GetCatalogOutput
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					resource.TestCheckResourceAttr(resourceName, names.AttrName, "s3tablescatalog"),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrCatalogID),
					resource.TestCheckResourceAttr(resourceName, names.AttrDescription, "Test S3 Tables federated catalog"),
					resource.TestCheckResourceAttr(resourceName, "federated_catalog.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "federated_catalog.0.identifier"),
					resource.TestCheckResourceAttr(resourceName, "federated_catalog.0.connection_name", "aws:s3tables"),
					acctest.MatchResourceAttrRegionalARN(ctx, resourceName, names.AttrARN, "glue", regexache.MustCompile(`catalog/.+$`)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCatalog_tags(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var catalog glue.GetCatalogOutput
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_tags1(rName, acctest.CtKey1, acctest.CtValue1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "1"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey1, acctest.CtValue1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCatalogConfig_tags2(rName, acctest.CtKey1, acctest.CtValue1Updated, acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "2"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey1, acctest.CtValue1Updated),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey2, acctest.CtValue2),
				),
			},
			{
				Config: testAccCatalogConfig_tags1(rName, acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "1"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey2, acctest.CtValue2),
				),
			},
		},
	})
}

func testAccCatalog_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var catalog glue.GetCatalogOutput
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					acctest.CheckFrameworkResourceDisappears(ctx, t, tfglue.ResourceCatalog, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCatalog_catalogProperties(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var catalog glue.GetCatalogOutput
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					resource.TestCheckResourceAttr(resourceName, names.AttrName, "s3tablescatalog"),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrCatalogID),
					resource.TestCheckResourceAttr(resourceName, names.AttrDescription, "Test S3 Tables federated catalog"),
					resource.TestCheckResourceAttr(resourceName, "federated_catalog.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "federated_catalog.0.identifier"),
					resource.TestCheckResourceAttr(resourceName, "federated_catalog.0.connection_name", "aws:s3tables"),
					acctest.MatchResourceAttrRegionalARN(ctx, resourceName, names.AttrARN, "glue", regexache.MustCompile(`catalog/.+$`)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCatalog_targetRedshiftCatalog(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var catalog glue.GetCatalogOutput
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_targetRedshiftCatalog(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					resource.TestCheckResourceAttr(resourceName, names.AttrName, rName),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrCatalogID),
					resource.TestCheckResourceAttr(resourceName, "target_redshift_catalog.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "target_redshift_catalog.0.catalog_arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCatalog_targetRedshiftCatalogProvisioned(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var catalog glue.GetCatalogOutput
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_targetRedshiftCatalogProvisioned(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					resource.TestCheckResourceAttr(resourceName, names.AttrName, rName),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrCatalogID),
					resource.TestCheckResourceAttr(resourceName, "target_redshift_catalog.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "target_redshift_catalog.0.catalog_arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCatalog_configurationError(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config:      testAccCatalogConfig_missingConfiguration(rName),
				ExpectError: regexache.MustCompile("Missing Required Configuration"),
			},
		},
	})
}

func testAccCatalog_Disappears_catalogProperties(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var catalog glue.GetCatalogOutput
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName, &catalog),
					acctest.CheckFrameworkResourceDisappears(ctx, t, tfglue.ResourceCatalog, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckCatalogDestroy(ctx context.Context, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).GlueClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_glue_catalog" {
				continue
			}

			_, err := tfglue.FindCatalogByID(ctx, conn, rs.Primary.ID)
			if retry.NotFound(err) {
				return nil
			}
			if err != nil {
				return smarterr.NewError(err)
			}

			return smarterr.NewError(errors.New("not destroyed"))
		}

		return nil
	}
}

func testAccCheckCatalogExists(ctx context.Context, t *testing.T, name string, catalog *glue.GetCatalogOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return smarterr.NewError(errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return smarterr.NewError(errors.New("not set"))
		}

		conn := acctest.ProviderMeta(ctx, t).GlueClient(ctx)

		resp, err := tfglue.FindCatalogByID(ctx, conn, rs.Primary.ID)
		if err != nil {
			return smarterr.NewError(err)
		}

		*catalog = glue.GetCatalogOutput{
			Catalog: resp,
		}

		return nil
	}
}

func testAccPreCheck(ctx context.Context, t *testing.T) {
	conn := acctest.ProviderMeta(ctx, t).GlueClient(ctx)

	if conn == nil {
		t.Fatal("Glue client is not configured")
	}
}

func testAccCatalogConfig_missingConfiguration(rName string) string {
	return fmt.Sprintf(`
resource "aws_glue_catalog" "test" {
  name        = %[1]q
  description = "Test federated catalog without required configuration"
}
`, rName)
}

func testAccCatalogConfig_s3Tables(rName string) string {
	return acctest.ConfigCompose(
		testAccCatalogConfig_s3TablesBase(rName), `
resource "aws_glue_catalog" "test" {
  name        = "s3tablescatalog"
  description = "Test S3 Tables federated catalog"

  federated_catalog {
    identifier      = "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:bucket/*"
    connection_name = "aws:s3tables"
  }

  depends_on = [
    aws_lakeformation_resource.test,
    aws_lakeformation_data_lake_settings.test,
  ]
}
`,
	)
}

func testAccCatalogConfig_s3TablesBase(rName string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}
data "aws_partition" "current" {}

data "aws_iam_session_context" "current" {
  arn = data.aws_caller_identity.current.arn
}

# Grant Lake Formation admin permissions to the test runner
resource "aws_lakeformation_data_lake_settings" "test" {
  admins = [data.aws_iam_session_context.current.issuer_arn]
}

# IAM role for Lake Formation data access
resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "lakeformation.amazonaws.com"
        }
        Action = [
          "sts:AssumeRole",
          "sts:SetSourceIdentity",
          "sts:SetContext"
        ]
      }
    ]
  })
}

# IAM policy for S3 Tables permissions
resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "LakeFormationPermissionsForS3ListTableBucket"
        Effect = "Allow"
        Action = [
          "s3tables:ListTableBuckets"
        ]
        Resource = ["*"]
      },
      {
        Sid    = "LakeFormationDataAccessPermissionsForS3TableBucket"
        Effect = "Allow"
        Action = [
          "s3tables:CreateTableBucket",
          "s3tables:GetTableBucket",
          "s3tables:CreateNamespace",
          "s3tables:GetNamespace",
          "s3tables:ListNamespaces",
          "s3tables:DeleteNamespace",
          "s3tables:DeleteTableBucket",
          "s3tables:CreateTable",
          "s3tables:DeleteTable",
          "s3tables:GetTable",
          "s3tables:ListTables",
          "s3tables:RenameTable",
          "s3tables:UpdateTableMetadataLocation",
          "s3tables:GetTableMetadataLocation",
          "s3tables:GetTableData",
          "s3tables:PutTableData"
        ]
        Resource = [
          "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:bucket/*"
        ]
      }
    ]
  })
}

# Register the S3 Tables location with Lake Formation
resource "aws_lakeformation_resource" "test" {
  arn      = "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:bucket/*"
  role_arn = aws_iam_role.test.arn

  depends_on = [aws_lakeformation_data_lake_settings.test]
}
`, rName)
}

func testAccCatalogConfig_tags1(rName, tagKey1, tagValue1 string) string {
	return acctest.ConfigCompose(
		testAccCatalogConfig_s3TablesBase(rName), fmt.Sprintf(`
resource "aws_glue_catalog" "test" {
  name        = "s3tablescatalog"
  description = "Test S3 Tables federated catalog"

  federated_catalog {
    identifier      = "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:bucket/*"
    connection_name = "aws:s3tables"
  }

  tags = {
    %[1]q = %[2]q
  }

  depends_on = [
    aws_lakeformation_resource.test,
    aws_lakeformation_data_lake_settings.test,
  ]
}
`, tagKey1, tagValue1),
	)
}

func testAccCatalogConfig_tags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return acctest.ConfigCompose(
		testAccCatalogConfig_s3TablesBase(rName), fmt.Sprintf(`
resource "aws_glue_catalog" "test" {
  name        = "s3tablescatalog"
  description = "Test S3 Tables federated catalog"

  federated_catalog {
    identifier      = "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:bucket/*"
    connection_name = "aws:s3tables"
  }

  tags = {
    %[1]q = %[2]q
    %[3]q = %[4]q
  }

  depends_on = [
    aws_lakeformation_resource.test,
    aws_lakeformation_data_lake_settings.test,
  ]
}
`, tagKey1, tagValue1, tagKey2, tagValue2),
	)
}

func testAccCatalogConfig_targetRedshiftCatalog(rName string) string {
	return fmt.Sprintf(`
data "aws_partition" "current" {}
data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

data "aws_iam_session_context" "current" {
  arn = data.aws_caller_identity.current.arn
}

resource "aws_lakeformation_data_lake_settings" "test" {
  admins = [data.aws_iam_session_context.current.issuer_arn]
}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = [
            "lakeformation.amazonaws.com",
            "glue.amazonaws.com",
            "redshift.amazonaws.com",
          ]
        }
        Action = [
          "sts:AssumeRole",
          "sts:SetSourceIdentity",
          "sts:SetContext"
        ]
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
          "redshift-serverless:GetCredentials",
          "redshift-serverless:GetWorkgroup"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_redshiftserverless_namespace" "test" {
  namespace_name = %[1]q
  db_name        = "test"
}

resource "aws_redshiftserverless_workgroup" "test" {
  namespace_name = aws_redshiftserverless_namespace.test.namespace_name
  workgroup_name = %[1]q
}

resource "aws_redshift_namespace_registration" "test" {
  consumer_identifier             = format("DataCatalog/%%s", data.aws_caller_identity.current.account_id)
  namespace_type                  = "serverless"
  serverless_namespace_identifier = aws_redshiftserverless_namespace.test.namespace_id
  serverless_workgroup_identifier = aws_redshiftserverless_workgroup.test.workgroup_name
}

locals {
  data_share_arn = format("arn:%%s:redshift:%%s:%%s:datashare:%%s/%%s",
    data.aws_partition.current.partition,
    data.aws_region.current.name,
    data.aws_caller_identity.current.account_id,
    aws_redshiftserverless_namespace.test.namespace_id,
    "ds_internal_namespace",
  )
}

resource "aws_redshift_data_share_consumer_association" "test" {
  data_share_arn = local.data_share_arn
  consumer_arn = format("arn:%%s:glue:%%s:%%s:catalog",
    data.aws_partition.current.partition,
    data.aws_region.current.name,
    data.aws_caller_identity.current.account_id,
  )

  depends_on = [
    aws_redshift_namespace_registration.test,
  ]
}

resource "aws_lakeformation_resource" "test" {
  depends_on = [aws_redshift_data_share_consumer_association.test]

  arn                     = local.data_share_arn
  use_service_linked_role = false
}

resource "aws_glue_catalog" "target" {
  name = "%[1]s-target"

  catalog_properties {
    data_lake_access_properties {
      data_lake_access   = true
      data_transfer_role = aws_iam_role.test.arn
    }
  }

  federated_catalog {
    identifier      = local.data_share_arn
    connection_name = "aws:redshift"
  }

  depends_on = [
    aws_lakeformation_data_lake_settings.test,
    aws_redshift_namespace_registration.test,
    aws_lakeformation_resource.test,
    aws_iam_role_policy.test,
  ]
}

resource "aws_glue_catalog" "test" {
  name = %[1]q

  target_redshift_catalog {
    catalog_arn = "${aws_glue_catalog.target.arn}/${aws_redshiftserverless_namespace.test.db_name}"
  }

  catalog_properties {
    data_lake_access_properties {
      data_lake_access   = true
      data_transfer_role = aws_iam_role.test.arn
    }
  }

  depends_on = [
    aws_lakeformation_data_lake_settings.test,
    aws_iam_role_policy.test,
  ]
}
`, rName)
}

func testAccCatalogConfig_targetRedshiftCatalogProvisioned(rName string) string {
	return fmt.Sprintf(`
data "aws_partition" "current" {}
data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

data "aws_iam_session_context" "current" {
  arn = data.aws_caller_identity.current.arn
}

resource "aws_lakeformation_data_lake_settings" "test" {
  admins = [data.aws_iam_session_context.current.issuer_arn]
}

resource "aws_iam_role" "test" {
  name = %[1]q

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = [
            "lakeformation.amazonaws.com",
            "glue.amazonaws.com",
            "redshift.amazonaws.com",
          ]
        }
        Action = [
          "sts:AssumeRole",
          "sts:SetSourceIdentity",
          "sts:SetContext"
        ]
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
          "redshift:GetClusterCredentials",
          "redshift:DescribeClusters"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_redshift_cluster" "test" {
  cluster_identifier  = %[1]q
  database_name       = "test"
  master_username     = "testuser"
  master_password     = "Testpass123"
  node_type           = "ra3.large"
  cluster_type        = "single-node"
  skip_final_snapshot = true
}

resource "aws_redshift_namespace_registration" "test" {
  consumer_identifier            = format("DataCatalog/%%s", data.aws_caller_identity.current.account_id)
  namespace_type                 = "provisioned"
  provisioned_cluster_identifier = aws_redshift_cluster.test.cluster_identifier
}

locals {
  # Extract namespace ID from cluster_namespace_arn
  # Format: arn:aws:redshift:region:account:namespace:namespace-id
  namespace_id = element(split(":", aws_redshift_cluster.test.cluster_namespace_arn), 6)
  data_share_arn = format("arn:%%s:redshift:%%s:%%s:datashare:%%s/%%s",
    data.aws_partition.current.partition,
    data.aws_region.current.name,
    data.aws_caller_identity.current.account_id,
    local.namespace_id,
    "ds_internal_namespace",
  )
}

resource "aws_redshift_data_share_consumer_association" "test" {
  data_share_arn = local.data_share_arn
  consumer_arn = format("arn:%%s:glue:%%s:%%s:catalog",
    data.aws_partition.current.partition,
    data.aws_region.current.name,
    data.aws_caller_identity.current.account_id,
  )

  depends_on = [
    aws_redshift_namespace_registration.test,
  ]
}

resource "aws_lakeformation_resource" "test" {
  depends_on = [aws_redshift_data_share_consumer_association.test]

  arn                     = local.data_share_arn
  use_service_linked_role = false
}

resource "aws_glue_catalog" "target" {
  name = "%[1]s-target"

  catalog_properties {
    data_lake_access_properties {
      data_lake_access   = true
      data_transfer_role = aws_iam_role.test.arn
    }
  }

  federated_catalog {
    identifier      = local.data_share_arn
    connection_name = "aws:redshift"
  }

  depends_on = [
    aws_lakeformation_data_lake_settings.test,
    aws_redshift_namespace_registration.test,
    aws_lakeformation_resource.test,
    aws_iam_role_policy.test,
  ]
}

resource "aws_glue_catalog" "test" {
  name = %[1]q

  target_redshift_catalog {
    catalog_arn = "${aws_glue_catalog.target.arn}/${aws_redshift_cluster.test.database_name}"
  }

  catalog_properties {
    data_lake_access_properties {
      data_lake_access   = true
      data_transfer_role = aws_iam_role.test.arn
    }
  }

  depends_on = [
    aws_lakeformation_data_lake_settings.test,
    aws_iam_role_policy.test,
  ]
}
`, rName)
}
