// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package glue

import (
	"context"
	"time"

	"github.com/YakDriver/smarterr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	awstypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
	sweepfw "github.com/hashicorp/terraform-provider-aws/internal/sweep/framework"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_glue_catalog", name="Catalog")
// @Tags(identifierAttribute="arn")
func newCatalogResource(_ context.Context) (resource.ResourceWithConfigure, error) {
	r := &catalogResource{}

	r.SetDefaultCreateTimeout(30 * time.Minute)
	r.SetDefaultUpdateTimeout(30 * time.Minute)
	r.SetDefaultDeleteTimeout(30 * time.Minute)

	return r, nil
}

const (
	ResNameCatalog = "Catalog"
)

type catalogResource struct {
	framework.ResourceWithModel[catalogResourceModel]
	framework.WithTimeouts
	framework.WithImportByIdentity
}

func (r *catalogResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("catalog_properties").AtListIndex(0).AtName("data_lake_access_properties"),
			path.MatchRoot("federated_catalog"),
			path.MatchRoot("target_redshift_catalog"),
		),
	}
}

func (r *catalogResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"allow_full_table_external_data_access": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				CustomType: fwtypes.StringEnumType[awstypes.AllowFullTableExternalDataAccessEnum](),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrARN: schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrCatalogID: schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_time": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrDescription: schema.StringAttribute{
				Optional: true,
			},
			names.AttrID: framework.IDAttribute(),
			names.AttrName: schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrParameters: schema.MapAttribute{
				CustomType:  fwtypes.MapOfStringType,
				Optional:    true,
				ElementType: types.StringType,
			},
			"update_time": schema.StringAttribute{
				Computed: true,
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
		Blocks: map[string]schema.Block{
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
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.Map{
								mapplanmodifier.UseStateForUnknown(),
							},
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
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"data_lake_access": schema.BoolAttribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.Bool{
											boolplanmodifier.UseStateForUnknown(),
										},
									},
									"data_transfer_role": schema.StringAttribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									names.AttrKMSKey: schema.StringAttribute{
										Optional: true,
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"managed_workgroup_name": schema.StringAttribute{
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"managed_workgroup_status": schema.StringAttribute{
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"redshift_database_name": schema.StringAttribute{
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									names.AttrStatusMessage: schema.StringAttribute{
										Computed: true,
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						},
					},
				},
			},
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
						"connection_type": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
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
			"create_database_default_permissions": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[principalPermissionsModel](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						names.AttrPermissions: schema.ListAttribute{
							CustomType:  fwtypes.ListOfStringType,
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						names.AttrPrincipal: schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[dataLakePrincipalModel](ctx),
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"data_lake_principal_identifier": schema.StringAttribute{
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
			"create_table_default_permissions": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[principalPermissionsModel](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						names.AttrPermissions: schema.ListAttribute{
							CustomType:  fwtypes.ListOfStringType,
							Optional:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						names.AttrPrincipal: schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[dataLakePrincipalModel](ctx),
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"data_lake_principal_identifier": schema.StringAttribute{
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

func (r *catalogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	conn := r.Meta().GlueClient(ctx)

	var plan catalogResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
	if resp.Diagnostics.HasError() {
		return
	}

	catalogInput := expandCatalogInput(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	input := &glue.CreateCatalogInput{
		Name:         plan.Name.ValueStringPointer(),
		CatalogInput: catalogInput,
		Tags:         getTagsIn(ctx),
	}

	_, err := conn.CreateCatalog(ctx, input)
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.Name.ValueString())
		return
	}

	plan.ID = plan.Name

	out, err := waitCatalogReady(ctx, conn, plan.ID.ValueString(), r.CreateTimeout(ctx, plan.Timeouts))
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.ValueString())
		return
	}

	flattenCatalog(ctx, out, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, plan))
}

func (r *catalogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	conn := r.Meta().GlueClient(ctx)

	var state catalogResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := findCatalogByID(ctx, conn, state.ID.ValueString())
	if retry.NotFound(err) {
		smerr.AddOne(ctx, &resp.Diagnostics, fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.ValueString())
		return
	}

	flattenCatalog(ctx, out, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &state))
}

func (r *catalogResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Not applicable on create or destroy.
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var plan, state catalogResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Federated catalogs do not support UpdateCatalog — force replacement
	// when any catalog attribute changes.
	if !plan.FederatedCatalog.IsNull() || !state.FederatedCatalog.IsNull() {
		if !plan.Description.Equal(state.Description) ||
			!plan.Parameters.Equal(state.Parameters) ||
			!plan.CatalogProperties.Equal(state.CatalogProperties) ||
			!plan.FederatedCatalog.Equal(state.FederatedCatalog) ||
			!plan.AllowFullTableExternalDataAccess.Equal(state.AllowFullTableExternalDataAccess) ||
			!plan.CreateDatabaseDefaultPermissions.Equal(state.CreateDatabaseDefaultPermissions) ||
			!plan.CreateTableDefaultPermissions.Equal(state.CreateTableDefaultPermissions) {
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("federated_catalog"))
		}
	}
}

func (r *catalogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	conn := r.Meta().GlueClient(ctx)

	var plan, state catalogResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Description.Equal(state.Description) ||
		!plan.Parameters.Equal(state.Parameters) ||
		!plan.CatalogProperties.Equal(state.CatalogProperties) ||
		!plan.FederatedCatalog.Equal(state.FederatedCatalog) ||
		!plan.TargetRedshiftCatalog.Equal(state.TargetRedshiftCatalog) ||
		!plan.AllowFullTableExternalDataAccess.Equal(state.AllowFullTableExternalDataAccess) ||
		!plan.CreateDatabaseDefaultPermissions.Equal(state.CreateDatabaseDefaultPermissions) ||
		!plan.CreateTableDefaultPermissions.Equal(state.CreateTableDefaultPermissions) {

		catalogInput := expandCatalogInput(ctx, &plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		input := &glue.UpdateCatalogInput{
			CatalogId:    plan.ID.ValueStringPointer(),
			CatalogInput: catalogInput,
		}

		_, err := conn.UpdateCatalog(ctx, input)
		if err != nil {
			smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.ValueString())
			return
		}
	}

	out, err := findCatalogByID(ctx, conn, plan.ID.ValueString())
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.ValueString())
		return
	}

	flattenCatalog(ctx, out, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &plan))
}

