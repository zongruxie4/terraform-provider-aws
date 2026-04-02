// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

// DONOTCOPY: Copying old resources spreads bad habits. Use skaff instead.

package ec2

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
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

	r.SetDefaultCreateTimeout(30 * time.Minute)
	r.SetDefaultUpdateTimeout(30 * time.Minute)
	r.SetDefaultDeleteTimeout(30 * time.Minute)

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
			names.AttrARN:              framework.ARNAttributeComputedOnly(),
			names.AttrID:               framework.IDAttribute(),
			names.AttrAvailabilityZone: framework.ARNAttributeComputedOnly(),
			names.AttrIOPS: schema.Int32Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.UseStateForUnknown(),
				},
			},
			names.AttrSize: schema.Int64Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
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
	conn := r.Meta().EC2Client(ctx)

	var plan ebsVolumeCopyResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
	if resp.Diagnostics.HasError() {
		return
	}

	var input ec2.CopyVolumesInput
	smerr.AddEnrich(ctx, &resp.Diagnostics, flex.Expand(ctx, plan, &input, flex.WithFieldNamePrefix("EBSVolumeCopy")))
	if resp.Diagnostics.HasError() {
		return
	}
	input.TagSpecifications = getTagSpecificationsIn(ctx, awstypes.ResourceTypeVolume)

	out, err := conn.CopyVolumes(ctx, &input)
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, "Copy of "+plan.SourceVolumeID.String())
		return
	}
	if out == nil || out.Volumes == nil {
		smerr.AddError(ctx, &resp.Diagnostics, errors.New("empty output"), smerr.ID, "Copy of "+plan.SourceVolumeID.String())
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, flex.Flatten(ctx, out, &plan))
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout := r.CreateTimeout(ctx, plan.Timeouts)
	_, err = waitVolumeCreated(ctx, conn, *out.Volumes[0].VolumeId, createTimeout)
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, *out.Volumes[0].VolumeId)
		return
	}

	volume := out.Volumes[0]

	smerr.AddEnrich(ctx, &resp.Diagnostics, r.flatten(ctx, &volume, &plan))
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringPointerValue(volume.VolumeId)
	plan.ARN = fwflex.StringValueToFramework(ctx, r.Meta().RegionalARN(ctx, names.EC2, "volume/"+aws.ToString(volume.VolumeId)))

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &plan))
}

func (r *ebsVolumeCopyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	conn := r.Meta().EC2Client(ctx)

	var state ebsVolumeCopyResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := findEBSVolumeByID(ctx, conn, state.ID.ValueString())

	if retry.NotFound(err) {
		resp.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.String())
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, r.flatten(ctx, out, &state))
	if resp.Diagnostics.HasError() {
		return
	}
	setTagsOut(ctx, out.Tags)

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &state))
}

func (r *ebsVolumeCopyResource) flatten(ctx context.Context, ebsVolumeCopy *awstypes.Volume, data *ebsVolumeCopyResourceModel) (diags diag.Diagnostics) {
	diags.Append(fwflex.Flatten(ctx, ebsVolumeCopy, data)...)
	return diags
}

func (r *ebsVolumeCopyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	conn := r.Meta().EC2Client(ctx)

	// TIP: -- 2. Fetch the plan
	var plan, state ebsVolumeCopyResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}

	// TIP: -- 3. Get the difference between the plan and state, if any
	_, d := flex.Diff(ctx, plan, state)
	smerr.AddEnrich(ctx, &resp.Diagnostics, d)
	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, plan))

	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Iops.Equal(state.Iops) || !plan.Size.Equal(state.Size) || !plan.Throughput.Equal(state.Throughput) || !plan.VolumeType.Equal(state.VolumeType) {
		var input ec2.ModifyVolumeInput
		smerr.AddEnrich(ctx, &resp.Diagnostics, flex.Expand(ctx, plan, &input, flex.WithFieldNamePrefix("EBSVolumeCopy")))
		if resp.Diagnostics.HasError() {
			return
		}

		input.VolumeId = aws.String(state.ID.ValueString())
		out, err := conn.ModifyVolume(ctx, &input)
		if err != nil {
			smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, "something wrong?"+plan.ID.String())
			return
		}
		if out == nil || out.VolumeModification == nil {
			smerr.AddError(ctx, &resp.Diagnostics, errors.New("empty output"), smerr.ID, plan.ID.String())
			return
		}

		smerr.AddEnrich(ctx, &resp.Diagnostics, flex.Flatten(ctx, out, &plan))
		if resp.Diagnostics.HasError() {
			return
		}

		updateTimeout := r.UpdateTimeout(ctx, plan.Timeouts)
		_, err = waitVolumeUpdated(ctx, conn, plan.ID.ValueString(), updateTimeout)
		if err != nil {
			smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.String())
			return
		}
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

	input := ec2.DeleteVolumeInput{
		VolumeId: state.ID.ValueStringPointer(),
	}

	_, err := conn.DeleteVolume(ctx, &input)
	if err != nil {
		if retry.NotFound(err) {
			return
		}

		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.String())
		return
	}

	deleteTimeout := r.DeleteTimeout(ctx, state.Timeouts)
	_, err = waitEBSVolumeCopyDeleted(ctx, conn, state.ID.ValueString(), deleteTimeout)
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.String())
		return
	}
}

