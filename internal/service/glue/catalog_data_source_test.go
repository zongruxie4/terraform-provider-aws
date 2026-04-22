// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package glue_test

import (
	"testing"

	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// testAccCatalogDataSource_catalogPropertiesDataLakeAccess is intentionally
// serial (resource.Test rather than acctest.ParallelTest):
// data_lake_access_properties requires the caller to be a Lake Formation
// admin, and the config manages aws_lakeformation_data_lake_settings — the
// admin list is a single account/region-wide value, so a parallel Destroy on
// one test can strip the admin principal while another test still needs it.
func testAccCatalogDataSource_catalogPropertiesDataLakeAccess(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	dataSourceName := "data.aws_glue_catalog.test"
	resourceName := "aws_glue_catalog.test"

	resource.Test(t, resource.TestCase{
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
				Config: testAccCatalogDataSourceConfig_catalogPropertiesDataLakeAccess(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrName, resourceName, names.AttrName),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrDescription, resourceName, names.AttrDescription),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrARN, resourceName, names.AttrARN),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrCatalogID, resourceName, names.AttrCatalogID),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("allow_full_table_external_data_access"), knownvalue.StringExact("True")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("catalog_properties"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("catalog_properties").AtSliceIndex(0).AtMapKey("data_lake_access_properties"), knownvalue.ListSizeExact(1)),
				},
			},
		},
	})
}

func testAccCatalogDataSource_federatedCatalog_mySQL(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	dataSourceName := "data.aws_glue_catalog.test"
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
				Config: testAccCatalogDataSourceConfig_federatedCatalog_mySQL(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrName, resourceName, names.AttrName),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrARN, resourceName, names.AttrARN),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("federated_catalog"), knownvalue.ListSizeExact(1)),
				},
			},
		},
	})
}

// TestAccGlueCatalogDataSource_targetRedshiftCatalog is intentionally serial
// (resource.Test rather than acctest.ParallelTest): the producer catalog uses
// data_lake_access_properties, which requires the caller to be a Lake
// Formation admin, and the config manages aws_lakeformation_data_lake_settings
// — an account/region-wide singleton that collides under parallel execution.
func testAccCatalogDataSource_targetRedshiftCatalog(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	dataSourceName := "data.aws_glue_catalog.test"
	resourceName := "aws_glue_catalog.test"

	resource.Test(t, resource.TestCase{
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
				Config: testAccCatalogDataSourceConfig_targetRedshiftCatalog(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrName, resourceName, names.AttrName),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrARN, resourceName, names.AttrARN),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("target_redshift_catalog"), knownvalue.ListSizeExact(1)),
				},
			},
		},
	})
}

// TestAccGlueCatalogDataSource_federatedCatalog_s3Tables is serial
// (resource.Test, not ParallelTest): AWS requires the catalog name to be the
// reserved value "s3tablescatalog", which is account/region-wide.
func testAccCatalogDataSource_federatedCatalog_s3Tables(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	dataSourceName := "data.aws_glue_catalog.test"
	resourceName := "aws_glue_catalog.test"

	resource.Test(t, resource.TestCase{
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
				Config: testAccCatalogDataSourceConfig_federatedCatalog_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrName, resourceName, names.AttrName),
					resource.TestCheckResourceAttrPair(dataSourceName, names.AttrARN, resourceName, names.AttrARN),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New(names.AttrName), knownvalue.StringExact("s3tablescatalog")),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("federated_catalog"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("federated_catalog").AtSliceIndex(0).AtMapKey("connection_name"), knownvalue.StringExact("aws:s3tables")),
				},
			},
		},
	})
}

// --- Config functions ---

func testAccCatalogDataSourceConfig_catalogPropertiesDataLakeAccess(rName string) string {
	return acctest.ConfigCompose(
		testAccCatalogConfig_catalogPropertiesDataLakeAccess(rName),
		`
data "aws_glue_catalog" "test" {
  name = aws_glue_catalog.test.name
}
`,
	)
}

func testAccCatalogDataSourceConfig_federatedCatalog_mySQL(rName string) string {
	return acctest.ConfigCompose(
		testAccCatalogConfig_federatedCatalog_mySQL(rName),
		`
data "aws_glue_catalog" "test" {
  name = aws_glue_catalog.test.name
}
`,
	)
}

func testAccCatalogDataSourceConfig_targetRedshiftCatalog(rName string) string {
	return acctest.ConfigCompose(
		testAccCatalogConfig_targetRedshiftCatalog(rName),
		`
data "aws_glue_catalog" "test" {
  name = aws_glue_catalog.test.name
}
`,
	)
}

func testAccCatalogDataSourceConfig_federatedCatalog_s3Tables(rName string) string {
	return acctest.ConfigCompose(
		testAccCatalogConfig_federatedCatalog_s3Tables(rName),
		`
data "aws_glue_catalog" "test" {
  name = aws_glue_catalog.test.name
}
`,
	)
}