func (r *catalogResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	conn := r.Meta().GlueClient(ctx)

	var state catalogResourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}

	// Retry on ConcurrentModificationException: Redshift-managed catalogs
	// (catalog_type = "aws:redshift") delegate to the Redshift workgroup,
	// which rejects deletes while another operation is still running.
	_, err := tfresource.RetryWhenIsA[any, *awstypes.ConcurrentModificationException](
		ctx,
		r.DeleteTimeout(ctx, state.Timeouts),
		func(ctx context.Context) (any, error) {
			return conn.DeleteCatalog(ctx, &glue.DeleteCatalogInput{
				CatalogId: state.ID.ValueStringPointer(),
			})
		},
	)

	if errs.IsA[*awstypes.EntityNotFoundException](err) {
		return
	}

	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.ValueString())
	}
}

func findCatalogByID(ctx context.Context, conn *glue.Client, id string) (*awstypes.Catalog, error) {
	input := &glue.GetCatalogInput{
		CatalogId: aws.String(id),
	}

	out, err := conn.GetCatalog(ctx, input)
	if err != nil {
		if errs.IsA[*awstypes.EntityNotFoundException](err) {
			return nil, smarterr.NewError(&retry.NotFoundError{
				LastError: err,
			})
		}
		return nil, smarterr.NewError(err)
	}

	if out == nil || out.Catalog == nil {
		return nil, smarterr.NewError(tfresource.NewEmptyResultError())
	}

	return out.Catalog, nil
}

