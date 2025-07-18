// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloudfront

import (
	"context"

	"github.com/hashicorp/aws-sdk-go-base/v2/endpoints"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKDataSource("aws_cloudfront_log_delivery_canonical_user_id", name="Log Delivery Canonical User ID")
func dataSourceLogDeliveryCanonicalUserID() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataSourceLogDeliveryCanonicalUserIDRead,

		Schema: map[string]*schema.Schema{
			// As the CloudFront service is global we define our own region attribute.
			names.AttrRegion: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func dataSourceLogDeliveryCanonicalUserIDRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	const (
		// See https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/AccessLogs.html#AccessLogsBucketAndFileOwnership.
		defaultLogDeliveryCanonicalUserID = "c4c1ede66af53448b93c283ce9448c4ba468c9432aa01d700d3878632f77d2d0"

		// See https://docs.amazonaws.cn/AmazonCloudFront/latest/DeveloperGuide/AccessLogs.html#AccessLogsBucketAndFileOwnership.
		cnLogDeliveryCanonicalUserID = "a52cb28745c0c06e84ec548334e44bfa7fc2a85c54af20cd59e4969344b7af56"
	)
	canonicalUserID := defaultLogDeliveryCanonicalUserID
	region := d.Get(names.AttrRegion).(string)
	if region == "" {
		region = meta.(*conns.AWSClient).Region(ctx)
	}
	if v := names.PartitionForRegion(region); v.ID() == endpoints.AwsCnPartitionID {
		canonicalUserID = cnLogDeliveryCanonicalUserID
	}

	d.SetId(canonicalUserID)

	return diags
}
