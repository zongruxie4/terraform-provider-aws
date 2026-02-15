// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networkmanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	awstypes "github.com/aws/aws-sdk-go-v2/service/networkmanager/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

// @FrameworkResource("aws_networkmanager_attachment_routing_policy_label", name="Attachment Routing Policy Label")
func newAttachmentRoutingPolicyLabelResource(_ context.Context) (resource.ResourceWithConfigure, error) {
	return &attachmentRoutingPolicyLabelResource{}, nil
}

const (
	ResNameAttachmentRoutingPolicyLabel = "Attachment Routing Policy Label"
)

type attachmentRoutingPolicyLabelResource struct {
	framework.ResourceWithModel[attachmentRoutingPolicyLabelResourceModel]
}

type attachmentRoutingPolicyLabelResourceModel struct {
	AttachmentID       types.String `tfsdk:"attachment_id"`
	CoreNetworkID      types.String `tfsdk:"core_network_id"`
	ID                 types.String `tfsdk:"id"`
	RoutingPolicyLabel types.String `tfsdk:"routing_policy_label"`
}

func (r *attachmentRoutingPolicyLabelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networkmanager_attachment_routing_policy_label"
}

func (r *attachmentRoutingPolicyLabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"attachment_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"core_network_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": framework.IDAttribute(),
			"routing_policy_label": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *attachmentRoutingPolicyLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	conn := r.Meta().NetworkManagerClient(ctx)

	var plan attachmentRoutingPolicyLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	coreNetworkID := plan.CoreNetworkID.ValueString()
	attachmentID := plan.AttachmentID.ValueString()
	routingPolicyLabel := plan.RoutingPolicyLabel.ValueString()

	input := &networkmanager.PutAttachmentRoutingPolicyLabelInput{
		CoreNetworkId:      aws.String(coreNetworkID),
		AttachmentId:       aws.String(attachmentID),
		RoutingPolicyLabel: aws.String(routingPolicyLabel),
	}

	_, err := conn.PutAttachmentRoutingPolicyLabel(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("creating Network Manager Attachment Routing Policy Label (%s/%s)", coreNetworkID, attachmentID),
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(attachmentRoutingPolicyLabelCreateResourceID(coreNetworkID, attachmentID))

	if _, err := waitAttachmentAvailable(ctx, conn, coreNetworkID, attachmentID, 20*time.Minute); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("waiting for Network Manager Attachment (%s) to become available after applying routing policy label", attachmentID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *attachmentRoutingPolicyLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	conn := r.Meta().NetworkManagerClient(ctx)

	var state attachmentRoutingPolicyLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	coreNetworkID, attachmentID, err := attachmentRoutingPolicyLabelParseResourceID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parsing resource ID", err.Error())
		return
	}

	label, err := findRoutingPolicyLabelByTwoPartKey(ctx, conn, coreNetworkID, attachmentID)
	if retry.NotFound(err) {
		resp.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("reading Network Manager Attachment Routing Policy Label (%s)", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	state.CoreNetworkID = types.StringValue(coreNetworkID)
	state.AttachmentID = types.StringValue(attachmentID)
	state.RoutingPolicyLabel = types.StringPointerValue(label)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *attachmentRoutingPolicyLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	conn := r.Meta().NetworkManagerClient(ctx)

	var state attachmentRoutingPolicyLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	coreNetworkID, attachmentID, err := attachmentRoutingPolicyLabelParseResourceID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("parsing resource ID", err.Error())
		return
	}

	if _, err := waitAttachmentAvailable(ctx, conn, coreNetworkID, attachmentID, 20*time.Minute); err != nil {
		// If the attachment itself is gone, nothing to delete.
		if retry.NotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("waiting for Network Manager Attachment (%s) to become available before removing routing policy label", attachmentID),
			err.Error(),
		)
		return
	}

	input := &networkmanager.RemoveAttachmentRoutingPolicyLabelInput{
		CoreNetworkId: aws.String(coreNetworkID),
		AttachmentId:  aws.String(attachmentID),
	}

	_, err = conn.RemoveAttachmentRoutingPolicyLabel(ctx, input)
	if errs.IsA[*awstypes.ResourceNotFoundException](err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("deleting Network Manager Attachment Routing Policy Label (%s)", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	if _, err := waitAttachmentAvailable(ctx, conn, coreNetworkID, attachmentID, 20*time.Minute); err != nil {
		// If the attachment itself is gone after remove, that's fine.
		if retry.NotFound(err) {
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("waiting for Network Manager Attachment (%s) to become available after removing routing policy label", attachmentID),
			err.Error(),
		)
		return
	}
}

func (r *attachmentRoutingPolicyLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	coreNetworkID, attachmentID, err := attachmentRoutingPolicyLabelParseResourceID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("parsing import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("core_network_id"), coreNetworkID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("attachment_id"), attachmentID)...)
}

const attachmentRoutingPolicyLabelIDSeparator = ","

func attachmentRoutingPolicyLabelCreateResourceID(coreNetworkID, attachmentID string) string {
	return strings.Join([]string{coreNetworkID, attachmentID}, attachmentRoutingPolicyLabelIDSeparator)
}

func attachmentRoutingPolicyLabelParseResourceID(id string) (string, string, error) {
	parts := strings.Split(id, attachmentRoutingPolicyLabelIDSeparator)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("unexpected format for ID (%[1]s), expected CORE-NETWORK-ID%[2]sATTACHMENT-ID", id, attachmentRoutingPolicyLabelIDSeparator)
}

func findAttachmentByTwoPartKey(ctx context.Context, conn *networkmanager.Client, coreNetworkID, attachmentID string) (*awstypes.Attachment, error) {
	input := &networkmanager.ListAttachmentsInput{
		CoreNetworkId: aws.String(coreNetworkID),
	}

	pages := networkmanager.NewListAttachmentsPaginator(conn, input)
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)

		if errs.IsA[*awstypes.ResourceNotFoundException](err) {
			return nil, &retry.NotFoundError{
				LastError: err,
			}
		}

		if err != nil {
			return nil, err
		}

		for _, attachment := range page.Attachments {
			if aws.ToString(attachment.AttachmentId) == attachmentID {
				return &attachment, nil
			}
		}
	}

	return nil, tfresource.NewEmptyResultError()
}

func statusAttachment(ctx context.Context, conn *networkmanager.Client, coreNetworkID, attachmentID string) retry.StateRefreshFunc {
	return func(_ context.Context) (interface{}, string, error) {
		output, err := findAttachmentByTwoPartKey(ctx, conn, coreNetworkID, attachmentID)

		if retry.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, string(output.State), nil
	}
}

func waitAttachmentAvailable(ctx context.Context, conn *networkmanager.Client, coreNetworkID, attachmentID string, timeout time.Duration) (*awstypes.Attachment, error) {
	stateConf := &retry.StateChangeConf{
		Pending: enum.Slice(awstypes.AttachmentStatePendingNetworkUpdate, awstypes.AttachmentStateUpdating),
		Target:  enum.Slice(awstypes.AttachmentStateAvailable),
		Timeout: timeout,
		Refresh: statusAttachment(ctx, conn, coreNetworkID, attachmentID),
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.Attachment); ok {
		return output, err
	}

	return nil, err
}
