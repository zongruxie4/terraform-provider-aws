// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package glue

import (
	"context"

	awstypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkDataSource("aws_glue_catalog", name="Catalog")
// @Tags(identifierAttribute="arn")
func newCatalogDataSource(context.Context) (datasource.DataSourceWithConfigure, error) {
	return &catalogDataSource{}, nil
}

type catalogDataSource struct {
	framework.DataSourceWithModel[catalogDataSourceModel]
}

func (d *catalogDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"allow_full_table_external_data_access": schema.StringAttribute{
				Computed:   true,
				CustomType: fwtypes.StringEnumType[awstypes.AllowFullTableExternalDataAccessEnum](),
			},
			names.AttrARN: schema.StringAttribute{
				Computed: true,
			},
			names.AttrCatalogID: schema.StringAttribute{
				Computed: true,
			},
			names.AttrCreateTime: schema.StringAttribute{
				CustomType: timetypes.RFC3339Type{},
				Computed:   true,
			},
			names.AttrDescription: schema.StringAttribute{
				Computed: true,
			},
			names.AttrID: framework.IDAttribute(),
			names.AttrName: schema.StringAttribute{
				Required: true,
			},
			names.AttrParameters: schema.MapAttribute{
				CustomType:  fwtypes.MapOfStringType,
				Computed:    true,
				ElementType: types.StringType,
			},
			names.AttrTags: tftags.TagsAttributeComputedOnly(),
			"update_time": schema.StringAttribute{
				CustomType: timetypes.RFC3339Type{},
				Computed:   true,
			},
		},
		Blocks: map[string]schema.Block{
			"catalog_properties": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[catalogPropertiesModel](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"custom_properties": schema.MapAttribute{
							CustomType:  fwtypes.MapOfStringType,
							Computed:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						"data_lake_access_properties": schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[dataLakeAccessPropertiesModel](ctx),
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"catalog_type": schema.StringAttribute{
										Computed: true,
									},
									"data_lake_access": schema.BoolAttribute{
										Computed: true,
									},
									"data_transfer_role": schema.StringAttribute{
										Computed: true,
									},
									names.AttrKMSKey: schema.StringAttribute{
										Computed: true,
									},
									"managed_workgroup_name": schema.StringAttribute{
										Computed: true,
									},
									"managed_workgroup_status": schema.StringAttribute{
										Computed: true,
									},
									"redshift_database_name": schema.StringAttribute{
										Computed: true,
									},
									names.AttrStatusMessage: schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"federated_catalog": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[federatedCatalogModel](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"connection_name": schema.StringAttribute{
							Computed: true,
						},
						"connection_type": schema.StringAttribute{
							Computed: true,
						},
						names.AttrIdentifier: schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"target_redshift_catalog": schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[targetRedshiftCatalogModel](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"catalog_arn": schema.StringAttribute{
							Computed: true,
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
							Computed:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						names.AttrPrincipal: schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[dataLakePrincipalModel](ctx),
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"data_lake_principal_identifier": schema.StringAttribute{
										Computed: true,
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
							Computed:    true,
							ElementType: types.StringType,
						},
					},
					Blocks: map[string]schema.Block{
						names.AttrPrincipal: schema.ListNestedBlock{
							CustomType: fwtypes.NewListNestedObjectTypeOf[dataLakePrincipalModel](ctx),
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"data_lake_principal_identifier": schema.StringAttribute{
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *catalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	conn := d.Meta().GlueClient(ctx)

	var data catalogDataSourceModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Config.Get(ctx, &data))
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := findCatalogByName(ctx, conn, data.Name.ValueString())
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, data.Name.ValueString())
		return
	}

	smerr.AddEnrich(ctx, &resp.Diagnostics, fwflex.Flatten(ctx, out, &data))
	if resp.Diagnostics.HasError() {
		return
	}
	data.ARN = types.StringPointerValue(out.ResourceArn)
	data.ID = types.StringPointerValue(out.CatalogId)

	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &data))
}

type catalogDataSourceModel struct {
	framework.WithRegionModel
	AllowFullTableExternalDataAccess fwtypes.StringEnum[awstypes.AllowFullTableExternalDataAccessEnum] `tfsdk:"allow_full_table_external_data_access"`
	ARN                              types.String                                                      `tfsdk:"arn" autoflex:"-"`
	CatalogID                        types.String                                                      `tfsdk:"catalog_id"`
	CatalogProperties                fwtypes.ListNestedObjectValueOf[catalogPropertiesModel]           `tfsdk:"catalog_properties"`
	CreateDatabaseDefaultPermissions fwtypes.ListNestedObjectValueOf[principalPermissionsModel]        `tfsdk:"create_database_default_permissions"`
	CreateTableDefaultPermissions    fwtypes.ListNestedObjectValueOf[principalPermissionsModel]        `tfsdk:"create_table_default_permissions"`
	CreateTime                       timetypes.RFC3339                                                 `tfsdk:"create_time"`
	Description                      types.String                                                      `tfsdk:"description"`
	FederatedCatalog                 fwtypes.ListNestedObjectValueOf[federatedCatalogModel]            `tfsdk:"federated_catalog"`
	ID                               types.String                                                      `tfsdk:"id" autoflex:"-"`
	Name                             types.String                                                      `tfsdk:"name"`
	Parameters                       fwtypes.MapOfString                                               `tfsdk:"parameters"`
	Tags                             tftags.Map                                                        `tfsdk:"tags"`
	TargetRedshiftCatalog            fwtypes.ListNestedObjectValueOf[targetRedshiftCatalogModel]       `tfsdk:"target_redshift_catalog"`
	UpdateTime                       timetypes.RFC3339                                                 `tfsdk:"update_time"`
}