// Managed workgroup status observed for catalogs created with
// DataLakeAccessProperties (RMS / "aws:redshift"). Redshift provisions the
// backing workgroup asynchronously and the status transitions through
// undocumented values (observed: CREATING, MODIFYING) before reaching
// AVAILABLE. Empty string means no managed workgroup (catalogs without
// data_lake_access_properties), which is also terminal.
const (
	managedWorkgroupStatusAvailable = "AVAILABLE"
)

// waitCatalogReady polls GetCatalog until the catalog's managed workgroup is
// AVAILABLE. For catalogs without data_lake_access_properties, no managed
// workgroup exists and the first read returns immediately. Any non-AVAILABLE
// non-empty status is treated as pending — AWS does not publish the enum, so
// we can't enumerate all transitional states and must accept anything that
// isn't the terminal value.
func waitCatalogReady(ctx context.Context, conn *glue.Client, id string, timeout time.Duration) (*awstypes.Catalog, error) {
	var catalog *awstypes.Catalog
	err := tfresource.Retry(ctx, timeout, func(ctx context.Context) *tfresource.RetryError {
		c, err := findCatalogByID(ctx, conn, id)
		if err != nil {
			return tfresource.NonRetryableError(err)
		}
		catalog = c

		if c.CatalogProperties == nil || c.CatalogProperties.DataLakeAccessProperties == nil {
			return nil
		}
		status := aws.ToString(c.CatalogProperties.DataLakeAccessProperties.ManagedWorkgroupStatus)
		if status == "" || status == managedWorkgroupStatusAvailable {
			return nil
		}
		return tfresource.RetryableError(smarterr.NewError(&retry.NotFoundError{
			Message: "managed workgroup status " + status,
		}))
	}, tfresource.WithPollInterval(10*time.Second))

	return catalog, err
}

// --- Expand helpers (model → AWS SDK types) ---

func expandCatalogInput(ctx context.Context, model *catalogResourceModel, diags *diag.Diagnostics) *awstypes.CatalogInput {
	input := &awstypes.CatalogInput{
		Description:                      model.Description.ValueStringPointer(),
		AllowFullTableExternalDataAccess: model.AllowFullTableExternalDataAccess.ValueEnum(),
	}

	if !model.Parameters.IsNull() && !model.Parameters.IsUnknown() {
		params := make(map[string]string)
		diags.Append(model.Parameters.ElementsAs(ctx, &params, false)...)
		if diags.HasError() {
			return nil
		}
		input.Parameters = params
	}

	if !model.CatalogProperties.IsNull() && !model.CatalogProperties.IsUnknown() {
		cpList, d := model.CatalogProperties.ToSlice(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil
		}
		if len(cpList) > 0 {
			input.CatalogProperties = expandCatalogProperties(ctx, cpList[0], diags)
			if diags.HasError() {
				return nil
			}
		}
	}

	if !model.FederatedCatalog.IsNull() && !model.FederatedCatalog.IsUnknown() {
		fcList, d := model.FederatedCatalog.ToSlice(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil
		}
		if len(fcList) > 0 {
			input.FederatedCatalog = expandFederatedCatalog(fcList[0])
		}
	}

	if !model.TargetRedshiftCatalog.IsNull() && !model.TargetRedshiftCatalog.IsUnknown() {
		trcList, d := model.TargetRedshiftCatalog.ToSlice(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil
		}
		if len(trcList) > 0 {
			input.TargetRedshiftCatalog = expandTargetRedshiftCatalog(trcList[0])
		}
	}

	if !model.CreateDatabaseDefaultPermissions.IsNull() && !model.CreateDatabaseDefaultPermissions.IsUnknown() {
		ppList, d := model.CreateDatabaseDefaultPermissions.ToSlice(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil
		}
		input.CreateDatabaseDefaultPermissions = expandPrincipalPermissionsList(ctx, ppList, diags)
		if diags.HasError() {
			return nil
		}
	} else {
		input.CreateDatabaseDefaultPermissions = []awstypes.PrincipalPermissions{}
	}

	if !model.CreateTableDefaultPermissions.IsNull() && !model.CreateTableDefaultPermissions.IsUnknown() {
		ppList, d := model.CreateTableDefaultPermissions.ToSlice(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil
		}
		input.CreateTableDefaultPermissions = expandPrincipalPermissionsList(ctx, ppList, diags)
		if diags.HasError() {
			return nil
		}
	} else {
		input.CreateTableDefaultPermissions = []awstypes.PrincipalPermissions{}
	}

	return input
}

