// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package securityhub_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccSecurityHubSecurityControlsDataSource_basic(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.SecurityHubServiceID)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "control_definitions.#"),
				),
			},
		},
	})
}

func TestAccSecurityHubSecurityControlsDataSource_standardsARN(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.SecurityHubServiceID)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_standardsARN(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "control_definitions.#"),
					resource.TestCheckResourceAttrSet(dataSourceName, "standards_arn"),
				),
			},
		},
	})
}

func TestAccSecurityHubSecurityControlsDataSource_currentRegionAvailability(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.SecurityHubServiceID)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_currentRegionAvailability(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "control_definitions.#"),
					resource.TestCheckResourceAttr(dataSourceName, "current_region_availability", "AVAILABLE"),
				),
			},
		},
	})
}

func TestAccSecurityHubSecurityControlsDataSource_severityRating(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, names.SecurityHubServiceID)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_severityRating(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "control_definitions.#"),
					resource.TestCheckResourceAttr(dataSourceName, "severity_rating", "CRITICAL"),
				),
			},
		},
	})
}

func testAccSecurityControlsDataSourceConfig_basic() string {
	return `
data "aws_securityhub_security_controls" "test" {}
`
}

func testAccSecurityControlsDataSourceConfig_standardsARN() string {
	return `
data "aws_securityhub_standards_subscriptions" "example" {}

data "aws_securityhub_security_controls" "test" {
  standards_arn = data.aws_securityhub_standards_subscriptions.example.standards_subscriptions[0].standards_arn
}
`
}

func testAccSecurityControlsDataSourceConfig_currentRegionAvailability() string {
	return `
data "aws_securityhub_security_controls" "test" {
  current_region_availability = "AVAILABLE"
}
`
}

func testAccSecurityControlsDataSourceConfig_severityRating() string {
	return `
data "aws_securityhub_security_controls" "test" {
  severity_rating = "CRITICAL"
}
`
}
