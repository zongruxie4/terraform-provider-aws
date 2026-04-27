// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package glue

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YakDriver/smarterr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	awstypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// Function annotations are used for resource registration to the Provider. DO NOT EDIT.
// @FrameworkResource("aws_glue_catalog", name="Catalog")
// @Tags(identifierAttribute="arn")
func newResourceCatalog(_ context.Context) (resource.ResourceWithConfigure, error) {
	r := &resourceCatalog{}

	r.SetDefaultCreateTimeout(30 * time.Minute)
	r.SetDefaultUpdateTimeout(30 * time.Minute)
	r.SetDefaultDeleteTimeout(30 * time.Minute)

	return r, nil
}

const (
	ResNameCatalog      = "Catalog"
	s3TablesCatalogName = "s3tablescatalog"
)

type resourceCatalog struct {
	framework.ResourceWithModel[resourceCatalogModel]
	framework.WithTimeouts
	framework.WithImportByID
}

func (r *resourceCatalog) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"allow_full_table_external_data_access": schema.BoolAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			names.AttrCatalogID: schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrDescription: schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrID: framework.IDAttribute(),
			names.AttrName: schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
		Blocks: map[string]schema.Block{
			"federated_catalog": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[federatedCatalogModel](ctx),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"connection_name": schema.StringAttribute{
							Optional: true,
						},
						names.AttrIdentifier: schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},
			"target_redshift_catalog": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[targetRedshiftCatalogModel](ctx),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"catalog_arn": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			"catalog_properties": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[catalogPropertiesModel](ctx),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"custom_properties": schema.MapAttribute{
							CustomType:  fwtypes.MapOfStringType,
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"data_lake_access_properties": schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[dataLakeAccessPropertiesModel](ctx),
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"catalog_type": schema.StringAttribute{
										Optional: true,
									},
									"data_lake_access": schema.BoolAttribute{
										Optional: true,
									},
									"data_transfer_role": schema.StringAttribute{
										Optional: true,
									},
									names.AttrKMSKey: schema.StringAttribute{
										Optional: true,
									},
								},
							},
						},
						"iceberg_optimization_properties": schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[icebergOptimizationPropertiesModel](ctx),
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"compaction": schema.MapAttribute{
										CustomType:  fwtypes.MapOfStringType,
										Optional:    true,
										ElementType: types.StringType,
									},
									"orphan_file_deletion": schema.MapAttribute{
										CustomType:  fwtypes.MapOfStringType,
										Optional:    true,
										ElementType: types.StringType,
									},
									"retention": schema.MapAttribute{
										CustomType:  fwtypes.MapOfStringType,
										Optional:    true,
										ElementType: types.StringType,
									},
									names.AttrRoleARN: schema.StringAttribute{
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			names.AttrTimeouts: timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *resourceCatalog) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	conn := r.Meta().GlueClient(ctx)
	var plan resourceCatalogModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.FederatedCatalog.IsNull() && plan.CatalogProperties.IsNull() && plan.TargetRedshiftCatalog.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Required Configuration",
			"At least one of 'federated_catalog', 'catalog_properties', or 'target_redshift_catalog' must be specified.",
		)
		return
	}

	if plan.CatalogId.IsNull() || plan.CatalogId.ValueString() == "" {
		plan.CatalogId = types.StringValue(r.Meta().AccountID(ctx))
	}

	var input glue.CreateCatalogInput
	resp.Diagnostics.Append(fwflex.Expand(ctx, plan, &input)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input.CatalogInput = &awstypes.CatalogInput{}
	input.CatalogInput.CreateDatabaseDefaultPermissions = []awstypes.PrincipalPermissions{}
	input.CatalogInput.CreateTableDefaultPermissions = []awstypes.PrincipalPermissions{}
	resp.Diagnostics.Append(fwflex.Expand(ctx, plan, input.CatalogInput)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle enum conversion for AllowFullTableExternalDataAccess
	if !plan.AllowFullTableExternalDataAccess.IsNull() {
		if plan.AllowFullTableExternalDataAccess.ValueBool() {
			input.CatalogInput.AllowFullTableExternalDataAccess = awstypes.AllowFullTableExternalDataAccessEnumTrue
		} else {
			input.CatalogInput.AllowFullTableExternalDataAccess = awstypes.AllowFullTableExternalDataAccessEnumFalse
		}
	}

	input.Tags = getTagsIn(ctx)

	_, err := conn.CreateCatalog(ctx, &input)
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.Name.ValueString())
		return
	}

	catalogId := plan.CatalogId.ValueString()
	catalogName := plan.Name.ValueString()
	if catalogName == s3TablesCatalogName {
		catalogId = fmt.Sprintf("%s:%s", catalogId, catalogName)
	}
	id := fmt.Sprintf("%s,%s", catalogId, catalogName)
	plan.ID = types.StringValue(id)

	catalog, err := waitCatalogCreated(ctx, conn, id, r.CreateTimeout(ctx, plan.Timeouts))
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, id)
		return
	}

	// Set computed values
	plan.CatalogId = fwflex.StringToFramework(ctx, catalog.CatalogId)

	if catalog.ResourceArn != nil {
		plan.ARN = types.StringValue(aws.ToString(catalog.ResourceArn))
	} else {
		partition := r.Meta().Partition(ctx)
		region := r.Meta().Region(ctx)
		accountID := r.Meta().AccountID(ctx)
		if catalogName == s3TablesCatalogName {
			plan.ARN = types.StringValue(fmt.Sprintf("arn:%s:glue:%s:%s:catalog/%s", partition, region, accountID, catalogName))
		} else {
			plan.ARN = types.StringValue(fmt.Sprintf("arn:%s:glue:%s:%s:catalog", partition, region, accountID))
		}
	}

	switch catalog.AllowFullTableExternalDataAccess {
	case awstypes.AllowFullTableExternalDataAccessEnumTrue:
		plan.AllowFullTableExternalDataAccess = types.BoolValue(true)
	case awstypes.AllowFullTableExternalDataAccessEnumFalse:
		plan.AllowFullTableExternalDataAccess = types.BoolValue(false)
	}

	if catalog.FederatedCatalog != nil {
		plan.FederatedCatalog = fwtypes.NewListNestedObjectValueOfPtrMust(ctx, &federatedCatalogModel{
			ConnectionName: fwflex.StringToFramework(ctx, catalog.FederatedCatalog.ConnectionName),
			Identifier:     fwflex.StringToFramework(ctx, catalog.FederatedCatalog.Identifier),
		})
	}

	if catalog.TargetRedshiftCatalog != nil {
		plan.TargetRedshiftCatalog = fwtypes.NewListNestedObjectValueOfPtrMust(ctx, &targetRedshiftCatalogModel{
			CatalogArn: fwflex.StringToFramework(ctx, catalog.TargetRedshiftCatalog.CatalogArn),
		})
	}

	if !plan.CatalogProperties.IsNull() && catalog.CatalogProperties != nil {
		catalogPropsModel := catalogPropertiesModel{}
		resp.Diagnostics.Append(fwflex.Flatten(ctx, catalog.CatalogProperties, &catalogPropsModel)...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.CatalogProperties = fwtypes.NewListNestedObjectValueOfPtrMust(ctx, &catalogPropsModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceCatalog) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	conn := r.Meta().GlueClient(ctx)
	var state resourceCatalogModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := findCatalogByID(ctx, conn, state.ID.ValueString())
	if retry.NotFound(err) {
		resp.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.ValueString())
		return
	}

	// Flatten basic fields
	state.CatalogId = fwflex.StringToFramework(ctx, out.CatalogId)
	state.Name = fwflex.StringToFramework(ctx, out.Name)
	state.Description = fwflex.StringToFramework(ctx, out.Description)

	// Handle ARN
	if out.ResourceArn != nil {
		state.ARN = types.StringValue(aws.ToString(out.ResourceArn))
	} else {
		partition := r.Meta().Partition(ctx)
		region := r.Meta().Region(ctx)
		accountID := r.Meta().AccountID(ctx)
		catalogName := state.Name.ValueString()
		if catalogName == s3TablesCatalogName {
			state.ARN = types.StringValue(fmt.Sprintf("arn:%s:glue:%s:%s:catalog/%s", partition, region, accountID, catalogName))
		} else {
			state.ARN = types.StringValue(fmt.Sprintf("arn:%s:glue:%s:%s:catalog", partition, region, accountID))
		}
	}

	// Handle enum conversion for AllowFullTableExternalDataAccess
	switch out.AllowFullTableExternalDataAccess {
	case awstypes.AllowFullTableExternalDataAccessEnumTrue:
		state.AllowFullTableExternalDataAccess = types.BoolValue(true)
	case awstypes.AllowFullTableExternalDataAccessEnumFalse:
		state.AllowFullTableExternalDataAccess = types.BoolValue(false)
	}

	// Flatten FederatedCatalog
	if out.FederatedCatalog != nil {
		state.FederatedCatalog = fwtypes.NewListNestedObjectValueOfPtrMust(ctx, &federatedCatalogModel{
			ConnectionName: fwflex.StringToFramework(ctx, out.FederatedCatalog.ConnectionName),
			Identifier:     fwflex.StringToFramework(ctx, out.FederatedCatalog.Identifier),
		})
	}

	// Flatten TargetRedshiftCatalog
	if out.TargetRedshiftCatalog != nil {
		state.TargetRedshiftCatalog = fwtypes.NewListNestedObjectValueOfPtrMust(ctx, &targetRedshiftCatalogModel{
			CatalogArn: fwflex.StringToFramework(ctx, out.TargetRedshiftCatalog.CatalogArn),
		})
	}

	// Flatten CatalogProperties only if it was in the original config
	if !state.CatalogProperties.IsNull() && out.CatalogProperties != nil {
		catalogPropsModel := catalogPropertiesModel{}
		resp.Diagnostics.Append(fwflex.Flatten(ctx, out.CatalogProperties, &catalogPropsModel)...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.CatalogProperties = fwtypes.NewListNestedObjectValueOfPtrMust(ctx, &catalogPropsModel)
	}

	tags, err := listTags(ctx, r.Meta().GlueClient(ctx), state.ARN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("listing tags for Glue Catalog (%s)", state.ID.ValueString()),
			err.Error(),
		)
		return
	}
	setTagsOut(ctx, tags.Map())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *resourceCatalog) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resourceCatalogModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if non-tag attributes changed
	if !plan.CatalogId.Equal(state.CatalogId) ||
		!plan.Description.Equal(state.Description) ||
		!plan.Name.Equal(state.Name) ||
		!plan.FederatedCatalog.Equal(state.FederatedCatalog) ||
		!plan.TargetRedshiftCatalog.Equal(state.TargetRedshiftCatalog) ||
		!plan.CatalogProperties.Equal(state.CatalogProperties) {
		resp.Diagnostics.AddError(
			"Update Not Supported",
			"AWS Glue catalogs do not support updates. All attributes except tags require replacement.",
		)
		return
	}

	// Handle tags update
	if !plan.TagsAll.Equal(state.TagsAll) {
		if err := updateTags(ctx, r.Meta().GlueClient(ctx), state.ARN.ValueString(), state.TagsAll, plan.TagsAll); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("updating tags for Glue Catalog (%s)", plan.ID.ValueString()),
				err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *resourceCatalog) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	conn := r.Meta().GlueClient(ctx)
	var state resourceCatalogModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	catalogId, name, err := readCatalogResourceID(state.ID.ValueString())
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.ValueString())
		return
	}

	input := glue.DeleteCatalogInput{
		CatalogId: aws.String(resolveCatalogID(catalogId, name)),
	}

	_, err = conn.DeleteCatalog(ctx, &input)
	if err != nil {
		if errs.IsA[*awstypes.EntityNotFoundException](err) {
			return
		}

		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.ValueString())
		return
	}

	_, err = waitCatalogDeleted(ctx, conn, state.ID.ValueString(), r.DeleteTimeout(ctx, state.Timeouts))
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.ValueString())
		return
	}
}