// TIP: ==== TERRAFORM IMPORTING ====
// The built-in import function, and Import ID Handler, if any, should handle populating the required
// attributes from the Import ID or Resource Identity.
// In some cases, additional attributes must be set when importing.
// Adding a custom ImportState function can handle those.
//
// See more:
// https://hashicorp.github.io/terraform-provider-aws/add-resource-identity-support/
// func (r *ebsVolumeCopyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
// 	r.WithImportByIdentity.ImportState(ctx, req, resp)
//
// 	// Set needed attribute values here
// }

// TIP: ==== STATUS CONSTANTS ====
// Create constants for states and statuses if the service does not
// already have suitable constants. We prefer that you use the constants
// provided in the service if available (e.g., awstypes.StatusInProgress).
const (
	statusChangePending = "Pending"
	statusDeleting      = "Deleting"
	statusNormal        = "Normal"
	statusUpdated       = "Updated"
)

// func waitEBSVolumeCopyUpdated(ctx context.Context, conn *ec2.Client, id string, timeout time.Duration) (*awstypes.Volume, error) {
// 	stateConf := &retry.StateChangeConf{
// 		Pending:                   []string{statusChangePending},
// 		Target:                    []string{statusUpdated},
// 		Refresh:                   statusEBSVolumeCopy(conn, id),
// 		Timeout:                   timeout,
// 		NotFoundChecks:            20,
// 		ContinuousTargetOccurence: 2,
// 	}

// 	outputRaw, err := stateConf.WaitForStateContext(ctx)
// 	if out, ok := outputRaw.(*awstypes.Volume); ok {
// 		return out, smarterr.NewError(err)
// 	}

// 	return nil, smarterr.NewError(err)
// }

func waitEBSVolumeCopyDeleted(ctx context.Context, conn *ec2.Client, id string, timeout time.Duration) (*awstypes.Volume, error) {
	return waitVolumeDeleted(ctx, conn, id, timeout)
}

// func statusEBSVolumeCopy(conn *ec2.Client, id string) retry.StateRefreshFunc {
// 	return func(ctx context.Context) (any, string, error) {
// 		out, err := findEBSVolumeByID(ctx, conn, id)
// 		if retry.NotFound(err) {
// 			return nil, "", nil
// 		}

// 		if err != nil {
// 			return nil, "", smarterr.NewError(err)
// 		}

// 		return out, aws.ToString((*string)(&out.State)), nil
// 	}
// }

// func findEBSVolumeCopyByID(ctx context.Context, conn *ec2.Client, id string) (*awstypes.Volume, error) {
// 	input := ec2.DescribeVolumesInput{
// 		VolumeIds: []string{id},
// 	}

// 	out, err := conn.DescribeVolumes(ctx, &input)
// 	if err != nil {
// 		if retry.NotFound(err) {
// 			return nil, smarterr.NewError(&retry.NotFoundError{
// 				LastError: err,
// 			})
// 		}

// 		return nil, smarterr.NewError(err)
// 	}

// 	if out == nil || out.Volumes == nil {
// 		return nil, smarterr.NewError(tfresource.NewEmptyResultError())
// 	}

// 	return &out.Volumes[0], nil
// }

