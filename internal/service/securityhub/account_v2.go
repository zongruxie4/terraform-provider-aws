// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package securityhub

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	awstypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	frameworkpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_securityhub_v2_account", name="V2 Account")
// @Tags(identifierAttribute="hub_arn")
// @Testing(existsType="github.com/aws/aws-sdk-go-v2/service/securityhub;securityhub;securityhub.DescribeSecurityHubV2Output")
// @Testing(serialize=true)
func newV2AccountResource(_ context.Context) (resource.ResourceWithConfigure, error) {
	return &v2AccountResource{}, nil
}

type v2AccountResource struct {
	framework.ResourceWithModel[v2AccountResourceModel]
}

func (r *v2AccountResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrID: framework.IDAttribute(),
			"hub_arn": schema.StringAttribute{
				Computed:    true,
				Description: "The ARN of the Security Hub V2 resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
	}
}

func (r *v2AccountResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data v2AccountResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	output, err := conn.EnableSecurityHubV2(ctx, &securityhub.EnableSecurityHubV2Input{
		Tags: getTagsIn(ctx),
	})

	if err != nil {
		response.Diagnostics.AddError("creating Security Hub V2 Account", err.Error())
		return
	}

	data.HubARN = types.StringPointerValue(output.HubV2Arn)
	data.setID()

	response.Diagnostics.Append(response.State.Set(ctx, data)...)
}

func (r *v2AccountResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data v2AccountResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	hub, err := findV2Account(ctx, conn)

	if retry.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("reading Security Hub V2 Account (%s)", data.ID.ValueString()), err.Error())
		return
	}

	data.HubARN = types.StringPointerValue(hub.HubV2Arn)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *v2AccountResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var old, new v2AccountResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &old)...)
	if response.Diagnostics.HasError() {
		return
	}
	response.Diagnostics.Append(request.Plan.Get(ctx, &new)...)
	if response.Diagnostics.HasError() {
		return
	}

	// Tags are updated automatically by the framework via @Tags annotation.
	// Preserve computed fields from state.
	new.HubARN = old.HubARN
	new.ID = old.ID

	response.Diagnostics.Append(response.State.Set(ctx, &new)...)
}

func (r *v2AccountResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data v2AccountResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SecurityHubClient(ctx)

	_, err := conn.DisableSecurityHubV2(ctx, &securityhub.DisableSecurityHubV2Input{})

	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("deleting Security Hub V2 Account (%s)", data.ID.ValueString()), err.Error())
	}
}

func (r *v2AccountResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	// Account is a singleton — Read uses DescribeSecurityHubV2 which needs no identifier.
	// Set a placeholder ID; Read will populate the real hub_arn.
	response.Diagnostics.Append(response.State.SetAttribute(ctx, frameworkpath.Root("id"), request.ID)...)
}

func findV2Account(ctx context.Context, conn *securityhub.Client) (*securityhub.DescribeSecurityHubV2Output, error) {
	output, err := conn.DescribeSecurityHubV2(ctx, &securityhub.DescribeSecurityHubV2Input{})

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

type v2AccountResourceModel struct {
	framework.WithRegionModel
	HubARN  types.String `tfsdk:"hub_arn"`
	ID      types.String `tfsdk:"id"`
	Tags    tftags.Map   `tfsdk:"tags"`
	TagsAll tftags.Map   `tfsdk:"tags_all"`
}

func (data *v2AccountResourceModel) InitFromID() error {
	data.HubARN = data.ID
	return nil
}

func (data *v2AccountResourceModel) setID() {
	data.ID = data.HubARN
}
