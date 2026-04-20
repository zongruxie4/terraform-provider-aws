// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package glue_test

import (
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccGlueCatalogDataSource_basic(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	dataSourceName := "data.aws_glue_catalog.test"

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
				Config: testAccCatalogDataSourceConfig_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, names.AttrName, "s3tablescatalog"),
					resource.TestCheckResourceAttrSet(dataSourceName, names.AttrCatalogID),
					resource.TestCheckResourceAttr(dataSourceName, names.AttrDescription, "Test S3 Tables federated catalog"),
					resource.TestCheckResourceAttr(dataSourceName, "federated_catalog.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "federated_catalog.0.identifier"),
					resource.TestCheckResourceAttr(dataSourceName, "federated_catalog.0.connection_name", "aws:s3tables"),
					acctest.MatchResourceAttrRegionalARN(ctx, dataSourceName, names.AttrARN, "glue", regexache.MustCompile(`catalog/.+$`)),
				),
			},
		},
	})
}

func TestAccGlueCatalogDataSource_s3Tables(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	dataSourceName := "data.aws_glue_catalog.test"

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
				Config: testAccCatalogDataSourceConfig_s3Tables(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, names.AttrName, "s3tablescatalog"),
					resource.TestCheckResourceAttrSet(dataSourceName, names.AttrCatalogID),
					resource.TestCheckResourceAttr(dataSourceName, names.AttrDescription, "Test S3 Tables federated catalog"),
					resource.TestCheckResourceAttr(dataSourceName, "federated_catalog.#", "1"),
					resource.TestCheckResourceAttrSet(dataSourceName, "federated_catalog.0.identifier"),
					resource.TestCheckResourceAttr(dataSourceName, "federated_catalog.0.connection_name", "aws:s3tables"),
					acctest.MatchResourceAttrRegionalARN(ctx, dataSourceName, names.AttrARN, "glue", regexache.MustCompile(`catalog/.+$`)),
				),
			},
		},
	})
}

func testAccCatalogDataSourceConfig_s3Tables(rName string) string {
	return fmt.Sprintf(`
%[1]s

data "aws_glue_catalog" "test" {
  name = aws_glue_catalog.test.name
}
`, testAccCatalogConfig_s3Tables(rName))
}
