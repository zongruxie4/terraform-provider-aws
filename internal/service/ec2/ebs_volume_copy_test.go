// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package ec2_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfec2 "github.com/hashicorp/terraform-provider-aws/internal/service/ec2"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccEC2EBSVolumeCopy_basic(t *testing.T) {
	ctx := acctest.Context(t)

	var ebsVolumeCopy awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSVolumeCopyConfig_basic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy),
					acctest.MatchResourceAttrRegionalARN(ctx, resourceName, names.AttrARN, "ec2", regexache.MustCompile(`volume/.+`)),
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

func TestAccEC2EBSVolumeCopy_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var ebsVolumeCopy awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSVolumeCopyConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy),
					acctest.CheckFrameworkResourceDisappearsWithStateFunc(ctx, t, tfec2.ResourceEBSVolumeCopy, resourceName, ebsVolumeCopyDisappearsStateFunc),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccEC2EBSVolumeCopy_tags(t *testing.T) {
	ctx := acctest.Context(t)
	var ebsVolumeCopy awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSVolumeCopyConfig_tags1(acctest.CtKey1, acctest.CtValue1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy),
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
				Config: testAccEBSVolumeCopyConfig_tags2(acctest.CtKey1, acctest.CtValue1Updated, acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "2"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey1, acctest.CtValue1Updated),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey2, acctest.CtValue2),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEBSVolumeCopyConfig_tags1(acctest.CtKey2, acctest.CtValue2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "1"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsKey2, acctest.CtValue2),
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

func TestAccEC2EBSVolumeCopy_defaultTags_providerOnly(t *testing.T) {
	ctx := acctest.Context(t)
	var ebsVolumeCopy awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: acctest.ConfigCompose(
					acctest.ConfigDefaultTags_Tags1(acctest.CtProviderKey1, acctest.CtProviderValue1),
					testAccEBSVolumeCopyConfig_basic(),
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsPercent, "0"),
					resource.TestCheckResourceAttr(resourceName, acctest.CtTagsAllPercent, "1"),
					resource.TestCheckResourceAttr(resourceName, "tags_all."+acctest.CtProviderKey1, acctest.CtProviderValue1),
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

func TestAccEC2EBSVolumeCopy_updateSize(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var ebsVolumeCopy1, ebsVolumeCopy2 awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSVolumeCopyConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy1),
					resource.TestCheckResourceAttr(resourceName, names.AttrSize, "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEBSVolumeCopyConfig_updateSize(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy2),
					testAccCheckEBSVolumeCopyNotRecreated(&ebsVolumeCopy1, &ebsVolumeCopy2),
					resource.TestCheckResourceAttr(resourceName, names.AttrSize, "2"),
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

func TestAccEC2EBSVolumeCopy_updateIops(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var ebsVolumeCopy1, ebsVolumeCopy2 awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSVolumeCopyConfig_iops(3000),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy1),
					resource.TestCheckResourceAttr(resourceName, names.AttrVolumeType, "gp3"),
					resource.TestCheckResourceAttr(resourceName, names.AttrIOPS, "3000"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEBSVolumeCopyConfig_iops(4000),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy2),
					testAccCheckEBSVolumeCopyNotRecreated(&ebsVolumeCopy1, &ebsVolumeCopy2),
					resource.TestCheckResourceAttr(resourceName, names.AttrVolumeType, "gp3"),
					resource.TestCheckResourceAttr(resourceName, names.AttrIOPS, "4000"),
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

func TestAccEC2EBSVolumeCopy_updateThroughput(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var ebsVolumeCopy1, ebsVolumeCopy2 awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSVolumeCopyConfig_throughput(125),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy1),
					resource.TestCheckResourceAttr(resourceName, names.AttrVolumeType, "gp3"),
					resource.TestCheckResourceAttr(resourceName, names.AttrThroughput, "125"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEBSVolumeCopyConfig_throughput(150),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy2),
					testAccCheckEBSVolumeCopyNotRecreated(&ebsVolumeCopy1, &ebsVolumeCopy2),
					resource.TestCheckResourceAttr(resourceName, names.AttrVolumeType, "gp3"),
					resource.TestCheckResourceAttr(resourceName, names.AttrThroughput, "150"),
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

func TestAccEC2EBSVolumeCopy_updateVolumeType(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	var ebsVolumeCopy1, ebsVolumeCopy2 awstypes.Volume
	resourceName := "aws_ebs_volume_copy.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.EC2)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.EC2ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckEBSVolumeCopyDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccEBSVolumeCopyConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy1),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccEBSVolumeCopyConfig_volumeTypeGP3(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEBSVolumeCopyExists(ctx, t, resourceName, &ebsVolumeCopy2),
					testAccCheckEBSVolumeCopyNotRecreated(&ebsVolumeCopy1, &ebsVolumeCopy2),
					resource.TestCheckResourceAttr(resourceName, names.AttrVolumeType, "gp3"),
					resource.TestCheckResourceAttr(resourceName, names.AttrIOPS, "3000"),
					resource.TestCheckResourceAttr(resourceName, names.AttrThroughput, "125"),
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

func testAccCheckEBSVolumeCopyDestroy(ctx context.Context, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_ebs_volume_copy" {
				continue
			}

			_, err := tfec2.FindEBSVolumeByID(ctx, conn, rs.Primary.ID)
			if retry.NotFound(err) {
				return nil
			}
			if err != nil {
				return create.Error(names.EC2, create.ErrActionCheckingDestroyed, "EBS Volume Copy", rs.Primary.ID, err)
			}

			return create.Error(names.EC2, create.ErrActionCheckingDestroyed, "EBS Volume Copy", rs.Primary.ID, errors.New("not destroyed"))
		}

		return nil
	}
}

