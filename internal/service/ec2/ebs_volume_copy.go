// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

// DONOTCOPY: Copying old resources spreads bad habits. Use skaff instead.

package ec2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hashicorp/aws-sdk-go-base/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_ebs_volume_copy", name="EBS Volume Copy")
// @Tags(identifierAttribute="id")
func newEBSVolumeCopyResource(_ context.Context) (resource.ResourceWithConfigure, error) {
	r := &ebsVolumeCopyResource{}

	r.SetDefaultCreateTimeout(5 * time.Minute)
	r.SetDefaultUpdateTimeout(5 * time.Minute)
	r.SetDefaultDeleteTimeout(10 * time.Minute)

	return r, nil
}

type ebsVolumeCopyResource struct {
	framework.ResourceWithModel[ebsVolumeCopyResourceModel]
	framework.WithTimeouts
	framework.WithImportByID
}

func (r *ebsVolumeCopyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			names.AttrID:  framework.IDAttribute(),
			names.AttrAvailabilityZone: schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrIOPS: schema.Int32Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			names.AttrSize: schema.Int32Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
			names.AttrThroughput: schema.Int32Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			names.AttrVolumeType: schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_volume_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			names.AttrTimeouts: timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *ebsVolumeCopyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	awsClient := r.Meta()
	conn := awsClient.EC2Client(ctx)

	var plan ebsVolumeCopyResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
	if resp.Diagnostics.HasError() {
		return
	}

	var input ec2.CopyVolumesInput
	smerr.AddEnrich(ctx, &resp.Diagnostics, fwflex.Expand(ctx, plan, &input))
	if resp.Diagnostics.HasError() {
		return
	}
	input.TagSpecifications = getTagSpecificationsIn(ctx, awstypes.ResourceTypeVolume)

	out, err := conn.CopyVolumes(ctx, &input)
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, "source_volume_id", plan.SourceVolumeID.String())
		return
	}
	if out == nil || len(out.Volumes) == 0 || out.Volumes[0].VolumeId == nil {
		smerr.AddError(ctx, &resp.Diagnostics, errors.New("empty output"), "source_volume_id", plan.SourceVolumeID.String())
		return
	}

	id := aws.ToString(out.Volumes[0].VolumeId)
	plan.ID = types.StringValue(id)
	plan.ARN = types.StringValue(ebsVolumeARN(ctx, awsClient, id))

	waitOut, err := waitVolumeCreated(ctx, conn, id, r.CreateTimeout(ctx, plan.Timeouts))
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, id)
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, fwflex.Flatten(ctx, waitOut, &plan))
	if resp.Diagnostics.HasError() {
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &plan))
}

func (r *ebsVolumeCopyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	awsClient := r.Meta()
	conn := awsClient.EC2Client(ctx)

	var state ebsVolumeCopyResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	out, err := findEBSVolumeByID(ctx, conn, id)
	if retry.NotFound(err) {
		resp.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.String())
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, fwflex.Flatten(ctx, out, &state))
	if resp.Diagnostics.HasError() {
		return
	}
	state.ARN = types.StringValue(ebsVolumeARN(ctx, awsClient, id))

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &state))
}

func (r *ebsVolumeCopyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	awsClient := r.Meta()
	conn := awsClient.EC2Client(ctx)

	var plan, state ebsVolumeCopyResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	if !plan.Iops.Equal(state.Iops) ||
		!plan.Size.Equal(state.Size) ||
		!plan.Throughput.Equal(state.Throughput) ||
		!plan.VolumeType.Equal(state.VolumeType) {
		input := ec2.ModifyVolumeInput{
			VolumeId: aws.String(id),
		}

		if !plan.Iops.Equal(state.Iops) && !plan.Iops.IsUnknown() && !plan.Iops.IsNull() {
			input.Iops = plan.Iops.ValueInt32Pointer()
		}

		if !plan.Size.Equal(state.Size) && !plan.Size.IsUnknown() && !plan.Size.IsNull() {
			input.Size = plan.Size.ValueInt32Pointer()
		}

		if !plan.Throughput.IsUnknown() && !plan.Throughput.IsNull() {
			if v := plan.Throughput.ValueInt32(); v > 0 && plan.VolumeType.ValueString() == string(awstypes.VolumeTypeGp3) {
				input.Throughput = aws.Int32(v)
			}
		}

		if !plan.VolumeType.Equal(state.VolumeType) {
			volumeType := awstypes.VolumeType(plan.VolumeType.ValueString())
			input.VolumeType = volumeType

			if (volumeType == awstypes.VolumeTypeIo1 || volumeType == awstypes.VolumeTypeIo2 || volumeType == awstypes.VolumeTypeGp3) && !plan.Iops.IsUnknown() && !plan.Iops.IsNull() {
				input.Iops = plan.Iops.ValueInt32Pointer()
			}
		}

		out, err := conn.ModifyVolume(ctx, &input)
		if err != nil {
			smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.String())
			return
		}
		if out == nil || out.VolumeModification == nil {
			smerr.AddError(ctx, &resp.Diagnostics, errors.New("empty output"), smerr.ID, plan.ID.String())
			return
		}

		waitOut, err := waitVolumeUpdated(ctx, conn, plan.ID.ValueString(), r.UpdateTimeout(ctx, plan.Timeouts))
		if err != nil {
			smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.String())
			return
		}

		smerr.AddEnrich(ctx, &resp.Diagnostics, fwflex.Flatten(ctx, waitOut, &plan))
		if resp.Diagnostics.HasError() {
			return
		}
		state.ARN = types.StringValue(ebsVolumeARN(ctx, awsClient, id))
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &plan))
}