func findCatalogByID(ctx context.Context, conn *glue.Client, id string) (*awstypes.Catalog, error) {
	catalogId, name, err := readCatalogResourceID(id)
	if err != nil {
		return nil, smarterr.NewError(err)
	}

	input := glue.GetCatalogInput{
		CatalogId: aws.String(catalogId),
	}

	out, err := conn.GetCatalog(ctx, &input)
	if err != nil {
		if errs.IsA[*awstypes.EntityNotFoundException](err) {
			return nil, smarterr.NewError(&retry.NotFoundError{
				LastError: err,
			})
		}
		// Lake Formation returns AccessDeniedException when catalog doesn't exist
		// and caller lacks Lake Formation permissions
		if errs.IsAErrorMessageContains[*awstypes.AccessDeniedException](err, "Lake Formation permission") {
			return nil, smarterr.NewError(&retry.NotFoundError{
				LastError: err,
			})
		}

		return nil, smarterr.NewError(err)
	}

	if out == nil || out.Catalog == nil {
		return nil, smarterr.NewError(tfresource.NewEmptyResultError())
	}

	actualName := aws.ToString(out.Catalog.Name)
	if actualName != name {
		return nil, smarterr.NewError(&retry.NotFoundError{
			Message: fmt.Sprintf("catalog name mismatch: expected %s, got %s", name, actualName),
		})
	}

	return out.Catalog, nil
}

