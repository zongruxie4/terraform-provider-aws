// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package s3_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	tfquerycheck "github.com/hashicorp/terraform-provider-aws/internal/acctest/querycheck"
	tfqueryfilter "github.com/hashicorp/terraform-provider-aws/internal/acctest/queryfilter"
	tfstatecheck "github.com/hashicorp/terraform-provider-aws/internal/acctest/statecheck"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccS3Object_List_basic(t *testing.T) {
	ctx := acctest.Context(t)

	resourceName1 := "aws_s3_object.test[0]"
	resourceName2 := "aws_s3_object.test[1]"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	identity1 := tfstatecheck.Identity()
	identity2 := tfstatecheck.Identity()

	acctest.ParallelTest(ctx, t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.S3ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckObjectDestroy(ctx, t),
		Steps: []resource.TestStep{
			// Step 1: Setup
			{
				ConfigDirectory: config.StaticDirectory("testdata/Object/list_basic/"),
				ConfigVariables: config.Variables{
					acctest.CtRName:  config.StringVariable(rName),
					"resource_count": config.IntegerVariable(2),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					identity1.GetIdentity(resourceName1),
					statecheck.ExpectKnownValue(resourceName1, tfjsonpath.New(names.AttrBucket), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName1, tfjsonpath.New(names.AttrKey), knownvalue.StringExact(rName+"-0")),

					identity2.GetIdentity(resourceName2),
					statecheck.ExpectKnownValue(resourceName2, tfjsonpath.New(names.AttrBucket), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName2, tfjsonpath.New(names.AttrKey), knownvalue.StringExact(rName+"-1")),
				},
			},

			// Step 2: Query
			{
				Query:           true,
				ConfigDirectory: config.StaticDirectory("testdata/Object/list_basic/"),
				ConfigVariables: config.Variables{
					acctest.CtRName:  config.StringVariable(rName),
					"resource_count": config.IntegerVariable(2),
				},
				QueryResultChecks: []querycheck.QueryResultCheck{
					tfquerycheck.ExpectIdentityFunc("aws_s3_object.test", identity1.Checks()),
					querycheck.ExpectResourceDisplayName("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity1.Checks()), knownvalue.StringExact(rName+"-0")),
					tfquerycheck.ExpectNoResourceObject("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity1.Checks())),

					tfquerycheck.ExpectIdentityFunc("aws_s3_object.test", identity2.Checks()),
					querycheck.ExpectResourceDisplayName("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity2.Checks()), knownvalue.StringExact(rName+"-1")),
					tfquerycheck.ExpectNoResourceObject("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity2.Checks())),
				},
			},
		},
	})
}

func TestAccS3Object_List_directoryBucket(t *testing.T) {
	ctx := acctest.Context(t)

	resourceName1 := "aws_s3_object.test[0]"
	resourceName2 := "aws_s3_object.test[1]"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	identity1 := tfstatecheck.Identity()
	identity2 := tfstatecheck.Identity()

	acctest.ParallelTest(ctx, t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.S3ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckObjectDestroy(ctx, t),
		Steps: []resource.TestStep{
			// Step 1: Setup
			{
				ConfigDirectory: config.StaticDirectory("testdata/Object/list_directory_bucket/"),
				ConfigVariables: config.Variables{
					acctest.CtRName:  config.StringVariable(rName),
					"resource_count": config.IntegerVariable(2),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					identity1.GetIdentity(resourceName1),
					statecheck.ExpectKnownValue(resourceName1, tfjsonpath.New(names.AttrBucket), knownvalue.StringRegexp(directoryBucketFullNameRegex(rName))),
					statecheck.ExpectKnownValue(resourceName1, tfjsonpath.New(names.AttrKey), knownvalue.StringExact(rName+"-0")),

					identity2.GetIdentity(resourceName2),
					statecheck.ExpectKnownValue(resourceName2, tfjsonpath.New(names.AttrBucket), knownvalue.StringRegexp(directoryBucketFullNameRegex(rName))),
					statecheck.ExpectKnownValue(resourceName2, tfjsonpath.New(names.AttrKey), knownvalue.StringExact(rName+"-1")),
				},
			},

			// Step 2: Query
			{
				Query:           true,
				ConfigDirectory: config.StaticDirectory("testdata/Object/list_directory_bucket/"),
				ConfigVariables: config.Variables{
					acctest.CtRName:  config.StringVariable(rName),
					"resource_count": config.IntegerVariable(2),
				},
				QueryResultChecks: []querycheck.QueryResultCheck{
					tfquerycheck.ExpectIdentityFunc("aws_s3_object.test", identity1.Checks()),
					querycheck.ExpectResourceDisplayName("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity1.Checks()), knownvalue.StringExact(rName+"-0")),
					tfquerycheck.ExpectNoResourceObject("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity1.Checks())),

					tfquerycheck.ExpectIdentityFunc("aws_s3_object.test", identity2.Checks()),
					querycheck.ExpectResourceDisplayName("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity2.Checks()), knownvalue.StringExact(rName+"-1")),
					tfquerycheck.ExpectNoResourceObject("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity2.Checks())),
				},
			},
		},
	})
}

func TestAccS3Object_List_prefix(t *testing.T) {
	ctx := acctest.Context(t)

	resourceName1 := "aws_s3_object.test[0]"
	resourceName2 := "aws_s3_object.test[1]"
	resourceName3 := "aws_s3_object.other[0]"
	resourceName4 := "aws_s3_object.other[1]"
	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)

	identity1 := tfstatecheck.Identity()
	identity2 := tfstatecheck.Identity()
	identity3 := tfstatecheck.Identity()
	identity4 := tfstatecheck.Identity()

	acctest.ParallelTest(ctx, t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.S3ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckObjectDestroy(ctx, t),
		Steps: []resource.TestStep{
			// Step 1: Setup
			{
				ConfigDirectory: config.StaticDirectory("testdata/Object/list_prefix/"),
				ConfigVariables: config.Variables{
					acctest.CtRName:  config.StringVariable(rName),
					"resource_count": config.IntegerVariable(2),
				},
				ConfigStateChecks: []statecheck.StateCheck{
					identity1.GetIdentity(resourceName1),
					statecheck.ExpectKnownValue(resourceName1, tfjsonpath.New(names.AttrBucket), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName1, tfjsonpath.New(names.AttrKey), knownvalue.StringExact("prefix-"+rName+"-0")),

					identity2.GetIdentity(resourceName2),
					statecheck.ExpectKnownValue(resourceName2, tfjsonpath.New(names.AttrBucket), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName2, tfjsonpath.New(names.AttrKey), knownvalue.StringExact("prefix-"+rName+"-1")),

					identity3.GetIdentity(resourceName3),
					statecheck.ExpectKnownValue(resourceName3, tfjsonpath.New(names.AttrBucket), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName3, tfjsonpath.New(names.AttrKey), knownvalue.StringExact("other-"+rName+"-0")),

					identity4.GetIdentity(resourceName4),
					statecheck.ExpectKnownValue(resourceName4, tfjsonpath.New(names.AttrBucket), knownvalue.StringExact(rName)),
					statecheck.ExpectKnownValue(resourceName4, tfjsonpath.New(names.AttrKey), knownvalue.StringExact("other-"+rName+"-1")),
				},
			},

			// Step 2: Query only for prefix- objects
			{
				Query:           true,
				ConfigDirectory: config.StaticDirectory("testdata/Object/list_prefix/"),
				ConfigVariables: config.Variables{
					acctest.CtRName:  config.StringVariable(rName),
					"resource_count": config.IntegerVariable(2),
				},
				QueryResultChecks: []querycheck.QueryResultCheck{
					tfquerycheck.ExpectIdentityFunc("aws_s3_object.test", identity1.Checks()),
					querycheck.ExpectResourceDisplayName("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity1.Checks()), knownvalue.StringExact("prefix-"+rName+"-0")),
					tfquerycheck.ExpectNoResourceObject("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity1.Checks())),

					tfquerycheck.ExpectIdentityFunc("aws_s3_object.test", identity2.Checks()),
					querycheck.ExpectResourceDisplayName("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity2.Checks()), knownvalue.StringExact("prefix-"+rName+"-1")),
					tfquerycheck.ExpectNoResourceObject("aws_s3_object.test", tfqueryfilter.ByResourceIdentityFunc(identity2.Checks())),

					tfquerycheck.ExpectNoIdentityFunc("aws_s3_object.test", identity3.Checks()),

					tfquerycheck.ExpectNoIdentityFunc("aws_s3_object.test", identity4.Checks()),
				},
			},
		},
	})
}
