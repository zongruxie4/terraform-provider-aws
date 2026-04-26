// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package securityhub_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	tfknownvalue "github.com/hashicorp/terraform-provider-aws/internal/acctest/knownvalue"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccSecurityHubSecurityControlsDataSource_basic(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_basic(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("security_control_ids"), tfknownvalue.ListNotEmpty()),
				},
			},
		},
	})
}

func TestAccSecurityHubSecurityControlsDataSource_standardsARN(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_standardsARN(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("security_control_ids"), tfknownvalue.ListNotEmpty()),
				},
			},
		},
	})
}

func TestAccSecurityHubSecurityControlsDataSource_currentRegionAvailability(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_currentRegionAvailability(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("security_control_ids"), tfknownvalue.ListNotEmpty()),
				},
			},
		},
	})
}

func TestAccSecurityHubSecurityControlsDataSource_severityRating(t *testing.T) {
	ctx := acctest.Context(t)
	dataSourceName := "data.aws_securityhub_security_controls.test"

	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.SecurityHubServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSecurityControlsDataSourceConfig_severityRating(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceName, tfjsonpath.New("security_control_ids"), tfknownvalue.ListNotEmpty()),
				},
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
