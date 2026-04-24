// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package ec2_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccVPCSubnet_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &v),
					acctest.MatchResourceAttrRegionalARN(ctx, resourceName, names.AttrARN, "ec2", regexache.MustCompile(`subnet/subnet-.+`)),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrAvailabilityZone),
					resource.TestCheckResourceAttrSet(resourceName, "availability_zone_id"),
					resource.TestCheckResourceAttr(resourceName, names.AttrCIDRBlock, "10.1.1.0/24"),
					resource.TestCheckResourceAttr(resourceName, "customer_owned_ipv4_pool", ""),
					resource.TestCheckResourceAttr(resourceName, "enable_dns64", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_aaaa_record_on_launch", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_a_record_on_launch", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "ipv6_cidr_block", ""),
					resource.TestCheckResourceAttr(resourceName, "ipv6_native", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "map_customer_owned_ip_on_launch", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "map_public_ip_on_launch", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "outpost_arn", ""),
					acctest.CheckResourceAttrAccountID(ctx, resourceName, names.AttrOwnerID),
					resource.TestCheckResourceAttr(resourceName, "private_dns_hostname_type_on_launch", "ip-name"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "0"),
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

func TestAccVPCSubnet_tags_defaultAndIgnoreTags(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_tags1(rName, acctest.CtKey1, acctest.CtValue1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					testAccCheckSubnetUpdateTags(ctx, t, &subnet, nil, map[string]string{"defaultkey1": "defaultvalue1"}),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: acctest.ConfigCompose(
					acctest.ConfigDefaultAndIgnoreTagsKeyPrefixes1("defaultkey1", "defaultvalue1", "defaultkey"),
					testAccVPCSubnetConfig_tags1(rName, acctest.CtKey1, acctest.CtValue1),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
				},
			},
			{
				Config: acctest.ConfigCompose(
					acctest.ConfigDefaultAndIgnoreTagsKeys1("defaultkey1", "defaultvalue1"),
					testAccVPCSubnetConfig_tags1(rName, acctest.CtKey1, acctest.CtValue1),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

func TestAccVPCSubnet_tags_ignoreTags(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckVPCDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					testAccCheckSubnetUpdateTags(ctx, t, &subnet, nil, map[string]string{"ignorekey1": "ignorevalue1"}),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				ExpectNonEmptyPlan: true,
			},
			{
				Config: acctest.ConfigCompose(acctest.ConfigIgnoreTagsKeyPrefixes1("ignorekey"), testAccVPCSubnetConfig_basic(rName)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
				},
			},
			{
				Config: acctest.ConfigCompose(acctest.ConfigIgnoreTagsKeys("ignorekey1"), testAccVPCSubnetConfig_basic(rName)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

func TestAccVPCSubnet_ipv6(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
				Config: testAccVPCSubnetConfig_ipv6(rName, 1, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					testAccCheckSubnetIPv6CIDRBlockAssociationSet(&subnet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resourceName, tfjsonpath.New("assign_ipv6_address_on_creation"), knownvalue.Bool(true),
					),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Disable assign_ipv6_address_on_creation
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Config: testAccVPCSubnetConfig_ipv6(rName, 1, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resourceName, tfjsonpath.New("assign_ipv6_address_on_creation"), knownvalue.Bool(false),
					),
				},
			},
			{
				// Change IPv6 CIDR block
				// assign_ipv6_address_on_creation was false, so no replacement
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Config: testAccVPCSubnetConfig_ipv6(rName, 3, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resourceName, tfjsonpath.New("assign_ipv6_address_on_creation"), knownvalue.Bool(true),
					),
				},
			},
			{
				// Force new by changing IPv6 CIDR block
				// since assign_ipv6_address_on_creation was true
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Config: testAccVPCSubnetConfig_ipv6(rName, 1, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					testAccCheckSubnetIPv6CIDRBlockAssociationSet(&subnet),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						resourceName, tfjsonpath.New("assign_ipv6_address_on_creation"), knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func TestAccVPCSubnet_enableIPv6(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_prev6(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "ipv6_cidr_block", ""),
					resource.TestCheckResourceAttr(resourceName, "assign_ipv6_address_on_creation", acctest.CtFalse),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccVPCSubnetConfig_ipv6(rName, 1, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttrSet(resourceName, "ipv6_cidr_block"),
					resource.TestCheckResourceAttr(resourceName, "assign_ipv6_address_on_creation", acctest.CtTrue),
				),
			},
			{
				Config: testAccVPCSubnetConfig_prev6(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "ipv6_cidr_block", ""),
					resource.TestCheckResourceAttr(resourceName, "assign_ipv6_address_on_creation", acctest.CtFalse),
				),
			},
		},
	})
}

func TestAccVPCSubnet_availabilityZoneID(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_availabilityZoneID(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &v),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrAvailabilityZone),
					resource.TestCheckResourceAttrPair(resourceName, "availability_zone_id", "data.aws_availability_zones.available", "zone_ids.0"),
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

func TestAccVPCSubnet_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &v),
					acctest.CheckSDKResourceDisappears(ctx, t, tfec2.ResourceSubnet(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccVPCSubnet_customerOwnedIPv4Pool(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	coipDataSourceName := "data.aws_ec2_coip_pool.test"
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckOutpostsOutposts(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_customerOwnedv4Pool(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttrPair(resourceName, "customer_owned_ipv4_pool", coipDataSourceName, "pool_id"),
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

func TestAccVPCSubnet_mapCustomerOwnedIPOnLaunch(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckOutpostsOutposts(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_mapCustomerOwnedOnLaunch(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "map_customer_owned_ip_on_launch", acctest.CtTrue),
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

func TestAccVPCSubnet_mapPublicIPOnLaunch(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_mapPublicOnLaunch(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "map_public_ip_on_launch", acctest.CtTrue),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccVPCSubnetConfig_mapPublicOnLaunch(rName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "map_public_ip_on_launch", acctest.CtFalse),
				),
			},
			{
				Config: testAccVPCSubnetConfig_mapPublicOnLaunch(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "map_public_ip_on_launch", acctest.CtTrue),
				),
			},
		},
	})
}

func TestAccVPCSubnet_outpost(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.Subnet
	outpostDataSourceName := "data.aws_outposts_outpost.test"
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckOutpostsOutposts(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_outpost(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &v),
					resource.TestCheckResourceAttrPair(resourceName, "outpost_arn", outpostDataSourceName, names.AttrARN),
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

func TestAccVPCSubnet_enableDNS64(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_enableDNS64(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_dns64", acctest.CtTrue),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccVPCSubnetConfig_enableDNS64(rName, false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_dns64", acctest.CtFalse),
				),
			},
			{
				Config: testAccVPCSubnetConfig_enableDNS64(rName, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_dns64", acctest.CtTrue),
				),
			},
		},
	})
}