const (
	statusAvailable = "available"
)

func waitCatalogCreated(ctx context.Context, conn *glue.Client, id string, timeout time.Duration) (*awstypes.Catalog, error) {
	stateConf := &retry.StateChangeConf{
		Pending:                   []string{},
		Target:                    []string{statusAvailable},
		Refresh:                   statusCatalog(ctx, conn, id),
		Timeout:                   timeout,
		NotFoundChecks:            20,
		ContinuousTargetOccurence: 2,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)
	if output, ok := outputRaw.(*awstypes.Catalog); ok {
		return output, err
	}

	return nil, err
}

func waitCatalogDeleted(ctx context.Context, conn *glue.Client, id string, timeout time.Duration) (*awstypes.Catalog, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{statusAvailable},
		Target:  []string{},
		Refresh: statusCatalog(ctx, conn, id),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)
	if output, ok := outputRaw.(*awstypes.Catalog); ok {
		return output, err
	}

	return nil, err
}

func statusCatalog(ctx context.Context, conn *glue.Client, id string) retry.StateRefreshFunc {
	return func(_ context.Context) (any, string, error) {
		output, err := findCatalogByID(ctx, conn, id)
		if retry.NotFound(err) {
			return nil, "", nil
		}
		if err != nil {
			return nil, "", err
		}

		return output, statusAvailable, nil
	}
}