// TIP: ==== DATA STRUCTURES ====
// With Terraform Plugin-Framework configurations are deserialized into
// Go types, providing type safety without the need for type assertions.
// These structs should match the schema definition exactly, and the `tfsdk`
// tag value should match the attribute name.
//
// Nested objects are represented in their own data struct. These will
// also have a corresponding attribute type mapping for use inside flex
// functions.
//
// See more:
// https://developer.hashicorp.com/terraform/plugin/framework/handling-data/accessing-values
type ebsVolumeCopyResourceModel struct {
	framework.WithRegionModel
	ARN              types.String   `tfsdk:"arn"`
	AvailabilityZone types.String   `tfsdk:"availability_zone"`
	ID               types.String   `tfsdk:"id"`
	Iops             types.Int32    `tfsdk:"iops"`
	Size             types.Int64    `tfsdk:"size"`
	SourceVolumeID   types.String   `tfsdk:"source_volume_id"`
	Tags             tftags.Map     `tfsdk:"tags"`
	TagsAll          tftags.Map     `tfsdk:"tags_all"`
	Throughput       types.Int32    `tfsdk:"throughput"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
	VolumeType       types.String   `tfsdk:"volume_type"`
}

// type complexArgumentModel struct {
// 	NestedRequired types.String `tfsdk:"nested_required"`
// 	NestedOptional types.String `tfsdk:"nested_optional"`
// }

// TIP: ==== IMPORT ID HANDLER ====
// When a resource type has a Resource Identity with multiple attributes, it needs a handler to
// parse the Import ID used for the `terraform import` command or an `import` block with the `id` parameter.
//
// The parser takes the string value of the Import ID and returns:
// * A string value that is typically ignored. See documentation for more details.
// * A map of the resource attributes derived from the Import ID.
// * An error value if there are parsing errors.
//
// For more information, see https://hashicorp.github.io/terraform-provider-aws/resource-identity/#plugin-framework
// var (
// 	_ inttypes.ImportIDParser = ebsVolumeCopyImportID{}
// )

// type ebsVolumeCopyImportID struct{}

// func (ebsVolumeCopyImportID) Parse(id string) (string, map[string]string, error) {
// 	someValue, anotherValue, found := strings.Cut(id, intflex.ResourceIdSeparator)
// 	if !found {
// 		return "", nil, fmt.Errorf("id \"%s\" should be in the format <some-value>"+intflex.ResourceIdSeparator+"<another-value>", id)
// 	}

// 	result := map[string]string{
// 		"some-value":    someValue,
// 		"another-value": anotherValue,
// 	}

// 	return id, result, nil
// }

// TIP: ==== SWEEPERS ====
// When acceptance testing resources, interrupted or failed tests may
// leave behind orphaned resources in an account. To facilitate cleaning
// up lingering resources, each resource implementation should include
// a corresponding "sweeper" function.
//
// The sweeper function lists all resources of a given type and sets the
// appropriate identifers required to delete the resource via the Delete
// method implemented above.
//
// Once the sweeper function is implemented, register it in sweep.go
// as follows:
//
//	awsv2.Register("aws_ec2_ebs_volume_copy", sweepEBSVolumeCopys)
//
// See more:
// https://hashicorp.github.io/terraform-provider-aws/running-and-writing-acceptance-tests/#acceptance-test-sweepers
// func sweepEBSVolumeCopys(ctx context.Context, client *conns.AWSClient) ([]sweep.Sweepable, error) {
// 	input := ec2.ListEBSVolumeCopysInput{}
// 	conn := client.EC2Client(ctx)
// 	var sweepResources []sweep.Sweepable

// 	pages := ec2.NewListEBSVolumeCopysPaginator(conn, &input)
// 	for pages.HasMorePages() {
// 		page, err := pages.NextPage(ctx)
// 		if err != nil {
// 			return nil, smarterr.NewError(err)
// 		}

// 		for _, v := range page.EBSVolumeCopys {
// 			sweepResources = append(sweepResources, sweepfw.NewSweepResource(newEBSVolumeCopyResource, client,
// 				sweepfw.NewAttribute(names.AttrID, aws.ToString(v.EBSVolumeCopyId))),
// 			)
// 		}
// 	}

// 	return sweepResources, nil
// }