func expandCatalogProperties(ctx context.Context, model *catalogPropertiesModel, diags *diag.Diagnostics) *awstypes.CatalogProperties {
	cp := &awstypes.CatalogProperties{}

	if !model.CustomProperties.IsNull() && !model.CustomProperties.IsUnknown() {
		props := make(map[string]string)
		diags.Append(model.CustomProperties.ElementsAs(ctx, &props, false)...)
		if diags.HasError() {
			return nil
		}
		cp.CustomProperties = props
	}

	if !model.DataLakeAccessProperties.IsNull() && !model.DataLakeAccessProperties.IsUnknown() {
		dlapList, d := model.DataLakeAccessProperties.ToSlice(ctx)
		diags.Append(d...)
		if diags.HasError() {
			return nil
		}
		if len(dlapList) > 0 {
			cp.DataLakeAccessProperties = expandDataLakeAccessProperties(dlapList[0])
		}
	}

	return cp
}

func expandDataLakeAccessProperties(model *dataLakeAccessPropertiesModel) *awstypes.DataLakeAccessProperties {
	dlap := &awstypes.DataLakeAccessProperties{}

	if !model.CatalogType.IsNull() && !model.CatalogType.IsUnknown() {
		dlap.CatalogType = model.CatalogType.ValueStringPointer()
	}
	if !model.DataLakeAccess.IsNull() && !model.DataLakeAccess.IsUnknown() {
		dlap.DataLakeAccess = model.DataLakeAccess.ValueBool()
	}
	if !model.DataTransferRole.IsNull() && !model.DataTransferRole.IsUnknown() {
		dlap.DataTransferRole = model.DataTransferRole.ValueStringPointer()
	}
	if !model.KmsKey.IsNull() && !model.KmsKey.IsUnknown() {
		dlap.KmsKey = model.KmsKey.ValueStringPointer()
	}

	return dlap
}

func expandFederatedCatalog(model *federatedCatalogModel) *awstypes.FederatedCatalog {
	fc := &awstypes.FederatedCatalog{}

	if !model.ConnectionName.IsNull() && !model.ConnectionName.IsUnknown() {
		fc.ConnectionName = model.ConnectionName.ValueStringPointer()
	}
	if !model.ConnectionType.IsNull() && !model.ConnectionType.IsUnknown() {
		fc.ConnectionType = model.ConnectionType.ValueStringPointer()
	}
	if !model.Identifier.IsNull() && !model.Identifier.IsUnknown() {
		fc.Identifier = model.Identifier.ValueStringPointer()
	}

	return fc
}

func expandTargetRedshiftCatalog(model *targetRedshiftCatalogModel) *awstypes.TargetRedshiftCatalog {
	return &awstypes.TargetRedshiftCatalog{
		CatalogArn: model.CatalogArn.ValueStringPointer(),
	}
}

func expandPrincipalPermissionsList(ctx context.Context, models []*principalPermissionsModel, diags *diag.Diagnostics) []awstypes.PrincipalPermissions {
	result := make([]awstypes.PrincipalPermissions, 0, len(models))

	for _, model := range models {
		pp := awstypes.PrincipalPermissions{}

		if !model.Permissions.IsNull() && !model.Permissions.IsUnknown() {
			var perms []string
			diags.Append(model.Permissions.ElementsAs(ctx, &perms, false)...)
			if diags.HasError() {
				return nil
			}
			for _, p := range perms {
				pp.Permissions = append(pp.Permissions, awstypes.Permission(p))
			}
		}

		if !model.Principal.IsNull() && !model.Principal.IsUnknown() {
			principalList, d := model.Principal.ToSlice(ctx)
			diags.Append(d...)
			if diags.HasError() {
				return nil
			}
			if len(principalList) > 0 {
				pp.Principal = &awstypes.DataLakePrincipal{
					DataLakePrincipalIdentifier: principalList[0].DataLakePrincipalIdentifier.ValueStringPointer(),
				}
			}
		}

		result = append(result, pp)
	}

	return result
}