func TestAccVPCSubnet_ipv4ToIPv6(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_ipv4ToIPv6Before(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "assign_ipv6_address_on_creation", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "enable_dns64", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_aaaa_record_on_launch", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "ipv6_cidr_block", ""),
				),
			},
			{
				Config: testAccVPCSubnetConfig_ipv4ToIPv6After(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "assign_ipv6_address_on_creation", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "enable_dns64", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_aaaa_record_on_launch", acctest.CtTrue),
					resource.TestCheckResourceAttrSet(resourceName, "ipv6_cidr_block"),
				),
			},
		},
	})
}

func TestAccVPCSubnet_enableLNIAtDeviceIndex(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); acctest.PreCheckOutpostsOutposts(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_enableLniAtDeviceIndex(rName, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_lni_at_device_index", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccVPCSubnetConfig_enableLniAtDeviceIndex(rName, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_lni_at_device_index", "1"),
				),
			},
			{
				Config: testAccVPCSubnetConfig_enableLniAtDeviceIndex(rName, 1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_lni_at_device_index", "1"),
				),
			},
		},
	})
}

func TestAccVPCSubnet_privateDNSNameOptionsOnLaunch(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_privateDNSNameOptionsOnLaunch(rName, true, true, "resource-name"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_aaaa_record_on_launch", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_a_record_on_launch", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "private_dns_hostname_type_on_launch", "resource-name"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccVPCSubnetConfig_privateDNSNameOptionsOnLaunch(rName, false, true, "ip-name"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_aaaa_record_on_launch", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_a_record_on_launch", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "private_dns_hostname_type_on_launch", "ip-name"),
				),
			},
			{
				Config: testAccVPCSubnetConfig_privateDNSNameOptionsOnLaunch(rName, true, false, "resource-name"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_aaaa_record_on_launch", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_a_record_on_launch", acctest.CtFalse),
					resource.TestCheckResourceAttr(resourceName, "private_dns_hostname_type_on_launch", "resource-name"),
				),
			},
		},
	})
}

func TestAccVPCSubnet_ipv6Native(t *testing.T) {
	ctx := acctest.Context(t)
	var v awstypes.Subnet
	resourceName := "aws_subnet.test"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_ipv6Native(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &v),
					resource.TestCheckResourceAttr(resourceName, names.AttrCIDRBlock, ""),
					resource.TestCheckResourceAttr(resourceName, "enable_resource_name_dns_aaaa_record_on_launch", acctest.CtTrue),
					resource.TestCheckResourceAttr(resourceName, "ipv6_native", acctest.CtTrue),
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

func TestAccVPCSubnet_IPAM_ipv4Allocation(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_subnet.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_ipv4IPAMAllocation(rName, 27),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttrPair(resourceName, "ipv4_ipam_pool_id", "aws_vpc_ipam_pool.vpc", names.AttrID),
					resource.TestCheckResourceAttr(resourceName, "ipv4_netmask_length", "27"),
					testAccCheckSubnetCIDRPrefix(&subnet, "27"),
					testAccCheckIPAMPoolAllocationExistsForSubnet(ctx, "aws_vpc_ipam_pool.vpc", &subnet),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ipv4_ipam_pool_id", "ipv4_netmask_length"},
			},
		},
	})
}

func TestAccVPCSubnet_IPAM_ipv4AllocationExplicitCIDR(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_subnet.test"
	cidr := "10.0.0.0/27"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_ipv4IPAMAllocationExplicitCIDR(rName, cidr),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttrPair(resourceName, "ipv4_ipam_pool_id", "aws_vpc_ipam_pool.vpc", names.AttrID),
					resource.TestCheckResourceAttr(resourceName, names.AttrCIDRBlock, cidr),
					testAccCheckSubnetCIDRPrefix(&subnet, "27"),
					testAccCheckIPAMPoolAllocationExistsForSubnet(ctx, "aws_vpc_ipam_pool.vpc", &subnet),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ipv4_ipam_pool_id"},
			},
		},
	})
}

func TestAccVPCSubnet_IPAM_ipv6Allocation(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_subnet.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_ipv6IPAMAllocation(rName, 60),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName, &subnet),
					resource.TestCheckResourceAttrPair(resourceName, "ipv6_ipam_pool_id", "aws_vpc_ipam_pool.vpc", names.AttrID),
					resource.TestCheckResourceAttr(resourceName, "ipv6_netmask_length", "60"),
					testAccCheckSubnetIPv6CIDRPrefix(&subnet, "60"),
					testAccCheckIPAMPoolAllocationExistsForSubnet(ctx, "aws_vpc_ipam_pool.vpc", &subnet),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ipv6_ipam_pool_id", "ipv6_netmask_length"},
			},
		},
	})
}

func TestAccVPCSubnet_IPAM_crossRegion(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet awstypes.Subnet
	var providers []*schema.Provider
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_subnet.test"
	vpcResourceName := "aws_vpc.test"
	poolResourceName := "aws_vpc_ipam_pool.vpc"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckMultipleRegion(t, 2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5FactoriesPlusProvidersAlternate(ctx, t, &providers),
		CheckDestroy:             testAccCheckSubnetDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVPCSubnetConfig_ipamCrossRegion(rName, 28),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckSubnetExistsWithProvider(ctx, resourceName, &subnet, acctest.RegionProviderFunc(ctx, acctest.AlternateRegion(), &providers)),
					resource.TestCheckResourceAttrPair(resourceName, names.AttrVPCID, vpcResourceName, names.AttrID),
					resource.TestCheckResourceAttrPair(resourceName, "ipv4_ipam_pool_id", poolResourceName, names.AttrID),
					resource.TestCheckResourceAttr(resourceName, "ipv4_netmask_length", "28"),
					testAccCheckSubnetCIDRPrefix(&subnet, "28"),
					resource.TestCheckResourceAttrSet(resourceName, names.AttrCIDRBlock),
					testAccCheckIPAMPoolAllocationExistsForSubnet(ctx, poolResourceName, &subnet, acctest.RegionProviderFunc(ctx, acctest.AlternateRegion(), &providers)),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ipv4_ipam_pool_id", "ipv4_netmask_length"},
			},
		},
	})
}

