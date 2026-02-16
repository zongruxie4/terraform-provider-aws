// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package ec2

import (
	"context"
	"fmt"
	"iter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
)

// @FrameworkListResource("aws_ec2_secondary_subnet")
func newSecondarySubnetResourceAsListResource() list.ListResourceWithConfigure {
	return &secondarySubnetListResource{}
}

var _ list.ListResource = &secondarySubnetListResource{}

type secondarySubnetListResource struct {
	secondarySubnetResource
	framework.WithList
}

func (r *secondarySubnetListResource) List(ctx context.Context, request list.ListRequest, stream *list.ListResultsStream) {
	conn := r.Meta().EC2Client(ctx)

	var query listSecondarySubnetModel
	if request.Config.Raw.IsKnown() && !request.Config.Raw.IsNull() {
		if diags := request.Config.Get(ctx, &query); diags.HasError() {
			stream.Results = list.ListResultsStreamDiagnostics(diags)
			return
		}
	}

	stream.Results = func(yield func(list.ListResult) bool) {
		result := request.NewListResult(ctx)
		var input ec2.DescribeSecondarySubnetsInput
		for item, err := range listSecondarySubnets(ctx, conn, &input) {
			if err != nil {
				result = fwdiag.NewListResultErrorDiagnostic(err)
				yield(result)
				return
			}

			var data secondarySubnetResourceModel

			r.SetResult(ctx, r.Meta(), request.IncludeResource, &data, &result, func() {
				if diags := fwflex.Flatten(ctx, item, &data, fwflex.WithFieldNamePrefix("SecondarySubnet")); diags.HasError() {
					result.Diagnostics.Append(diags...)
					yield(result)
					return
				}

				id := aws.ToString(item.SecondarySubnetId)
				data.ID = fwflex.StringValueToFramework(ctx, id)
				result.DisplayName = id
			})

			if result.Diagnostics.HasError() {
				yield(result)
				return
			}

			if !yield(result) {
				return
			}
		}
	}
}

type listSecondarySubnetModel struct {
	framework.WithRegionModel
}

func listSecondarySubnets(ctx context.Context, conn *ec2.Client, input *ec2.DescribeSecondarySubnetsInput) iter.Seq2[awstypes.SecondarySubnet, error] {
	return func(yield func(awstypes.SecondarySubnet, error) bool) {
		pages := ec2.NewDescribeSecondarySubnetsPaginator(conn, input)
		for pages.HasMorePages() {
			page, err := pages.NextPage(ctx)
			if err != nil {
				yield(awstypes.SecondarySubnet{}, fmt.Errorf("listing EC2 (Elastic Compute Cloud) Secondary Subnet resources: %w", err))
				return
			}

			for _, item := range page.SecondarySubnets {
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}