// --- Flatten helpers (AWS SDK types → model) ---

func flattenCatalog(ctx context.Context, catalog *awstypes.Catalog, model *catalogResourceModel, diags *diag.Diagnostics) {
	model.Name = types.StringPointerValue(catalog.Name)
	model.CatalogID = types.StringPointerValue(catalog.CatalogId)
	model.ID = types.StringPointerValue(catalog.CatalogId)
	model.ARN = types.StringPointerValue(catalog.ResourceArn)
	model.Description = types.StringPointerValue(catalog.Description)
	model.AllowFullTableExternalDataAccess = fwtypes.StringEnumValue(catalog.AllowFullTableExternalDataAccess)

	if catalog.CreateTime != nil {
		model.CreateTime = types.StringValue(catalog.CreateTime.Format(time.RFC3339))
	} else {
		model.CreateTime = types.StringNull()
	}
	if catalog.UpdateTime != nil {
		model.UpdateTime = types.StringValue(catalog.UpdateTime.Format(time.RFC3339))
	} else {
		model.UpdateTime = types.StringNull()
	}

	if len(catalog.Parameters) > 0 {
		elems := make(map[string]attr.Value, len(catalog.Parameters))
		for k, v := range catalog.Parameters {
			elems[k] = types.StringValue(v)
		}
		params, d := fwtypes.NewMapValueOf[basetypes.StringValue](ctx, elems)
		diags.Append(d...)
		model.Parameters = params
	} else if model.Parameters.IsNull() {
		// Keep null if was null.
	} else {
		model.Parameters = fwtypes.NewMapValueOfNull[basetypes.StringValue](ctx)
	}

	if catalog.CatalogProperties != nil && !model.CatalogProperties.IsNull() {
		model.CatalogProperties = flattenCatalogPropertiesOutput(ctx, catalog.CatalogProperties, diags)
	}

	if catalog.FederatedCatalog != nil {
		fcModel := &federatedCatalogModel{
			ConnectionName: types.StringPointerValue(catalog.FederatedCatalog.ConnectionName),
			ConnectionType: types.StringPointerValue(catalog.FederatedCatalog.ConnectionType),
			Identifier:     types.StringPointerValue(catalog.FederatedCatalog.Identifier),
		}
		val, d := fwtypes.NewListNestedObjectValueOfPtr(ctx, fcModel)
		diags.Append(d...)
		model.FederatedCatalog = val
	} else {
		model.FederatedCatalog = fwtypes.NewListNestedObjectValueOfNull[federatedCatalogModel](ctx)
	}

	if catalog.TargetRedshiftCatalog != nil {
		trcModel := &targetRedshiftCatalogModel{
			CatalogArn: types.StringPointerValue(catalog.TargetRedshiftCatalog.CatalogArn),
		}
		val, d := fwtypes.NewListNestedObjectValueOfPtr(ctx, trcModel)
		diags.Append(d...)
		model.TargetRedshiftCatalog = val
	} else {
		model.TargetRedshiftCatalog = fwtypes.NewListNestedObjectValueOfNull[targetRedshiftCatalogModel](ctx)
	}

	if len(catalog.CreateDatabaseDefaultPermissions) > 0 {
		model.CreateDatabaseDefaultPermissions = flattenPrincipalPermissionsList(ctx, catalog.CreateDatabaseDefaultPermissions, diags)
	} else {
		model.CreateDatabaseDefaultPermissions = fwtypes.NewListNestedObjectValueOfNull[principalPermissionsModel](ctx)
	}

	if len(catalog.CreateTableDefaultPermissions) > 0 {
		model.CreateTableDefaultPermissions = flattenPrincipalPermissionsList(ctx, catalog.CreateTableDefaultPermissions, diags)
	} else {
		model.CreateTableDefaultPermissions = fwtypes.NewListNestedObjectValueOfNull[principalPermissionsModel](ctx)
	}
}