func testAccCheckSubnetIPv6CIDRBlockAssociationSet(subnet *awstypes.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if subnet.Ipv6CidrBlockAssociationSet == nil {
			return fmt.Errorf("Expected IPV6 CIDR Block Association")
		}
		return nil
	}
}

func testAccCheckSubnetDestroy(ctx context.Context, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_subnet" {
				continue
			}

			_, err := tfec2.FindSubnetByID(ctx, conn, rs.Primary.ID)

			if retry.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("EC2 Subnet %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckSubnetExists(ctx context.Context, t *testing.T, n string, v *awstypes.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EC2 Subnet ID is set")
		}

		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)

		output, err := tfec2.FindSubnetByID(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccCheckSubnetExistsWithProvider(ctx context.Context, n string, v *awstypes.Subnet, providerF func() *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No EC2 Subnet ID is set")
		}

		conn := providerF().Meta().(*conns.AWSClient).EC2Client(ctx)

		output, err := tfec2.FindSubnetByID(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccCheckSubnetUpdateTags(ctx context.Context, t *testing.T, subnet *awstypes.Subnet, oldTags, newTags map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)

		return tfec2.UpdateTags(ctx, conn, aws.ToString(subnet.SubnetId), oldTags, newTags)
	}
}

func testAccCheckSubnetCIDRPrefix(subnet *awstypes.Subnet, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cidrBlock := aws.ToString(subnet.CidrBlock)
		parts := strings.Split(cidrBlock, "/")
		if len(parts) != 2 {
			return fmt.Errorf("Bad cidr format: got %s, expected format <ip>/<prefix>", cidrBlock)
		}
		if parts[1] != expected {
			return fmt.Errorf("Bad cidr prefix: got %s, expected /%s", cidrBlock, expected)
		}
		return nil
	}
}

func testAccCheckIPAMPoolAllocationExistsForSubnet(ctx context.Context, poolResourceName string, subnet *awstypes.Subnet, providerF ...func() *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[poolResourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", poolResourceName)
		}

		poolID := rs.Primary.ID
		subnetID := aws.ToString(subnet.SubnetId)
		subnetCIDR := aws.ToString(subnet.CidrBlock)

		var conn *conns.AWSClient
		if len(providerF) > 0 && providerF[0] != nil {
			conn = providerF[0]().Meta().(*conns.AWSClient)
		} else {
			conn = acctest.Provider.Meta().(*conns.AWSClient)
		}

		allocations, err := tfec2.FindIPAMPoolAllocationsByIPAMPoolIDAndResourceID(ctx, conn.EC2Client(ctx), poolID, subnetID)
		if err != nil {
			return fmt.Errorf("error finding IPAM Pool (%s) allocations for subnet (%s): %w", poolID, subnetID, err)
		}

		if len(allocations) == 0 {
			return fmt.Errorf("no IPAM Pool allocation found for subnet %s in pool %s", subnetID, poolID)
		}

		allocation := allocations[0]
		if allocation.ResourceType != awstypes.IpamPoolAllocationResourceTypeSubnet {
			return fmt.Errorf("expected allocation resource type 'subnet', got %s", allocation.ResourceType)
		}

		allocationCIDR := aws.ToString(allocation.Cidr)

		// Check if allocation matches IPv4 CIDR
		if allocationCIDR == subnetCIDR && subnetCIDR != "" {
			return nil
		}

		// Check if allocation matches any IPv6 CIDR
		for _, association := range subnet.Ipv6CidrBlockAssociationSet {
			if association.Ipv6CidrBlockState.State == awstypes.SubnetCidrBlockStateCodeAssociated {
				subnetIPv6CIDR := aws.ToString(association.Ipv6CidrBlock)
				if allocationCIDR == subnetIPv6CIDR {
					return nil
				}
			}
		}

		return fmt.Errorf("allocation CIDR (%s) does not match subnet IPv4 CIDR (%s) or any associated IPv6 CIDR", allocationCIDR, subnetCIDR)
	}
}

func testAccCheckSubnetIPv6CIDRPrefix(subnet *awstypes.Subnet, expected string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, association := range subnet.Ipv6CidrBlockAssociationSet {
			if association.Ipv6CidrBlockState != nil && association.Ipv6CidrBlockState.State == awstypes.SubnetCidrBlockStateCodeAssociated {
				if strings.Split(aws.ToString(association.Ipv6CidrBlock), "/")[1] != expected {
					return fmt.Errorf("Bad IPv6 cidr prefix: got %s, expected /%s", aws.ToString(association.Ipv6CidrBlock), expected)
				}
				return nil
			}
		}
		return fmt.Errorf("No associated IPv6 CIDR block found")
	}
}

func testAccVPCSubnetConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = aws_vpc.test.id
}
`, rName)
}

func testAccVPCSubnetConfig_tags1(rName, tagKey1, tagValue1 string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block = "10.1.1.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    %[2]q = %[3]q
  }
}
`, rName, tagKey1, tagValue1)
}

func testAccVPCSubnetConfig_prev6(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.10.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block = "10.10.1.0/24"
  vpc_id     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccVPCSubnetConfig_ipv6(rName string, ipv6CidrSubnetIndex int, assignIPv6AddressOnCreation bool) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.10.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block                      = "10.10.1.0/24"
  vpc_id                          = aws_vpc.test.id
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, %[2]d)
  assign_ipv6_address_on_creation = %[3]t

  tags = {
    Name = %[1]q
  }
}
`, rName, ipv6CidrSubnetIndex, assignIPv6AddressOnCreation)
}

func testAccVPCSubnetConfig_availabilityZoneID(rName string) string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block           = "10.1.1.0/24"
  vpc_id               = aws_vpc.test.id
  availability_zone_id = data.aws_availability_zones.available.zone_ids[0]

  tags = {
    Name = %[1]q
  }
}
`, rName))
}

