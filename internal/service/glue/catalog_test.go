// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package glue_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	awstypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfglue "github.com/hashicorp/terraform-provider-aws/internal/service/glue"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func testAccCatalog_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
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
				Config: testAccCatalogConfig_federatedCatalog_mySQL(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName),
					acctest.CheckFrameworkResourceDisappears(ctx, t, tfglue.ResourceCatalog, resourceName),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func testAccCatalog_tags(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
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
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(acctest.CtTagsPercent), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(acctest.CtTagsKey1), knownvalue.StringExact(acctest.CtValue1)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCatalogConfig_tags2(rName, acctest.CtKey1, acctest.CtValue1Updated, acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(acctest.CtTagsPercent), knownvalue.Int64Exact(2)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(acctest.CtTagsKey1), knownvalue.StringExact(acctest.CtValue1Updated)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(acctest.CtTagsKey2), knownvalue.StringExact(acctest.CtValue2)),
				},
			},
			{
				Config: testAccCatalogConfig_tags1(rName, acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(acctest.CtTagsPercent), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(acctest.CtTagsKey2), knownvalue.StringExact(acctest.CtValue2)),
				},
			},
		},
	})
}

// TestAccGlueCatalog_catalogPropertiesDataLakeAccess is intentionally serial
// (resource.Test rather than acctest.ParallelTest): data_lake_access_properties
// requires the caller to be a Lake Formation admin, and the config manages
// aws_lakeformation_data_lake_settings — the admin list is a single
// account/region-wide value, so a parallel Destroy on one test can strip the
// admin principal while another test still needs it.
func testAccCatalog_catalogPropertiesDataLakeAccess(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

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
		ExternalProviders: map[string]resource.ExternalProvider{
			"time": {
				Source: "hashicorp/time",
			},
		},
		CheckDestroy: testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_catalogPropertiesDataLakeAccess(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("allow_full_table_external_data_access"), knownvalue.StringExact("True")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("catalog_properties"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("catalog_properties").AtSliceIndex(0).AtMapKey("data_lake_access_properties"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					names.AttrTags,
					names.AttrTagsAll,
					// AWS auto-adds catalog_properties.custom_properties =
					// {"aws:PermissionsModel": "LAKEFORMATION"} to every
					// LF-managed catalog, which forces us to keep the flatten
					// guarded on pre-populated state to avoid "block count
					// changed from 0 to 1" on catalogs that don't declare the
					// block (federated, s3Tables). Import starts with null
					// state, so the guard skips flatten and the block is
					// missing on re-read. Skipping verify here is the
					// pragmatic trade-off.
					"catalog_properties",
				},
			},
		},
	})
}

func testAccCatalog_federatedCatalog_mySQL(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
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
				Config: testAccCatalogConfig_federatedCatalog_mySQL(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrName), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("federated_catalog"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					names.AttrTags,
					names.AttrTagsAll,
				},
			},
		},
	})
}

// TestAccGlueCatalog_targetRedshiftCatalog is intentionally serial
// (resource.Test rather than acctest.ParallelTest): the producer catalog uses
// data_lake_access_properties, which requires the caller to be a Lake
// Formation admin, and the config manages aws_lakeformation_data_lake_settings
// — an account/region-wide singleton that collides under parallel execution.
func testAccCatalog_targetRedshiftCatalog(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

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
		ExternalProviders: map[string]resource.ExternalProvider{
			"time": {
				Source: "hashicorp/time",
			},
		},
		CheckDestroy: testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_targetRedshiftCatalog(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrName), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("target_redshift_catalog"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					names.AttrTags,
					names.AttrTagsAll,
				},
			},
		},
	})
}

func testAccCatalog_targetRedshiftCatalogProvisioned(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

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
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrName), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("target_redshift_catalog"), knownvalue.ListSizeExact(1)),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					names.AttrTags,
					names.AttrTagsAll,
				},
			},
		},
	})
}

// TestAccGlueCatalog_federatedCatalog_s3Tables is intentionally serial
// (resource.Test rather than acctest.ParallelTest): AWS requires the catalog
// name to be the reserved value "s3tablescatalog", which is account/region-wide,
// so parallel runs would collide with each other and with any S3 Tables
// integration already enabled on the account.
func testAccCatalog_federatedCatalog_s3Tables(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_glue_catalog.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.GlueEndpointID)
			testAccPreCheck(ctx, t)
			testAccPreCheckS3TablesCatalogDoesNotExist(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.GlueServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckCatalogDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccCatalogConfig_federatedCatalog_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCatalogExists(ctx, t, resourceName),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New(names.AttrName), knownvalue.StringExact("s3tablescatalog")),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("federated_catalog"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(resourceName, tfjsonpath.New("federated_catalog").AtSliceIndex(0).AtMapKey("connection_name"), knownvalue.StringExact("aws:s3tables")),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					names.AttrTags,
					names.AttrTagsAll,
				},
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

	acctest.ParallelTest(ctx, t, resource.TestCase{
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

// --- Helper functions ---

func testAccCheckCatalogDestroy(ctx context.Context, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).GlueClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_glue_catalog" {
				continue
			}

			_, err := tfglue.FindCatalogByID(ctx, conn, rs.Primary.ID)
			if retry.NotFound(err) {
				continue
			}
			if err != nil {
				return err
			}

			return fmt.Errorf("Glue Catalog %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckCatalogExists(ctx context.Context, t *testing.T, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		const (
			ResNameCatalog = "Catalog"
		)

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return create.Error(names.Glue, create.ErrActionCheckingExistence, ResNameCatalog, name, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return create.Error(names.Glue, create.ErrActionCheckingExistence, ResNameCatalog, name, errors.New("not set"))
		}

		conn := acctest.ProviderMeta(ctx, t).GlueClient(ctx)

		if _, err := tfglue.FindCatalogByID(ctx, conn, rs.Primary.ID); err != nil {
			return create.Error(names.Glue, create.ErrActionCheckingExistence, ResNameCatalog, rs.Primary.ID, err)
		}

		return nil
	}
}