func flattenCatalogPropertiesOutput(ctx context.Context, cp *awstypes.CatalogPropertiesOutput, diags *diag.Diagnostics) fwtypes.ListNestedObjectValueOf[catalogPropertiesModel] {
	cpModel := &catalogPropertiesModel{}

	if len(cp.CustomProperties) > 0 {
		elems := make(map[string]attr.Value, len(cp.CustomProperties))
		for k, v := range cp.CustomProperties {
			elems[k] = types.StringValue(v)
		}
		props, d := fwtypes.NewMapValueOf[basetypes.StringValue](ctx, elems)
		diags.Append(d...)
		cpModel.CustomProperties = props
	} else {
		cpModel.CustomProperties = fwtypes.NewMapValueOfNull[basetypes.StringValue](ctx)
	}

	if cp.DataLakeAccessProperties != nil {
		dlapModel := &dataLakeAccessPropertiesModel{
			CatalogType:            types.StringPointerValue(cp.DataLakeAccessProperties.CatalogType),
			DataLakeAccess:         types.BoolValue(cp.DataLakeAccessProperties.DataLakeAccess),
			DataTransferRole:       types.StringPointerValue(cp.DataLakeAccessProperties.DataTransferRole),
			KmsKey:                 types.StringPointerValue(cp.DataLakeAccessProperties.KmsKey),
			ManagedWorkgroupName:   types.StringPointerValue(cp.DataLakeAccessProperties.ManagedWorkgroupName),
			ManagedWorkgroupStatus: types.StringPointerValue(cp.DataLakeAccessProperties.ManagedWorkgroupStatus),
			RedshiftDatabaseName:   types.StringPointerValue(cp.DataLakeAccessProperties.RedshiftDatabaseName),
			StatusMessage:          types.StringPointerValue(cp.DataLakeAccessProperties.StatusMessage),
		}
		val, d := fwtypes.NewListNestedObjectValueOfPtr(ctx, dlapModel)
		diags.Append(d...)
		cpModel.DataLakeAccessProperties = val
	} else {
		cpModel.DataLakeAccessProperties = fwtypes.NewListNestedObjectValueOfNull[dataLakeAccessPropertiesModel](ctx)
	}

	val, d := fwtypes.NewListNestedObjectValueOfPtr(ctx, cpModel)
	diags.Append(d...)
	return val
}

func flattenPrincipalPermissionsList(ctx context.Context, perms []awstypes.PrincipalPermissions, diags *diag.Diagnostics) fwtypes.ListNestedObjectValueOf[principalPermissionsModel] {
	models := make([]*principalPermissionsModel, 0, len(perms))

	for _, pp := range perms {
		model := &principalPermissionsModel{}

		if len(pp.Permissions) > 0 {
			elems := make([]attr.Value, len(pp.Permissions))
			for i, p := range pp.Permissions {
				elems[i] = types.StringValue(string(p))
			}
			val, d := fwtypes.NewListValueOf[basetypes.StringValue](ctx, elems)
			diags.Append(d...)
			model.Permissions = val
		} else {
			model.Permissions = fwtypes.NewListValueOfNull[basetypes.StringValue](ctx)
		}

		if pp.Principal != nil {
			principalModel := &dataLakePrincipalModel{
				DataLakePrincipalIdentifier: types.StringPointerValue(pp.Principal.DataLakePrincipalIdentifier),
			}
			val, d := fwtypes.NewListNestedObjectValueOfPtr(ctx, principalModel)
			diags.Append(d...)
			model.Principal = val
		} else {
			model.Principal = fwtypes.NewListNestedObjectValueOfNull[dataLakePrincipalModel](ctx)
		}

		models = append(models, model)
	}

	val, d := fwtypes.NewListNestedObjectValueOfSlice(ctx, models, nil)
	diags.Append(d...)
	return val
}

// --- Model structs ---