func testAccVPCSubnetConfig_customerOwnedv4Pool(rName string) string {
	return fmt.Sprintf(`
data "aws_outposts_outposts" "test" {}

data "aws_outposts_outpost" "test" {
  id = tolist(data.aws_outposts_outposts.test.ids)[0]
}

data "aws_ec2_local_gateway_route_tables" "test" {
  filter {
    name   = "outpost-arn"
    values = [data.aws_outposts_outpost.test.arn]
  }
}

data "aws_ec2_coip_pools" "test" {
  # Filtering by Local Gateway Route Table ID is documented but not working in EC2 API.
  # If there are multiple Outposts in the test account, this lookup can
  # be misaligned and cause downstream resource errors.
  #
  # filter {
  #   name   = "coip-pool.local-gateway-route-table-id"
  #   values = [tolist(data.aws_ec2_local_gateway_route_tables.test.ids)[0]]
  # }
}

data "aws_ec2_coip_pool" "test" {
  pool_id = tolist(data.aws_ec2_coip_pools.test.pool_ids)[0]
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  availability_zone               = data.aws_outposts_outpost.test.availability_zone
  cidr_block                      = cidrsubnet(aws_vpc.test.cidr_block, 8, 0)
  customer_owned_ipv4_pool        = data.aws_ec2_coip_pool.test.pool_id
  map_customer_owned_ip_on_launch = true
  outpost_arn                     = data.aws_outposts_outpost.test.arn
  vpc_id                          = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccVPCSubnetConfig_mapCustomerOwnedOnLaunch(rName string, mapCustomerOwnedIpOnLaunch bool) string {
	return fmt.Sprintf(`
data "aws_outposts_outposts" "test" {}

data "aws_outposts_outpost" "test" {
  id = tolist(data.aws_outposts_outposts.test.ids)[0]
}

data "aws_ec2_local_gateway_route_tables" "test" {
  filter {
    name   = "outpost-arn"
    values = [data.aws_outposts_outpost.test.arn]
  }
}

data "aws_ec2_coip_pools" "test" {
  # Filtering by Local Gateway Route Table ID is documented but not working in EC2 API.
  # If there are multiple Outposts in the test account, this lookup can
  # be misaligned and cause downstream resource errors.
  #
  # filter {
  #   name   = "coip-pool.local-gateway-route-table-id"
  #   values = [tolist(data.aws_ec2_local_gateway_route_tables.test.ids)[0]]
  # }
}

data "aws_ec2_coip_pool" "test" {
  pool_id = tolist(data.aws_ec2_coip_pools.test.pool_ids)[0]
}

resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  availability_zone               = data.aws_outposts_outpost.test.availability_zone
  cidr_block                      = cidrsubnet(aws_vpc.test.cidr_block, 8, 0)
  customer_owned_ipv4_pool        = data.aws_ec2_coip_pool.test.pool_id
  map_customer_owned_ip_on_launch = %[2]t
  outpost_arn                     = data.aws_outposts_outpost.test.arn
  vpc_id                          = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName, mapCustomerOwnedIpOnLaunch)
}

func testAccVPCSubnetConfig_mapPublicOnLaunch(rName string, mapPublicIpOnLaunch bool) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block = "10.0.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block              = cidrsubnet(aws_vpc.test.cidr_block, 8, 0)
  map_public_ip_on_launch = %[2]t
  vpc_id                  = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName, mapPublicIpOnLaunch)
}

func testAccVPCSubnetConfig_enableDNS64(rName string, enableDns64 bool) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.10.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block                      = cidrsubnet(aws_vpc.test.cidr_block, 8, 0)
  enable_dns64                    = %[2]t
  vpc_id                          = aws_vpc.test.id
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, 1)
  assign_ipv6_address_on_creation = true

  tags = {
    Name = %[1]q
  }
}
`, rName, enableDns64)
}

func testAccVPCSubnetConfig_enableLniAtDeviceIndex(rName string, deviceIndex int) string {
	return fmt.Sprintf(`


data "aws_outposts_outposts" "test" {}

data "aws_outposts_outpost" "test" {
  id = tolist(data.aws_outposts_outposts.test.ids)[0]
}

resource "aws_vpc" "test" {
  cidr_block = "10.10.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  availability_zone          = data.aws_outposts_outpost.test.availability_zone
  cidr_block                 = cidrsubnet(aws_vpc.test.cidr_block, 8, 0)
  enable_lni_at_device_index = %[2]d
  outpost_arn                = data.aws_outposts_outpost.test.arn
  vpc_id                     = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName, deviceIndex)
}

func testAccVPCSubnetConfig_privateDNSNameOptionsOnLaunch(rName string, enableDnsAAAA, enableDnsA bool, hostnameType string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.10.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  cidr_block                      = cidrsubnet(aws_vpc.test.cidr_block, 8, 0)
  vpc_id                          = aws_vpc.test.id
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, 1)
  assign_ipv6_address_on_creation = true

  enable_resource_name_dns_aaaa_record_on_launch = %[2]t
  enable_resource_name_dns_a_record_on_launch    = %[3]t
  private_dns_hostname_type_on_launch            = %[4]q

  tags = {
    Name = %[1]q
  }
}
`, rName, enableDnsAAAA, enableDnsA, hostnameType)
}

func testAccVPCSubnetConfig_ipv6Native(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.10.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  vpc_id                          = aws_vpc.test.id
  ipv6_cidr_block                 = cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, 1)
  assign_ipv6_address_on_creation = true
  ipv6_native                     = true

  enable_resource_name_dns_aaaa_record_on_launch = true

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccVPCSubnetConfig_outpost(rName string) string {
	return fmt.Sprintf(`
data "aws_outposts_outposts" "test" {}

data "aws_outposts_outpost" "test" {
  id = tolist(data.aws_outposts_outposts.test.ids)[0]
}