func (r *ebsVolumeCopyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	conn := r.Meta().EC2Client(ctx)

	var state ebsVolumeCopyResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	input := ec2.DeleteVolumeInput{
		VolumeId: aws.String(id),
	}

	_, err := conn.DeleteVolume(ctx, &input)
	if err != nil {
		if retry.NotFound(err) || tfawserr.ErrCodeEquals(err, errCodeInvalidVolumeNotFound) {
			return
		}

		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, id)
		return
	}

	_, err = waitVolumeDeleted(ctx, conn, id, r.DeleteTimeout(ctx, state.Timeouts))
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, id)
		return
	}
}

func (r *ebsVolumeCopyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var volumeType types.String
	var iops, throughput types.Int32

	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Config.GetAttribute(ctx, path.Root(names.AttrVolumeType), &volumeType))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Config.GetAttribute(ctx, path.Root(names.AttrIOPS), &iops))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Config.GetAttribute(ctx, path.Root(names.AttrThroughput), &throughput))
	if resp.Diagnostics.HasError() {
		return
	}

	if volumeType.IsNull() || volumeType.IsUnknown() {
		return
	}

	vt := awstypes.VolumeType(volumeType.ValueString())

	if !throughput.IsNull() && !throughput.IsUnknown() && throughput.ValueInt32() > 0 && vt != awstypes.VolumeTypeGp3 {
		resp.Diagnostics.AddAttributeError(
			path.Root(names.AttrThroughput),
			"Invalid Throughput Configuration",
			fmt.Sprintf("`throughput` must not be set when `volume_type` is %q.", vt),
		)
	}

	if !iops.IsNull() && !iops.IsUnknown() && iops.ValueInt32() > 0 {
		switch vt {
		case awstypes.VolumeTypeIo1, awstypes.VolumeTypeIo2, awstypes.VolumeTypeGp3:
		default:
			resp.Diagnostics.AddAttributeError(
				path.Root(names.AttrIOPS),
				"Invalid IOPS Configuration",
				fmt.Sprintf("`iops` must not be set when `volume_type` is %q.", vt),
			)
		}
	}

	if (vt == awstypes.VolumeTypeIo1 || vt == awstypes.VolumeTypeIo2) && (iops.IsNull() || iops.IsUnknown() || iops.ValueInt32() == 0) {
		resp.Diagnostics.AddAttributeError(
			path.Root(names.AttrIOPS),
			"Missing IOPS Configuration",
			fmt.Sprintf("`iops` must be set when `volume_type` is %q.", vt),
		)
	}
}

func (r *ebsVolumeCopyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var planVolumeType, stateVolumeType types.String
	var configIops, configThroughput types.Int32

	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.GetAttribute(ctx, path.Root(names.AttrVolumeType), &planVolumeType))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.GetAttribute(ctx, path.Root(names.AttrVolumeType), &stateVolumeType))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Config.GetAttribute(ctx, path.Root(names.AttrIOPS), &configIops))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Config.GetAttribute(ctx, path.Root(names.AttrThroughput), &configThroughput))
	if resp.Diagnostics.HasError() {
		return
	}

	if !planVolumeType.IsNull() && !planVolumeType.IsUnknown() {
		vt := awstypes.VolumeType(planVolumeType.ValueString())

		if !configThroughput.IsNull() && !configThroughput.IsUnknown() && configThroughput.ValueInt32() > 0 && vt != awstypes.VolumeTypeGp3 {
			resp.Diagnostics.AddAttributeError(
				path.Root(names.AttrThroughput),
				"Invalid Throughput Configuration",
				fmt.Sprintf("`throughput` must not be set when `volume_type` is %q.", vt),
			)
		}

		if !configIops.IsNull() && !configIops.IsUnknown() && configIops.ValueInt32() > 0 {
			switch vt {
			case awstypes.VolumeTypeIo1, awstypes.VolumeTypeIo2, awstypes.VolumeTypeGp3:
			default:
				resp.Diagnostics.AddAttributeError(
					path.Root(names.AttrIOPS),
					"Invalid IOPS Configuration",
					fmt.Sprintf("`iops` must not be set when `volume_type` is %q.", vt),
				)
			}
		}
	}

	if !planVolumeType.Equal(stateVolumeType) {
		if configIops.IsNull() && !configIops.IsUnknown() {
			smerr.AddEnrich(ctx, &resp.Diagnostics, resp.Plan.SetAttribute(ctx, path.Root(names.AttrIOPS), types.Int32Unknown()))
		}

		if configThroughput.IsNull() && !configThroughput.IsUnknown() {
			smerr.AddEnrich(ctx, &resp.Diagnostics, resp.Plan.SetAttribute(ctx, path.Root(names.AttrThroughput), types.Int32Unknown()))
		}
	}
}

type ebsVolumeCopyResourceModel struct {
	framework.WithRegionModel
	ARN              types.String   `tfsdk:"arn"`
	AvailabilityZone types.String   `tfsdk:"availability_zone"`
	ID               types.String   `tfsdk:"id"`
	Iops             types.Int32    `tfsdk:"iops"`
	Size             types.Int32    `tfsdk:"size"`
	SourceVolumeID   types.String   `tfsdk:"source_volume_id"`
	Tags             tftags.Map     `tfsdk:"tags"`
	TagsAll          tftags.Map     `tfsdk:"tags_all"`
	Throughput       types.Int32    `tfsdk:"throughput"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
	VolumeType       types.String   `tfsdk:"volume_type"`
}
