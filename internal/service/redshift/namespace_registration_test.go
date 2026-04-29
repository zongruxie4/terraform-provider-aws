// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package redshift_test

import (
	"context"
	"fmt"
	"testing"

	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfredshift "github.com/hashicorp/terraform-provider-aws/internal/service/redshift"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccRedshiftNamespaceRegistration_basic(t *testing.T) {
	ctx := acctest.Context(t)
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_redshift_namespace_registration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.RedshiftServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckNamespaceRegistrationDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccNamespaceRegistrationConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNamespaceRegistrationExists(ctx, t, resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testAccCheckNamespaceRegistrationDestroy(ctx context.Context, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).RedshiftClient(ctx)
		serverlessConn := acctest.Provider.Meta().(*conns.AWSClient).RedshiftServerlessClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_redshift_namespace_registration" {
				continue
			}

			consumerIdentifier := rs.Primary.Attributes["consumer_identifier"]
			namespaceType := rs.Primary.Attributes["namespace_type"]
			serverlessNamespaceIdentifier := rs.Primary.Attributes["serverless_namespace_identifier"]
			serverlessWorkgroupIdentifier := rs.Primary.Attributes["serverless_workgroup_identifier"]
			provisionedClusterIdentifier := rs.Primary.Attributes["provisioned_cluster_identifier"]

			_, err := tfredshift.FindNamespaceRegistrationByID(ctx, conn, serverlessConn, consumerIdentifier, namespaceType, serverlessNamespaceIdentifier, serverlessWorkgroupIdentifier, provisionedClusterIdentifier)

			if retry.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("Redshift Namespace Registration %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckNamespaceRegistrationExists(ctx context.Context, t *testing.T, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).RedshiftClient(ctx)
		serverlessConn := acctest.Provider.Meta().(*conns.AWSClient).RedshiftServerlessClient(ctx)

		// Extract parameters from state
		consumerIdentifier := rs.Primary.Attributes["consumer_identifier"]
		namespaceType := rs.Primary.Attributes["namespace_type"]
		serverlessNamespaceIdentifier := rs.Primary.Attributes["serverless_namespace_identifier"]
		serverlessWorkgroupIdentifier := rs.Primary.Attributes["serverless_workgroup_identifier"]
		provisionedClusterIdentifier := rs.Primary.Attributes["provisioned_cluster_identifier"]

		_, err := tfredshift.FindNamespaceRegistrationByID(ctx, conn, serverlessConn, consumerIdentifier, namespaceType, serverlessNamespaceIdentifier, serverlessWorkgroupIdentifier, provisionedClusterIdentifier)

		return err
	}
}

func testAccNamespaceRegistrationConfig_basic(rName string) string {
	return fmt.Sprintf(`
data "aws_caller_identity" "current" {}

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
  serverless_namespace_identifier = aws_redshiftserverless_namespace.test.namespace_name
  serverless_workgroup_identifier = aws_redshiftserverless_workgroup.test.workgroup_name
}
`, rName)
}
