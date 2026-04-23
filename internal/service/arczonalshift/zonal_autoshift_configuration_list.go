// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package arczonalshift

import (
	"context"
	"fmt"
	"iter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/arczonalshift"
	awstypes "github.com/aws/aws-sdk-go-v2/service/arczonalshift/types"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	flex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/logging"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// Function annotations are used for list resource registration to the Provider. DO NOT EDIT.
// @FrameworkListResource("aws_arczonalshift_zonal_autoshift_configuration")
func newZonalAutoshiftConfigurationResourceAsListResource() list.ListResourceWithConfigure {
	return &zonalAutoshiftConfigurationListResource{}
}

var _ list.ListResource = &zonalAutoshiftConfigurationListResource{}

type zonalAutoshiftConfigurationListResource struct {
	resourceZonalAutoshiftConfiguration
	framework.WithList
}

func (l *zonalAutoshiftConfigurationListResource) List(ctx context.Context, request list.ListRequest, stream *list.ListResultsStream) {
	awsClient := l.Meta()
	conn := awsClient.ARCZonalShiftClient(ctx)

	tflog.Info(ctx, "Listing resources")

	input := arczonalshift.ListManagedResourcesInput{}

	stream.Results = func(yield func(list.ListResult) bool) {
		for item, err := range listManagedResources(ctx, conn, &input) {
			if err != nil {
				result := fwdiag.NewListResultErrorDiagnostic(err)
				yield(result)
				return
			}

			// Skip resources that have no practice run configuration — they are not
			// managed by aws_arczonalshift_zonal_autoshift_configuration.
			if item.PracticeRunStatus == "" {
				continue
			}

			arn := aws.ToString(item.Arn)
			ctx := tflog.SetField(ctx, logging.ResourceAttributeKey(names.AttrResourceARN), arn)

			out, err := findManagedResourceByIdentifier(ctx, conn, arn)
			if err != nil {
				tflog.Error(ctx, "Reading ARC Zonal Shift Managed Resource", map[string]any{
					"error": err.Error(),
				})
				continue
			}

			if out == nil || out.PracticeRunConfiguration == nil {
				continue
			}

			result := request.NewListResult(ctx)
			var data resourceZonalAutoshiftConfigurationModel

			l.SetResult(ctx, awsClient, request.IncludeResource, &data, &result, func() {
				data.ResourceARN = flex.StringToFrameworkARN(ctx, out.Arn)
				if request.IncludeResource {
					smerr.AddEnrich(ctx, &result.Diagnostics, l.flatten(ctx, out, &data))
					if result.Diagnostics.HasError() {
						return
					}
				}
				result.DisplayName = aws.ToString(out.Name)
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

func listManagedResources(ctx context.Context, conn *arczonalshift.Client, input *arczonalshift.ListManagedResourcesInput) iter.Seq2[awstypes.ManagedResourceSummary, error] {
	return func(yield func(awstypes.ManagedResourceSummary, error) bool) {
		pages := arczonalshift.NewListManagedResourcesPaginator(conn, input)
		for pages.HasMorePages() {
			page, err := pages.NextPage(ctx)
			if err != nil {
				yield(awstypes.ManagedResourceSummary{}, fmt.Errorf("listing ARC Zonal Shift Managed Resources: %w", err))
				return
			}

			for _, item := range page.Items {
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}