func testAccCheckEBSVolumeCopyExists(ctx context.Context, t *testing.T, name string, ebsVolumeCopy *awstypes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return create.Error(names.EC2, create.ErrActionCheckingExistence, "EBS Volume Copy", name, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return create.Error(names.EC2, create.ErrActionCheckingExistence, "EBS Volume Copy", name, errors.New("not set"))
		}

		conn := acctest.ProviderMeta(ctx, t).EC2Client(ctx)

		resp, err := tfec2.FindEBSVolumeByID(ctx, conn, rs.Primary.ID)
		if err != nil {
			return create.Error(names.EC2, create.ErrActionCheckingExistence, "EBS Volume Copy", rs.Primary.ID, err)
		}

		*ebsVolumeCopy = *resp

		return nil
	}
}

func ebsVolumeCopyDisappearsStateFunc(ctx context.Context, state *tfsdk.State, is *terraform.InstanceState) error {
	if is.ID == "" {
		return errors.New(`identifying attribute "id" not defined`)
	}

	if err := fwdiag.DiagnosticsError(state.SetAttribute(ctx, path.Root(names.AttrID), is.ID)); err != nil {
		return err
	}

	if _, ok := state.Schema.GetAttributes()[names.AttrRegion]; ok {
		if err := fwdiag.DiagnosticsError(state.SetAttribute(ctx, path.Root(names.AttrRegion), acctest.Region())); err != nil {
			return err
		}
	}

	return nil
}

func testAccCheckEBSVolumeCopyNotRecreated(before, after *awstypes.Volume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if before, after := aws.ToString(before.VolumeId), aws.ToString(after.VolumeId); before != after {
			return errors.New("EC2 EBS Volume Copy was recreated")
		}

		return nil
	}
}

func testAccEBSVolumeCopyConfigBaseConfig() string {
	return acctest.ConfigCompose(acctest.ConfigAvailableAZsNoOptIn(), `
data "aws_region" "current" {}

resource "aws_ebs_volume" "test" {
  availability_zone = data.aws_availability_zones.available.names[0]
  size              = 1
  encrypted         = true
}
`)
}

func testAccEBSVolumeCopyConfig_basic() string {
	return acctest.ConfigCompose(testAccEBSVolumeCopyConfigBaseConfig(), `
resource "aws_ebs_volume_copy" "test" {
  source_volume_id = aws_ebs_volume.test.id
}
`)
}

func testAccEBSVolumeCopyConfig_updateSize() string {
	return acctest.ConfigCompose(testAccEBSVolumeCopyConfigBaseConfig(), `
resource "aws_ebs_volume_copy" "test" {
  source_volume_id = aws_ebs_volume.test.id
  size             = 2
}
`)
}

func testAccEBSVolumeCopyConfig_iops(iops int) string {
	return acctest.ConfigCompose(testAccEBSVolumeCopyConfigBaseConfig(), fmt.Sprintf(`
resource "aws_ebs_volume_copy" "test" {
  source_volume_id = aws_ebs_volume.test.id
  volume_type      = "gp3"
  iops             = %d
  size             = 8
}
`, iops))
}

func testAccEBSVolumeCopyConfig_throughput(throughput int) string {
	return acctest.ConfigCompose(testAccEBSVolumeCopyConfigBaseConfig(), fmt.Sprintf(`
resource "aws_ebs_volume_copy" "test" {
  source_volume_id = aws_ebs_volume.test.id
  volume_type      = "gp3"
  throughput       = %d
  size             = 1
}
`, throughput))
}

func testAccEBSVolumeCopyConfig_volumeTypeGP3() string {
	return acctest.ConfigCompose(testAccEBSVolumeCopyConfigBaseConfig(), `
resource "aws_ebs_volume_copy" "test" {
  source_volume_id = aws_ebs_volume.test.id
  volume_type      = "gp3"
}
`)
}

func testAccEBSVolumeCopyConfig_tags1(tagKey1, tagValue1 string) string {
	return acctest.ConfigCompose(testAccEBSVolumeCopyConfigBaseConfig(), fmt.Sprintf(`
resource "aws_ebs_volume_copy" "test" {
  source_volume_id = aws_ebs_volume.test.id

  tags = {
    %[1]q = %[2]q
  }
}
`, tagKey1, tagValue1))
}

func testAccEBSVolumeCopyConfig_tags2(tagKey1, tagValue1, tagKey2, tagValue2 string) string {
	return acctest.ConfigCompose(testAccEBSVolumeCopyConfigBaseConfig(), fmt.Sprintf(`
resource "aws_ebs_volume_copy" "test" {
  source_volume_id = aws_ebs_volume.test.id

  tags = {
    %[1]q = %[2]q
    %[3]q = %[4]q
  }
}
`, tagKey1, tagValue1, tagKey2, tagValue2))
}