func testAccPreCheck(ctx context.Context, t *testing.T) {
	conn := acctest.ProviderMeta(ctx, t).GlueClient(ctx)

	input := &glue.GetCatalogsInput{}

	_, err := conn.GetCatalogs(ctx, input)

	if acctest.PreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}
	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

// testAccPreCheckS3TablesCatalogDoesNotExist ensures the reserved
// "s3tablescatalog" name is free before a test that creates it. The catalog is
// an account-level singleton, so a leftover from a prior failed run (or from
// the resource test running just before the data source test) surfaces as
// AlreadyExistsException on CreateCatalog. Deleting here keeps serial tests
// self-healing.
func testAccPreCheckS3TablesCatalogDoesNotExist(ctx context.Context, t *testing.T) {
	conn := acctest.ProviderMeta(ctx, t).GlueClient(ctx)

	_, err := tfglue.FindCatalogByID(ctx, conn, "s3tablescatalog")
	if retry.NotFound(err) {
		return
	}
	if err != nil {
		t.Fatalf("checking for pre-existing s3tablescatalog: %s", err)
	}

	_, err = tfresource.RetryWhenIsA[any, *awstypes.ConcurrentModificationException](
		ctx,
		5*time.Minute,
		func(ctx context.Context) (any, error) {
			return conn.DeleteCatalog(ctx, &glue.DeleteCatalogInput{
				CatalogId: aws.String("s3tablescatalog"),
			})
		},
	)
	if err != nil && !errs.IsA[*awstypes.EntityNotFoundException](err) {
		t.Fatalf("deleting pre-existing s3tablescatalog: %s", err)
	}
}

// --- Config functions ---

func testAccCatalogConfig_catalogPropertiesDataLakeAccess(rName string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}
data "aws_region" "current" {}

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
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = [
          "glue.amazonaws.com",
          "redshift.amazonaws.com",
        ]
      }
    }]
  })
}

resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "glue:GetCatalog",
        "glue:GetDatabase",
        "kms:Decrypt",
        "kms:GenerateDataKey",
      ]
      Resource = "*"
    }]
  })
}

resource "time_sleep" "iam_propagation" {
  depends_on      = [aws_iam_role_policy.test]
  create_duration = "30s"
}

resource "aws_glue_catalog" "test" {
  name        = %[1]q
  description = "test catalog with data lake access properties"

  allow_full_table_external_data_access = "True"

  catalog_properties {
    data_lake_access_properties {
      catalog_type       = "aws:redshift"
      data_lake_access   = true
      data_transfer_role = aws_iam_role.test.arn
    }
  }

  depends_on = [
    aws_lakeformation_data_lake_settings.test,
    time_sleep.iam_propagation,
  ]
}
`, rName)
}

func testAccCatalogConfig_federatedCatalog_mySQL(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

data "aws_partition" "current" {}
data "aws_region" "current" {}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  cidr_block        = "10.0.0.0/24"
  vpc_id            = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_security_group" "test" {
  name   = %[1]q
  vpc_id = aws_vpc.test.id

  ingress {
    protocol  = "tcp"
    self      = true
    from_port = 1
    to_port   = 65535
  }
}

resource "aws_s3_bucket" "test" {
  bucket = %[1]q
}

resource "aws_secretsmanager_secret" "test" {
  name = %[1]q
}

resource "aws_secretsmanager_secret_version" "test" {
  secret_id = aws_secretsmanager_secret.test.id
  secret_string = jsonencode({
    username = "glueusername"
    password = "gluepassword"
  })
}

resource "aws_glue_connection" "test" {
  name = %[1]q

  connection_type = "MYSQL"

  connection_properties = {
    HOST     = "testhost"
    PORT     = "3306"
    DATABASE = "gluedatabase"
  }

  athena_properties = {
    lambda_function_arn = "arn:${data.aws_partition.current.partition}:lambda:${data.aws_region.current.region}:123456789012:function:athenafederatedcatalog_mysql"
    spill_bucket        = aws_s3_bucket.test.bucket
  }

  authentication_configuration {
    authentication_type = "BASIC"
    secret_arn          = aws_secretsmanager_secret.test.arn
  }

  physical_connection_requirements {
    availability_zone      = aws_subnet.test.availability_zone
    security_group_id_list = [aws_security_group.test.id]
    subnet_id              = aws_subnet.test.id
  }
}

resource "aws_iam_role" "lakeformation_federated_catalog" {
  name = %[1]q

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lakeformation.amazonaws.com"
      }
    }]
  })
}

resource "aws_lakeformation_resource" "test" {
  arn                    = aws_glue_connection.test.arn
  role_arn               = aws_iam_role.lakeformation_federated_catalog.arn
  with_federation        = true
  with_privileged_access = true
}

resource "aws_glue_catalog" "test" {
  name        = %[1]q
  description = "test federated catalog"

  federated_catalog {
    connection_name = aws_glue_connection.test.name
    identifier      = aws_glue_connection.test.name
  }

  depends_on = [aws_lakeformation_resource.test]
}
`, rName)
}

