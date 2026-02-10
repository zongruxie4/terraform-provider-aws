// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package ec2_test

import (
	"context"
	"fmt"
	"testing"

	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccEC2SecondaryNetwork_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var secondaryNetwork awstypes.SecondaryNetwork
	resourceName := "aws_secondary_network.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSecondaryNetworkDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccSecondaryNetworkConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecondaryNetworkExists(ctx, resourceName, &secondaryNetwork),
					resource.TestCheckResourceAttr(resourceName, "ipv4_cidr_block", "10.0.0.0/16"),
					resource.TestCheckResourceAttr(resourceName, "network_type", "rdma"),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrARN),
					resource.TestCheckResourceAttrSet(resourceName, "owner_id"),
					resource.TestCheckResourceAttrSet(resourceName, "secondary_network_id"),
					resource.TestCheckResourceAttr(resourceName, names.AttrState, tfec2.SecondaryNetworkStateCreateComplete),
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

func TestAccEC2SecondaryNetwork_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var secondaryNetwork awstypes.SecondaryNetwork
	resourceName := "aws_secondary_network.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSecondaryNetworkDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccSecondaryNetworkConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecondaryNetworkExists(ctx, resourceName, &secondaryNetwork),
					acctest.CheckFrameworkResourceDisappears(ctx, t, tfec2.ResourceSecondaryNetwork, resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2SecondaryNetwork_tags(t *testing.T) {
	ctx := acctest.Context(t)
	var secondaryNetwork awstypes.SecondaryNetwork
	resourceName := "aws_secondary_network.test"
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSecondaryNetworkDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccSecondaryNetworkConfig_tags1(rName, "key1", "value1"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecondaryNetworkExists(ctx, resourceName, &secondaryNetwork),
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
				Config: testAccSecondaryNetworkConfig_tags2(rName, "key1", "value1updated", "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecondaryNetworkExists(ctx, resourceName, &secondaryNetwork),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.key1", "value1updated"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
			{
				Config: testAccSecondaryNetworkConfig_tags1(rName, "key2", "value2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSecondaryNetworkExists(ctx, resourceName, &secondaryNetwork),
					resource.TestCheckResourceAttr(resourceName, "tags.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "tags.key2", "value2"),
				),
			},
		},
	})
}

func testAccCheckSecondaryNetworkExists(ctx context.Context, n string, v *awstypes.SecondaryNetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Client(ctx)

		output, err := tfec2.FindSecondaryNetworkResourceByID(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccCheckSecondaryNetworkDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).EC2Client(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_secondary_network" {
				continue
			}

			_, err := tfec2.FindSecondaryNetworkResourceByID(ctx, conn, rs.Primary.ID)

			if retry.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("EC2 Secondary Network %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccSecondaryNetworkConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_secondary_network" "test" {
  ipv4_cidr_block = "10.0.0.0/16"
  network_type    = "rdma"

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccSecondaryNetworkConfig_tags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_secondary_network" "test" {
  ipv4_cidr_block = "10.0.0.0/16"
  network_type    = "rdma"

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccSecondaryNetworkConfig_tags2(rName, tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return fmt.Sprintf(`
resource "aws_secondary_network" "test" {
  ipv4_cidr_block = "10.0.0.0/16"
  network_type    = "rdma"

  tags = {
    %[2]q = %[3]q
    %[4]q = %[5]q
  }
}
`, rName, tagKey1, tagValue1, tagKey2, tagValue2)
}
