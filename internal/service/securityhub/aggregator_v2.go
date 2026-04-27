// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package securityhub

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	awstypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_securityhub_aggregator_v2", name="Aggregator V2")
// @ArnIdentity
// @Tags(identifierAttribute="arn")
// @Testing(existsType="github.com/aws/aws-sdk-go-v2/service/securityhub;securityhub;securityhub.GetAggregatorV2Output")
// @Testing(serialize=true)
// @Testing(tagsTest=false)
// @Testing(hasNoPreExistingResource=true)
// @Testing(generator=false)
func newAggregatorV2Resource(_ context.Context) (resource.ResourceWithConfigure, error) {
	return &aggregatorV2Resource{}, nil
}

type aggregatorV2Resource struct {
	framework.ResourceWithModel[aggregatorV2ResourceModel]
	framework.WithImportByIdentity
}

func (r *aggregatorV2Resource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			"aggregation_region": schema.StringAttribute{
				Computed:    true,
				Description: "The AWS Region where data is aggregated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region_linking_mode": schema.StringAttribute{
				Required:    true,
				Description: "Determines how Regions are linked: ALL_REGIONS, ALL_REGIONS_EXCEPT_SPECIFIED, or SPECIFIED_REGIONS.",
			},
			"linked_regions": schema.ListAttribute{
				CustomType:  fwtypes.ListOfStringType,
				Optional:    true,
				ElementType: types.StringType,
				Description: "The list of Regions linked to the aggregation Region.",
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
	}
}

func (r *aggregatorV2Resource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data aggregatorV2ResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	input := securityhub.CreateAggregatorV2Input{
		RegionLinkingMode: aws.String(data.RegionLinkingMode.ValueString()),
		Tags:              getTagsIn(ctx),
	}

	if !data.LinkedRegions.IsNull() && !data.LinkedRegions.IsUnknown() {
		var regions []string
		response.Diagnostics.Append(data.LinkedRegions.ElementsAs(ctx, &regions, false)...)
		if response.Diagnostics.HasError() {
			return
		}
		input.LinkedRegions = regions
	}

	output, err := conn.CreateAggregatorV2(ctx, &input)

	if err != nil {
		response.Diagnostics.AddError("creating Security Hub V2 Aggregator", err.Error())
		return
	}

	data.ARN = fwflex.StringToFramework(ctx, output.AggregatorV2Arn)
	data.AggregationRegion = fwflex.StringToFramework(ctx, output.AggregationRegion)

	if output.LinkedRegions != nil {
		regionsList, diags := types.ListValueFrom(ctx, types.StringType, output.LinkedRegions)
		response.Diagnostics.Append(diags...)
		data.LinkedRegions = fwtypes.ListValueOf[types.String]{ListValue: regionsList}
	}

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (r *aggregatorV2Resource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data aggregatorV2ResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	output, err := findAggregatorV2ByARN(ctx, conn, data.ARN.ValueString())

	if retry.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		response.Diagnostics.AddError("reading Security Hub V2 Aggregator", err.Error())
		return
	}

	data.ARN = fwflex.StringToFramework(ctx, output.AggregatorV2Arn)
	data.AggregationRegion = fwflex.StringToFramework(ctx, output.AggregationRegion)
	if output.RegionLinkingMode != nil {
		data.RegionLinkingMode = fwflex.StringToFramework(ctx, output.RegionLinkingMode)
	}
	if output.LinkedRegions != nil {
		sorted := make([]string, len(output.LinkedRegions))
		copy(sorted, output.LinkedRegions)
		sort.Strings(sorted)
		regionsList, diags := types.ListValueFrom(ctx, types.StringType, sorted)
		response.Diagnostics.Append(diags...)
		data.LinkedRegions = fwtypes.ListValueOf[types.String]{ListValue: regionsList}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *aggregatorV2Resource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var old, new aggregatorV2ResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &new)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.State.Get(ctx, &old)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	if !new.RegionLinkingMode.Equal(old.RegionLinkingMode) || !new.LinkedRegions.Equal(old.LinkedRegions) {
		input := securityhub.UpdateAggregatorV2Input{
			AggregatorV2Arn:   aws.String(old.ARN.ValueString()),
			RegionLinkingMode: aws.String(new.RegionLinkingMode.ValueString()),
		}

		if !new.LinkedRegions.IsNull() && !new.LinkedRegions.IsUnknown() {
			var regions []string
			response.Diagnostics.Append(new.LinkedRegions.ElementsAs(ctx, &regions, false)...)
			if response.Diagnostics.HasError() {
				return
			}
			input.LinkedRegions = regions
		}

		output, err := conn.UpdateAggregatorV2(ctx, &input)

		if err != nil {
			response.Diagnostics.AddError("updating Security Hub V2 Aggregator", err.Error())
			return
		}

		new.AggregationRegion = fwflex.StringToFramework(ctx, output.AggregationRegion)
		if output.LinkedRegions != nil {
			regionsList, diags := types.ListValueFrom(ctx, types.StringType, output.LinkedRegions)
			response.Diagnostics.Append(diags...)
			new.LinkedRegions = fwtypes.ListValueOf[types.String]{ListValue: regionsList}
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &new)...)
}

func (r *aggregatorV2Resource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data aggregatorV2ResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	input := securityhub.DeleteAggregatorV2Input{
		AggregatorV2Arn: aws.String(data.ARN.ValueString()),
	}
	_, err := conn.DeleteAggregatorV2(ctx, &input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError("deleting Security Hub V2 Aggregator", err.Error())
	}
}

func findAggregatorV2ByARN(ctx context.Context, conn *securityhub.Client, arn string) (*securityhub.GetAggregatorV2Output, error) {
	input := securityhub.GetAggregatorV2Input{
		AggregatorV2Arn: aws.String(arn),
	}
	output, err := conn.GetAggregatorV2(ctx, &input)

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return nil, &retry.NotFoundError{
			LastError: err,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, tfresource.NewEmptyResultError()
	}

	return output, nil
}

type aggregatorV2ResourceModel struct {
	framework.WithRegionModel
	ARN               types.String                      `tfsdk:"arn"`
	AggregationRegion types.String                      `tfsdk:"aggregation_region"`
	LinkedRegions     fwtypes.ListValueOf[types.String] `tfsdk:"linked_regions"`
	RegionLinkingMode types.String                      `tfsdk:"region_linking_mode"`
	Tags              tftags.Map                        `tfsdk:"tags"`
	TagsAll           tftags.Map                        `tfsdk:"tags_all"`
}