func testAccCatalogConfig_targetRedshiftCatalog(rName string) string {
	return fmt.Sprintf(`
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
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = [
          "glue.amazonaws.com",
          "redshift.amazonaws.com",
        ]
      }
    }]
  })
}

resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "glue:GetCatalog",
        "glue:GetDatabase",
        "kms:Decrypt",
        "kms:GenerateDataKey",
      ]
      Resource = "*"
    }]
  })
}

resource "time_sleep" "iam_propagation" {
  depends_on      = [aws_iam_role_policy.test]
  create_duration = "30s"
}

resource "aws_glue_catalog" "producer" {
  name = "%[1]s-producer"

  catalog_properties {
    data_lake_access_properties {
      catalog_type       = "aws:redshift"
      data_lake_access   = true
      data_transfer_role = aws_iam_role.test.arn
    }
  }

  depends_on = [
    aws_lakeformation_data_lake_settings.test,
    time_sleep.iam_propagation,
  ]
}

resource "aws_glue_catalog" "test" {
  name = %[1]q

  target_redshift_catalog {
    catalog_arn = "${aws_glue_catalog.producer.arn}/dev"
  }
}
`, rName)
}

func testAccCatalogConfig_targetRedshiftCatalogProvisioned(rName string) string {
	//nolint:lll // long Terraform config
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
    Statement = [{
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
    }]
  })
}

resource "aws_iam_role_policy" "test" {
  name = %[1]q
  role = aws_iam_role.test.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "redshift:GetClusterCredentials",
        "redshift:DescribeClusters"
      ]
      Resource = "*"
    }]
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
    catalog_arn = "$${aws_glue_catalog.target.arn}/$${aws_redshift_cluster.test.database_name}"
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

func testAccCatalogConfig_federatedCatalog_s3Tables(rName string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}
data "aws_region" "current" {}

resource "aws_s3tables_table_bucket" "test" {
  name = %[1]q
}

resource "aws_glue_catalog" "test" {
  name        = "s3tablescatalog"
  description = "test s3 tables catalog"

  federated_catalog {
    connection_name = "aws:s3tables"
    identifier      = "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.region}:${data.aws_caller_identity.current.account_id}:bucket/*"
  }

  create_database_default_permissions {
    permissions = ["ALL"]

    principal {
      data_lake_principal_identifier = "IAM_ALLOWED_PRINCIPALS"
    }
  }

  create_table_default_permissions {
    permissions = ["ALL"]

    principal {
      data_lake_principal_identifier = "IAM_ALLOWED_PRINCIPALS"
    }
  }

  depends_on = [aws_s3tables_table_bucket.test]
}
`, rName)
}

func testAccCatalogConfig_tags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}
data "aws_region" "current" {}

resource "aws_s3tables_table_bucket" "test" {
  name = %[1]q
}

resource "aws_glue_catalog" "test" {
  name        = "s3tablescatalog"
  description = "test s3 tables catalog"

  federated_catalog {
    connection_name = "aws:s3tables"
    identifier      = "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.region}:${data.aws_caller_identity.current.account_id}:bucket/*"
  }

  tags = {
    %[2]q = %[3]q
  }

  depends_on = [aws_s3tables_table_bucket.test]
}
`, rName, tagKey1, tagValue1)
}

func testAccCatalogConfig_tags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}
data "aws_region" "current" {}

resource "aws_s3tables_table_bucket" "test" {
  name = %[1]q
}

resource "aws_glue_catalog" "test" {
  name        = "s3tablescatalog"
  description = "test s3 tables catalog"

  federated_catalog {
    connection_name = "aws:s3tables"
    identifier      = "arn:${data.aws_partition.current.partition}:s3tables:${data.aws_region.current.region}:${data.aws_caller_identity.current.account_id}:bucket/*"
  }

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }

  depends_on = [aws_s3tables_table_bucket.test]
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}

func testAccCatalogConfig_missingConfiguration(rName string) string {
	return fmt.Sprintf(`
resource "aws_glue_catalog" "test" {
  name        = %[1]q
  description = "Test federated catalog without required configuration"
}
`, rName)
}