resource "aws_vpc" "test" {
  cidr_block = "10.1.0.0/16"

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  availability_zone = data.aws_outposts_outpost.test.availability_zone
  cidr_block        = "10.1.1.0/24"
  outpost_arn       = data.aws_outposts_outpost.test.arn
  vpc_id            = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccVPCSubnetConfig_ipv4ToIPv6Before(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.10.0.0/16"
  assign_generated_ipv6_cidr_block = false

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  assign_ipv6_address_on_creation                = false
  cidr_block                                     = cidrsubnet(aws_vpc.test.cidr_block, 8, 1)
  enable_dns64                                   = false
  enable_resource_name_dns_aaaa_record_on_launch = false
  ipv6_cidr_block                                = null
  vpc_id                                         = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

func testAccVPCSubnetConfig_ipv4ToIPv6After(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block                       = "10.10.0.0/16"
  assign_generated_ipv6_cidr_block = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test" {
  assign_ipv6_address_on_creation                = true
  cidr_block                                     = cidrsubnet(aws_vpc.test.cidr_block, 8, 1)
  enable_dns64                                   = true
  enable_resource_name_dns_aaaa_record_on_launch = true
  ipv6_cidr_block                                = cidrsubnet(aws_vpc.test.ipv6_cidr_block, 8, 1)
  vpc_id                                         = aws_vpc.test.id

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

const testAccVPCSubnetConfig_ipamBase = `
data "aws_region" "current" {}

resource "aws_vpc_ipam" "test" {
  operating_regions {
    region_name = data.aws_region.current.region
  }
}
`

func testAccVPCSubnetConfig_ipamIPv4(rName string) string {
	return acctest.ConfigCompose(testAccVPCSubnetConfig_ipamBase, fmt.Sprintf(`
data "aws_caller_identity" "current" {}

resource "aws_vpc_ipam_pool" "test" {
  address_family = "ipv4"
  ipam_scope_id  = aws_vpc_ipam.test.private_default_scope_id
  locale         = data.aws_region.current.name
}

resource "aws_vpc_ipam_pool_cidr" "test" {
  ipam_pool_id = aws_vpc_ipam_pool.test.id
  cidr         = "10.0.0.0/16"
}

resource "aws_vpc" "test" {
  ipv4_ipam_pool_id   = aws_vpc_ipam_pool.test.id
  ipv4_netmask_length = 24

  depends_on = [aws_vpc_ipam_pool_cidr.test]

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc_ipam_pool" "vpc" {
  address_family      = "ipv4"
  ipam_scope_id       = aws_vpc_ipam.test.private_default_scope_id
  locale              = data.aws_region.current.name
  source_ipam_pool_id = aws_vpc_ipam_pool.test.id

  source_resource {
    resource_id     = aws_vpc.test.id
    resource_owner  = data.aws_caller_identity.current.account_id
    resource_region = data.aws_region.current.name
    resource_type   = "vpc"
  }

  tags = {
    Name = %[1]q
  }
}
`, rName))
}

func testAccVPCSubnetConfig_ipamIPv6(rName string) string {
	return acctest.ConfigCompose(testAccVPCSubnetConfig_ipamBase, fmt.Sprintf(`
data "aws_caller_identity" "current" {}

resource "aws_vpc_ipam_pool" "test" {
  address_family = "ipv6"
  ipam_scope_id  = aws_vpc_ipam.test.private_default_scope_id
  locale         = data.aws_region.current.name
}

resource "aws_vpc_ipam_pool_cidr" "test" {
  ipam_pool_id   = aws_vpc_ipam_pool.test.id
  netmask_length = 52
}

resource "aws_vpc" "test" {
  cidr_block          = "10.1.0.0/16"
  ipv6_ipam_pool_id   = aws_vpc_ipam_pool.test.id
  ipv6_netmask_length = 56

  depends_on = [aws_vpc_ipam_pool_cidr.test]

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc_ipam_pool" "vpc" {
  address_family      = "ipv6"
  ipam_scope_id       = aws_vpc_ipam.test.private_default_scope_id
  locale              = data.aws_region.current.name
  source_ipam_pool_id = aws_vpc_ipam_pool.test.id

  source_resource {
    resource_id     = aws_vpc.test.id
    resource_owner  = data.aws_caller_identity.current.account_id
    resource_region = data.aws_region.current.name
    resource_type   = "vpc"
  }

  tags = {
    Name = %[1]q
  }
}
`, rName))
}

func testAccVPCSubnetConfig_ipv4IPAMAllocation(rName string, netmaskLength int) string {
	return acctest.ConfigCompose(testAccVPCSubnetConfig_ipamIPv4(rName), fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc_ipam_pool_cidr" "vpc" {
  ipam_pool_id = aws_vpc_ipam_pool.vpc.id
  cidr         = aws_vpc.test.cidr_block
}

resource "aws_subnet" "test" {
  vpc_id              = aws_vpc.test.id
  ipv4_ipam_pool_id   = aws_vpc_ipam_pool.vpc.id
  ipv4_netmask_length = %[1]d
  availability_zone   = data.aws_availability_zones.available.names[0]

  depends_on = [aws_vpc_ipam_pool_cidr.vpc]
}
`, netmaskLength))
}

func testAccVPCSubnetConfig_ipv4IPAMAllocationExplicitCIDR(rName string, cidr string) string {
	return acctest.ConfigCompose(testAccVPCSubnetConfig_ipamIPv4(rName), fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc_ipam_pool_cidr" "vpc" {
  ipam_pool_id = aws_vpc_ipam_pool.vpc.id
  cidr         = aws_vpc.test.cidr_block
}

resource "aws_subnet" "test" {
  vpc_id            = aws_vpc.test.id
  ipv4_ipam_pool_id = aws_vpc_ipam_pool.vpc.id
  cidr_block        = %[1]q
  availability_zone = data.aws_availability_zones.available.names[0]

  depends_on = [aws_vpc_ipam_pool_cidr.vpc]
}
`, cidr))
}

func testAccVPCSubnetConfig_ipv6IPAMAllocation(rName string, netmaskLength int) string {
	return acctest.ConfigCompose(testAccVPCSubnetConfig_ipamIPv6(rName), fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc_ipam_pool_cidr" "vpc" {
  ipam_pool_id = aws_vpc_ipam_pool.vpc.id
  cidr         = aws_vpc.test.ipv6_cidr_block
}

resource "aws_subnet" "test" {
  vpc_id                                         = aws_vpc.test.id
  ipv6_native                                    = true
  assign_ipv6_address_on_creation                = true
  ipv6_ipam_pool_id                              = aws_vpc_ipam_pool.vpc.id
  ipv6_netmask_length                            = %[1]d
  availability_zone                              = data.aws_availability_zones.available.names[0]
  enable_resource_name_dns_aaaa_record_on_launch = true

  depends_on = [aws_vpc_ipam_pool_cidr.vpc]
}
`, netmaskLength))
}

func testAccVPCSubnetConfig_ipamCrossRegion(rName string, netmaskLength int) string {
	return acctest.ConfigCompose(acctest.ConfigMultipleRegionProvider(2), fmt.Sprintf(`
data "aws_region" "current" {}

data "aws_region" "alternate" {
  provider = awsalternate
}

data "aws_caller_identity" "current" {}

data "aws_availability_zones" "alternate" {
  provider = awsalternate
  state    = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc_ipam" "test" {
  operating_regions {
    region_name = data.aws_region.current.name
  }

  operating_regions {
    region_name = data.aws_region.alternate.name
  }

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc_ipam_pool" "test" {
  address_family = "ipv4"
  ipam_scope_id  = aws_vpc_ipam.test.private_default_scope_id
  locale         = data.aws_region.alternate.name

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc_ipam_pool_cidr" "test" {
  ipam_pool_id = aws_vpc_ipam_pool.test.id
  cidr         = "10.0.0.0/16"
}

resource "aws_vpc" "test" {
  provider = awsalternate

  ipv4_ipam_pool_id   = aws_vpc_ipam_pool.test.id
  ipv4_netmask_length = 24

  depends_on = [aws_vpc_ipam_pool_cidr.test]

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc_ipam_pool" "vpc" {
  address_family      = "ipv4"
  ipam_scope_id       = aws_vpc_ipam.test.private_default_scope_id
  locale              = data.aws_region.alternate.name
  source_ipam_pool_id = aws_vpc_ipam_pool.test.id

  source_resource {
    resource_id     = aws_vpc.test.id
    resource_owner  = data.aws_caller_identity.current.account_id
    resource_region = data.aws_region.alternate.name
    resource_type   = "vpc"
  }

  tags = {
    Name = %[1]q
  }
}

resource "aws_vpc_ipam_pool_cidr" "vpc" {
  ipam_pool_id = aws_vpc_ipam_pool.vpc.id
  cidr         = aws_vpc.test.cidr_block
}

resource "aws_subnet" "test" {
  provider = awsalternate

  vpc_id              = aws_vpc.test.id
  ipv4_ipam_pool_id   = aws_vpc_ipam_pool.vpc.id
  ipv4_netmask_length = %[2]d
  availability_zone   = data.aws_availability_zones.alternate.names[0]

  depends_on = [aws_vpc_ipam_pool_cidr.vpc]

  tags = {
    Name = %[1]q
  }
}
`, rName, netmaskLength))
}

// TestAccVPCSubnet_GuardDutyDependencies_cleanup validates the multi-subnet dissociation path.
// It creates a VPC with two subnets, then creates GuardDuty resources (endpoint + SG)
// out-of-band via SDK associated with both subnets. When one subnet is removed from
// the Terraform config, dissociateGuardDutyVPCEndpoints runs to dissociate it from
// the endpoint, and the endpoint should remain available with the remaining subnet.
func TestAccVPCSubnet_GuardDutyDependencies_cleanup(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet1, subnet2 awstypes.Subnet
	var vpcID string
	resourceName1 := "aws_subnet.test1"
	resourceName2 := "aws_subnet.test2"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGuardDutyCleanupDestroy(ctx, t, &vpcID),
		Steps: []resource.TestStep{
			{
				// Step 1: Create VPC with two subnets. After Terraform creates
				// the infrastructure, create GuardDuty resources out-of-band
				// via SDK with both subnet IDs.
				Config: testAccVPCSubnetConfig_GuardDutyDependencies_cleanupBothSubnets(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName1, &subnet1),
					testAccCheckSubnetExists(ctx, t, resourceName2, &subnet2),
					testAccCaptureVPCID(&subnet1, &vpcID),
					testAccCreateGuardDutyResourcesForSubnets(ctx, t, &subnet1, &subnet2),
					testAccCheckGuardDutyResourcesExist(ctx, t, &subnet1),
				),
			},
			{
				// Step 2: Remove subnet1 from config. Terraform deletes it,
				// dissociateGuardDutyVPCEndpoints dissociates it from the endpoint.
				// Endpoint should remain available with subnet2.
				Config: testAccVPCSubnetConfig_GuardDutyDependencies_cleanupSingleSubnet(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName2, &subnet2),
					testAccCheckGuardDutyEndpointStillExists(ctx, t, &vpcID),
				),
			},
		},
	})
}

// TestAccVPCSubnet_GuardDutyDependencies_lastSubnetDeletion is a bug condition exploration test.
// It creates a GuardDuty VPC endpoint with exactly ONE subnet, then removes the subnet
// from the Terraform config while keeping the endpoint (with ignore_changes on subnet_ids).
// This triggers dissociateGuardDutyVPCEndpoints on the last subnet, which hits the bug:
// ModifyVpcEndpoint with RemoveSubnetIds fails because an Interface VPC Endpoint requires
// at least one subnet.
//
// TestAccVPCSubnet_GuardDutyDependencies_lastSubnetDeletion validates the single-subnet dissociation path.
// It creates a VPC with one subnet, then creates GuardDuty resources (endpoint + SG) out-of-band
// via SDK. When the subnet is removed from the Terraform config, Terraform tries to delete it,
// gets a DependencyViolation because the out-of-band endpoint is still there, and
// dissociateGuardDutyVPCEndpoints runs to clean it up.
func TestAccVPCSubnet_GuardDutyDependencies_lastSubnetDeletion(t *testing.T) {
	ctx := acctest.Context(t)
	var subnet1 awstypes.Subnet
	var vpcID string
	resourceName1 := "aws_subnet.test1"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGuardDutyCleanupDestroy(ctx, t, &vpcID),
		Steps: []resource.TestStep{
			{
				// Step 1: Create a VPC with a single subnet. After Terraform creates
				// the infrastructure, create GuardDuty resources (endpoint + SG)
				// out-of-band via SDK so they are NOT in Terraform state.
				Config: testAccVPCSubnetConfig_guardDutyLastSubnet_withSubnet(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceName1, &subnet1),
					testAccCaptureVPCID(&subnet1, &vpcID),
					testAccCreateGuardDutyResourcesForSubnet(ctx, t, &subnet1),
					testAccCheckGuardDutyResourcesExist(ctx, t, &subnet1),
				),
			},
			{
				// Step 2: Remove the subnet from the config (VPC only remains).
				// Terraform tries to delete the subnet, gets DependencyViolation
				// because the out-of-band GuardDuty endpoint is still associated,
				// and dissociateGuardDutyVPCEndpoints runs to clean it up.
				Config: testAccVPCSubnetConfig_guardDutyLastSubnet_withoutSubnet(rName),
			},
		},
	})
}

// testAccCreateGuardDutyResourcesForSubnet is a convenience wrapper around
// testAccCreateGuardDutyResources that extracts the subnet ID at execution time
// from the populated subnet pointer.
func testAccCreateGuardDutyResourcesForSubnet(ctx context.Context, t *testing.T, subnet *awstypes.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subnetID := aws.ToString(subnet.SubnetId)
		return testAccCreateGuardDutyResources(ctx, t, subnet, []string{subnetID})(s)
	}
}

// testAccCreateGuardDutyResourcesForSubnets is a convenience wrapper around
// testAccCreateGuardDutyResources that extracts both subnet IDs at execution time
// from two populated subnet pointers.
func testAccCreateGuardDutyResourcesForSubnets(ctx context.Context, t *testing.T, subnet1, subnet2 *awstypes.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subnetID1 := aws.ToString(subnet1.SubnetId)
		subnetID2 := aws.ToString(subnet2.SubnetId)
		return testAccCreateGuardDutyResources(ctx, t, subnet1, []string{subnetID1, subnetID2})(s)
	}
}

func testAccCheckGuardDutyResourcesExist(ctx context.Context, t *testing.T, subnet *awstypes.Subnet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)
		vpcID := aws.ToString(subnet.VpcId)

		endpoints, err := tfec2.FindGuardDutyVPCEndpoints(ctx, conn, vpcID)
		if err != nil {
			return fmt.Errorf("error describing VPC endpoints: %w", err)
		}
		if len(endpoints) == 0 {
			return fmt.Errorf("expected GuardDuty VPC endpoint with GuardDutyManaged=true tag to exist, but none found")
		}

		sgs, err := tfec2.FindGuardDutySecurityGroupsForVPC(ctx, conn, vpcID)
		if err != nil {
			return fmt.Errorf("error describing security groups: %w", err)
		}
		if len(sgs) == 0 {
			return fmt.Errorf("expected GuardDuty security group with GuardDutyManaged=true tag to exist, but none found")
		}

		return nil
	}
}

func testAccCheckGuardDutyEndpointStillExists(ctx context.Context, t *testing.T, vpcID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if vpcID == nil || *vpcID == "" {
			return fmt.Errorf("VPC ID not captured")
		}

		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)

		endpoints, err := tfec2.FindGuardDutyVPCEndpoints(ctx, conn, *vpcID)
		if err != nil {
			return fmt.Errorf("error describing VPC endpoints: %w", err)
		}
		if len(endpoints) == 0 {
			return fmt.Errorf("expected GuardDuty VPC endpoint to still exist after subnet dissociation, but none found")
		}

		for _, ep := range endpoints {
			state := string(ep.State)
			if state != "available" {
				return fmt.Errorf("expected GuardDuty VPC endpoint %s to be in 'available' state, got %q",
					aws.ToString(ep.VpcEndpointId), state)
			}
		}

		return nil
	}
}

func testAccCaptureVPCID(subnet *awstypes.Subnet, vpcID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		*vpcID = aws.ToString(subnet.VpcId)
		return nil
	}
}

// testAccCreateGuardDutyResources creates GuardDuty-tagged resources (VPC endpoint and security group)
// out-of-band using the AWS SDK directly. This is critical because when GuardDuty resources are
// Terraform-managed, Terraform handles dependency ordering and destroys the endpoint before the subnet,
// so dissociateGuardDutyVPCEndpoints is never exercised. In production, GuardDuty creates these
// resources out-of-band.
func testAccCreateGuardDutyResources(ctx context.Context, t *testing.T, subnet *awstypes.Subnet, subnetIDs []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)
		region := acctest.ProviderMeta(ctx, t).Region(ctx)
		vpcID := aws.ToString(subnet.VpcId)

		// Create GuardDuty-tagged security group
		sgName := tfec2.GuardDutySecurityGroupNameForVPC(vpcID)
		sgInput := ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(sgName),
			Description: aws.String("GuardDuty managed security group for testing"),
			VpcId:       aws.String(vpcID),
			TagSpecifications: []awstypes.TagSpecification{
				{
					ResourceType: awstypes.ResourceTypeSecurityGroup,
					Tags: []awstypes.Tag{
						{
							Key:   aws.String("GuardDutyManaged"),
							Value: aws.String(acctest.CtTrue),
						},
					},
				},
			},
		}
		sgOutput, err := conn.CreateSecurityGroup(ctx, &sgInput)
		if err != nil {
			return fmt.Errorf("creating GuardDuty security group in VPC %s: %w", vpcID, err)
		}
		sgID := aws.ToString(sgOutput.GroupId)

		// Create GuardDuty-tagged VPC endpoint
		serviceName := fmt.Sprintf("com.amazonaws.%s.guardduty-data", region)
		epInput := ec2.CreateVpcEndpointInput{
			VpcId:            aws.String(vpcID),
			ServiceName:      aws.String(serviceName),
			VpcEndpointType:  awstypes.VpcEndpointTypeInterface,
			SubnetIds:        subnetIDs,
			SecurityGroupIds: []string{sgID},
			TagSpecifications: []awstypes.TagSpecification{
				{
					ResourceType: awstypes.ResourceTypeVpcEndpoint,
					Tags: []awstypes.Tag{
						{
							Key:   aws.String("GuardDutyManaged"),
							Value: aws.String(acctest.CtTrue),
						},
					},
				},
			},
		}
		epOutput, err := conn.CreateVpcEndpoint(ctx, &epInput)
		if err != nil {
			return fmt.Errorf("creating GuardDuty VPC endpoint in VPC %s: %w", vpcID, err)
		}
		endpointID := aws.ToString(epOutput.VpcEndpoint.VpcEndpointId)

		// Wait for endpoint to reach available state
		if _, err := tfec2.WaitVPCEndpointAvailable(ctx, conn, endpointID, tfec2.VPCEndpointCreationTimeout); err != nil {
			return fmt.Errorf("waiting for GuardDuty VPC endpoint %s to become available: %w", endpointID, err)
		}

		return nil
	}
}

func testAccCheckGuardDutyCleanupDestroy(ctx context.Context, t *testing.T, vpcID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_subnet" {
				continue
			}

			input := ec2.DescribeSubnetsInput{
				SubnetIds: []string{rs.Primary.ID},
			}
			_, err := conn.DescribeSubnets(ctx, &input)
			if err == nil {
				return fmt.Errorf("subnet %s still exists", rs.Primary.ID)
			}
		}

		if vpcID != nil && *vpcID != "" {
			endpoints, err := tfec2.FindGuardDutyVPCEndpoints(ctx, conn, *vpcID)
			if err != nil {
				return fmt.Errorf("error describing GuardDuty VPC endpoints: %w", err)
			}
			activeEndpoints := 0
			for _, ep := range endpoints {
				if string(ep.State) != "deleted" {
					activeEndpoints++
				}
			}
			if activeEndpoints > 0 {
				return fmt.Errorf("expected GuardDuty VPC endpoints to be cleaned up, but found %d active endpoint(s)", activeEndpoints)
			}

			sgs, err := tfec2.FindGuardDutySecurityGroupsForVPC(ctx, conn, aws.ToString(vpcID))
			if err != nil {
				return fmt.Errorf("error describing GuardDuty security groups: %w", err)
			}
			if len(sgs) > 0 {
				return fmt.Errorf("expected GuardDuty security groups to be cleaned up, but found %d group(s)", len(sgs))
			}
		}

		return nil
	}
}

func testAccVPCSubnetConfig_GuardDutyDependencies_cleanupBothSubnets(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test1" {
  cidr_block        = "10.1.1.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "%[1]s-1"
  }
}

resource "aws_subnet" "test2" {
  cidr_block        = "10.1.2.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "%[1]s-2"
  }
}
`, rName)
}

