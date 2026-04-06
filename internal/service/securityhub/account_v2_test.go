// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package securityhub_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfsecurityhub "github.com/hashicorp/terraform-provider-aws/internal/service/securityhub"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func testAccV2Account_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var hub securityhub.DescribeSecurityHubV2Output
	resourceName := "aws_securityhub_v2_account.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckV2AccountDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccV2AccountConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckV2AccountExists(ctx, t, resourceName, &hub),
					resource.TestCheckResourceAttrSet(resourceName, "hub_arn"),
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

func testAccV2Account_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var hub securityhub.DescribeSecurityHubV2Output
	resourceName := "aws_securityhub_v2_account.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckV2AccountDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccV2AccountConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckV2AccountExists(ctx, t, resourceName, &hub),
					acctest.CheckFrameworkResourceDisappears(ctx, t, tfsecurityhub.ResourceV2Account, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccV2Account_tags(t *testing.T) {
	ctx := acctest.Context(t)
	var hub securityhub.DescribeSecurityHubV2Output
	resourceName := "aws_securityhub_v2_account.test"

	acctest.Test(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckV2AccountDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccV2AccountConfig_tags1("key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckV2AccountExists(ctx, t, resourceName, &hub),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccV2AccountConfig_tags2("key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckV2AccountExists(ctx, t, resourceName, &hub),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccV2AccountConfig_tags1("key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckV2AccountExists(ctx, t, resourceName, &hub),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func testAccCheckV2AccountExists(ctx context.Context, t *testing.T, n string, v *securityhub.DescribeSecurityHubV2Output) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		awsClient := acctest.ProviderMeta(ctx, t)
		conn := awsClient.SecurityHubClient(ctx)

		output, err := tfsecurityhub.FindV2Account(ctx, conn)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccCheckV2AccountDestroy(ctx context.Context, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		awsClient := acctest.ProviderMeta(ctx, t)
		conn := awsClient.SecurityHubClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_securityhub_v2_account" {
				continue
			}

			_, err := tfsecurityhub.FindV2Account(ctx, conn)

			if retry.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("Security Hub V2 Account %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

const testAccV2AccountConfig_basic = `
resource "aws_securityhub_v2_account" "test" {}
`

func testAccV2AccountConfig_tags1(tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_securityhub_v2_account" "test" {
  tags = {
    %[1]q = %[2]q
  }
}
`, tagKey1, tagValue1)
}

func testAccV2AccountConfig_tags2(tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_securityhub_v2_account" "test" {
  tags = {
    %[1]q = %[2]q
    %[3]q = %[4]q
  }
}
`, tagKey1, tagValue1, tagKey2, tagValue2)
}