func readCatalogResourceID(id string) (catalogId, name string, err error) {
	parts := strings.SplitN(id, ",", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format for ID (%[1]s), expected catalog_id,name", id)
	}
	return parts[0], parts[1], nil
}

func resolveCatalogID(catalogId, name string) string {
	if name == s3TablesCatalogName {
		return name
	}
	return catalogId
}

type resourceCatalogModel struct {
	framework.WithRegionModel
	AllowFullTableExternalDataAccess types.Bool                                                  `tfsdk:"allow_full_table_external_data_access"`
	ARN                              types.String                                                `tfsdk:"arn"`
	CatalogId                        types.String                                                `tfsdk:"catalog_id"`
	CatalogProperties                fwtypes.ListNestedObjectValueOf[catalogPropertiesModel]     `tfsdk:"catalog_properties"`
	Description                      types.String                                                `tfsdk:"description"`
	FederatedCatalog                 fwtypes.ListNestedObjectValueOf[federatedCatalogModel]      `tfsdk:"federated_catalog"`
	ID                               types.String                                                `tfsdk:"id"`
	Name                             types.String                                                `tfsdk:"name"`
	Tags                             tftags.Map                                                  `tfsdk:"tags"`
	TagsAll                          tftags.Map                                                  `tfsdk:"tags_all"`
	TargetRedshiftCatalog            fwtypes.ListNestedObjectValueOf[targetRedshiftCatalogModel] `tfsdk:"target_redshift_catalog"`
	Timeouts                         timeouts.Value                                              `tfsdk:"timeouts"`
}

type federatedCatalogModel struct {
	ConnectionName types.String `tfsdk:"connection_name"`
	Identifier     types.String `tfsdk:"identifier"`
}

type targetRedshiftCatalogModel struct {
	CatalogArn types.String `tfsdk:"catalog_arn"`
}

type catalogPropertiesModel struct {
	CustomProperties              fwtypes.MapValueOf[types.String]                                    `tfsdk:"custom_properties" autoflex:",omitempty"`
	DataLakeAccessProperties      fwtypes.ListNestedObjectValueOf[dataLakeAccessPropertiesModel]      `tfsdk:"data_lake_access_properties" autoflex:",omitempty"`
	IcebergOptimizationProperties fwtypes.ListNestedObjectValueOf[icebergOptimizationPropertiesModel] `tfsdk:"iceberg_optimization_properties" autoflex:",omitempty"`
}

type dataLakeAccessPropertiesModel struct {
	CatalogType      types.String `tfsdk:"catalog_type" autoflex:",omitempty"`
	DataLakeAccess   types.Bool   `tfsdk:"data_lake_access"`
	DataTransferRole types.String `tfsdk:"data_transfer_role" autoflex:",omitempty"`
	KmsKey           types.String `tfsdk:"kms_key" autoflex:",omitempty"`
}

type icebergOptimizationPropertiesModel struct {
	Compaction         fwtypes.MapValueOf[types.String] `tfsdk:"compaction" autoflex:",omitempty"`
	OrphanFileDeletion fwtypes.MapValueOf[types.String] `tfsdk:"orphan_file_deletion" autoflex:",omitempty"`
	Retention          fwtypes.MapValueOf[types.String] `tfsdk:"retention" autoflex:",omitempty"`
	RoleArn            types.String                     `tfsdk:"role_arn" autoflex:",omitempty"`
}