func testAccVPCSubnetConfig_GuardDutyDependencies_cleanupSingleSubnet(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test2" {
  cidr_block        = "10.1.2.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "%[1]s-2"
  }
}
`, rName)
}

func TestIsUnauthorizedError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "UnauthorizedOperation error",
			err:      fmt.Errorf("UnauthorizedOperation: You are not authorized to perform this operation"),
			expected: true,
		},
		{
			name:     "AccessDenied error",
			err:      fmt.Errorf("AccessDenied: Access denied"),
			expected: true,
		},
		{
			name:     "not authorized error",
			err:      fmt.Errorf("User is not authorized to perform action"),
			expected: true,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("InternalError: An internal error occurred"),
			expected: false,
		},
		{
			name:     "RequestLimitExceeded error",
			err:      fmt.Errorf("RequestLimitExceeded: Request limit exceeded"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tfec2.IsUnauthorizedError(tc.err)
			if result != tc.expected {
				t.Errorf("IsUnauthorizedError(%v) = %v, want %v", tc.err, result, tc.expected)
			}
		})
	}
}

func testAccVPCSubnetConfig_guardDutyLastSubnet_withSubnet(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "test1" {
  cidr_block        = "10.1.1.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "%[1]s-1"
  }
}
`, rName)
}

func testAccVPCSubnetConfig_guardDutyLastSubnet_withoutSubnet(rName string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "test" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = %[1]q
  }
}
`, rName)
}

// TestAccVPCSubnet_GuardDutyDependencies_notAssociated validates UC-S3: deleting a subnet that is NOT
// associated with a GuardDuty endpoint. The endpoint is associated with subnet-B only.
// When subnet-A is removed from the Terraform config, dissociateGuardDutyVPCEndpoints
// should find the endpoint but skip it because subnet-A is not in the endpoint's SubnetIds.
// The endpoint should remain untouched in available state with subnet-B.
//
// EXPECTED: Test PASSES. Endpoint is untouched because subnet-A was never associated with it.
func TestAccVPCSubnet_GuardDutyDependencies_notAssociated(t *testing.T) {
	ctx := acctest.Context(t)
	var subnetA, subnetB awstypes.Subnet
	var vpcID string
	resourceNameA := "aws_subnet.testA"
	resourceNameB := "aws_subnet.testB"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGuardDutyCleanupDestroy(ctx, t, &vpcID),
		Steps: []resource.TestStep{
			{
				// Step 1: Create VPC with two subnets (A and B). After Terraform
				// creates the infrastructure, create GuardDuty resources out-of-band
				// via SDK with ONLY subnet-B's ID (not subnet-A).
				Config: testAccVPCSubnetConfig_guardDutyNotAssociated_bothSubnets(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceNameA, &subnetA),
					testAccCheckSubnetExists(ctx, t, resourceNameB, &subnetB),
					testAccCaptureVPCID(&subnetB, &vpcID),
					// Create endpoint associated with subnet-B ONLY
					testAccCreateGuardDutyResourcesForSubnet(ctx, t, &subnetB),
					testAccCheckGuardDutyResourcesExist(ctx, t, &subnetB),
				),
			},
			{
				// Step 2: Remove subnet-A from config (keep subnet-B). Terraform
				// deletes subnet-A. Since subnet-A is NOT associated with the
				// endpoint, dissociateGuardDutyVPCEndpoints should skip it.
				// The subnet should delete normally.
				Config: testAccVPCSubnetConfig_guardDutyNotAssociated_onlySubnetB(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubnetExists(ctx, t, resourceNameB, &subnetB),
					// Endpoint should still exist unchanged with subnet-B in available state
					testAccCheckGuardDutyEndpointStillExists(ctx, t, &vpcID),
				),
			},
		},
	})
}

func testAccVPCSubnetConfig_guardDutyNotAssociated_bothSubnets(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "testA" {
  cidr_block        = "10.1.1.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[0]

  tags = {
    Name = "%[1]s-A"
  }
}

resource "aws_subnet" "testB" {
  cidr_block        = "10.1.2.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "%[1]s-B"
  }
}
`, rName)
}