type catalogResourceModel struct {
	framework.WithRegionModel
	AllowFullTableExternalDataAccess fwtypes.StringEnum[awstypes.AllowFullTableExternalDataAccessEnum] `tfsdk:"allow_full_table_external_data_access"`
	ARN                              types.String                                                      `tfsdk:"arn"`
	CatalogID                        types.String                                                      `tfsdk:"catalog_id"`
	CatalogProperties                fwtypes.ListNestedObjectValueOf[catalogPropertiesModel]           `tfsdk:"catalog_properties"`
	CreateDatabaseDefaultPermissions fwtypes.ListNestedObjectValueOf[principalPermissionsModel]        `tfsdk:"create_database_default_permissions"`
	CreateTableDefaultPermissions    fwtypes.ListNestedObjectValueOf[principalPermissionsModel]        `tfsdk:"create_table_default_permissions"`
	CreateTime                       types.String                                                      `tfsdk:"create_time"`
	Description                      types.String                                                      `tfsdk:"description"`
	FederatedCatalog                 fwtypes.ListNestedObjectValueOf[federatedCatalogModel]            `tfsdk:"federated_catalog"`
	ID                               types.String                                                      `tfsdk:"id"`
	Name                             types.String                                                      `tfsdk:"name"`
	Parameters                       fwtypes.MapOfString                                               `tfsdk:"parameters"`
	Tags                             tftags.Map                                                        `tfsdk:"tags"`
	TagsAll                          tftags.Map                                                        `tfsdk:"tags_all"`
	TargetRedshiftCatalog            fwtypes.ListNestedObjectValueOf[targetRedshiftCatalogModel]       `tfsdk:"target_redshift_catalog"`
	Timeouts                         timeouts.Value                                                    `tfsdk:"timeouts"`
	UpdateTime                       types.String                                                      `tfsdk:"update_time"`
}

type catalogPropertiesModel struct {
	CustomProperties         fwtypes.MapOfString                                            `tfsdk:"custom_properties"`
	DataLakeAccessProperties fwtypes.ListNestedObjectValueOf[dataLakeAccessPropertiesModel] `tfsdk:"data_lake_access_properties"`
}

type dataLakeAccessPropertiesModel struct {
	CatalogType            types.String `tfsdk:"catalog_type"`
	DataLakeAccess         types.Bool   `tfsdk:"data_lake_access"`
	DataTransferRole       types.String `tfsdk:"data_transfer_role"`
	KmsKey                 types.String `tfsdk:"kms_key"`
	ManagedWorkgroupName   types.String `tfsdk:"managed_workgroup_name"`
	ManagedWorkgroupStatus types.String `tfsdk:"managed_workgroup_status"`
	RedshiftDatabaseName   types.String `tfsdk:"redshift_database_name"`
	StatusMessage          types.String `tfsdk:"status_message"`
}

type federatedCatalogModel struct {
	ConnectionName types.String `tfsdk:"connection_name"`
	ConnectionType types.String `tfsdk:"connection_type"`
	Identifier     types.String `tfsdk:"identifier"`
}

type targetRedshiftCatalogModel struct {
	CatalogArn types.String `tfsdk:"catalog_arn"`
}

type principalPermissionsModel struct {
	Permissions fwtypes.ListOfString                                    `tfsdk:"permissions"`
	Principal   fwtypes.ListNestedObjectValueOf[dataLakePrincipalModel] `tfsdk:"principal"`
}

type dataLakePrincipalModel struct {
	DataLakePrincipalIdentifier types.String `tfsdk:"data_lake_principal_identifier"`
}

// --- Sweep ---

func sweepCatalogs(ctx context.Context, client *conns.AWSClient) ([]sweep.Sweepable, error) {
	conn := client.GlueClient(ctx)
	var sweepResources []sweep.Sweepable

	input := &glue.GetCatalogsInput{
		Recursive: true,
	}

	for {
		output, err := conn.GetCatalogs(ctx, input)
		if err != nil {
			return nil, smarterr.NewError(err)
		}

		for _, v := range output.CatalogList {
			sweepResources = append(sweepResources, sweepfw.NewSweepResource(newCatalogResource, client,
				sweepfw.NewAttribute(names.AttrID, aws.ToString(v.CatalogId))),
			)
		}

		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return sweepResources, nil
}