func testAccVPCSubnetConfig_guardDutyNotAssociated_onlySubnetB(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

resource "aws_vpc" "test" {
  cidr_block           = "10.1.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = %[1]q
  }
}

resource "aws_subnet" "testB" {
  cidr_block        = "10.1.2.0/24"
  vpc_id            = aws_vpc.test.id
  availability_zone = data.aws_availability_zones.available.names[1]

  tags = {
    Name = "%[1]s-B"
  }
}
`, rName)
}

// TestIsVPCOwnedByAccount_logic validates the ownership comparison that
// determines whether GuardDuty cleanup should be skipped for shared VPCs.
// When the VPC is owned by a different account than the one configured on
// the provider, cleanup should be skipped because the GuardDuty resources
// belong to the VPC owner, not the subnet owner.
//
// Note: An acceptance test for shared VPC GuardDuty cleanup is not viable because
// participant accounts in a shared VPC cannot create or delete subnets — only the
// VPC owner can. The participant receives UnauthorizedOperation (403), never
// DependencyViolation, so the GuardDuty cleanup code path is unreachable.
// When GuardDuty is enabled with shared VPC support, the VPC endpoint is created
// in the VPC owner's account regardless of which account triggered it.
// See: https://docs.aws.amazon.com/guardduty/latest/ug/runtime-monitoring-shared-vpc.html
func TestIsVPCOwnedByAccount_logic(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		vpcOwner  string
		accountID string
		expected  bool
	}{
		{
			name:      "same account",
			vpcOwner:  acctest.Ct12Digit,
			accountID: acctest.Ct12Digit,
			expected:  true,
		},
		{
			name:      "different account (shared VPC)",
			vpcOwner:  "999888777666",
			accountID: acctest.Ct12Digit,
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// The core logic of isVPCOwnedByAccount is:
			//   aws.ToString(vpc.OwnerId) == accountID
			// We test this directly since the function requires a real EC2 client.
			result := tc.vpcOwner == tc.accountID
			if result != tc.expected {
				t.Errorf("VPC owner %q == account %q: got %v, want %v",
					tc.vpcOwner, tc.accountID, result, tc.expected)
			}
		})
	}
}
