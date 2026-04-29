// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package redshift

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/redshift"
	awstypes "github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

const (
	namespaceRegistrationInvalidClusterStateFaultTimeout = 15 * time.Minute
)

// @FrameworkResource("aws_redshift_namespace_registration", name="Namespace Registration")
// @IdentityAttribute("id")
// @Testing(hasNoPreExistingResource=true)
func newNamespaceRegistrationResource(context.Context) (resource.ResourceWithConfigure, error) {
	return &namespaceRegistrationResource{}, nil
}

type namespaceRegistrationResource struct {
	framework.ResourceWithConfigure
	framework.WithNoUpdate
	framework.WithImportByIdentity
	framework.ResourceWithModel[namespaceRegistrationResourceModel]
}

func (r *namespaceRegistrationResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "aws_redshift_namespace_registration"
}

func (r *namespaceRegistrationResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"consumer_identifier": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			names.AttrID: framework.IDAttribute(),
			"namespace_type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"serverless_namespace_identifier": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"serverless_workgroup_identifier": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provisioned_cluster_identifier": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *namespaceRegistrationResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data namespaceRegistrationResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().RedshiftClient(ctx)

	input := &redshift.RegisterNamespaceInput{
		ConsumerIdentifiers: []string{data.ConsumerIdentifier.ValueString()},
	}

	if data.NamespaceType.ValueString() == "serverless" {
		input.NamespaceIdentifier = &awstypes.NamespaceIdentifierUnionMemberServerlessIdentifier{
			Value: awstypes.ServerlessIdentifier{
				NamespaceIdentifier: fwflex.StringFromFramework(ctx, data.ServerlessNamespaceIdentifier),
				WorkgroupIdentifier: fwflex.StringFromFramework(ctx, data.ServerlessWorkgroupIdentifier),
			},
		}
		data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s",
			data.ConsumerIdentifier.ValueString(),
			data.ServerlessNamespaceIdentifier.ValueString(),
			data.ServerlessWorkgroupIdentifier.ValueString()))
	} else {
		input.NamespaceIdentifier = &awstypes.NamespaceIdentifierUnionMemberProvisionedIdentifier{
			Value: awstypes.ProvisionedIdentifier{
				ClusterIdentifier: fwflex.StringFromFramework(ctx, data.ProvisionedClusterIdentifier),
			},
		}
		data.ID = types.StringValue(fmt.Sprintf("%s/%s",
			data.ConsumerIdentifier.ValueString(),
			data.ProvisionedClusterIdentifier.ValueString()))
	}

	_, err := tfresource.RetryWhenIsA[any, *awstypes.InvalidClusterStateFault](ctx, namespaceRegistrationInvalidClusterStateFaultTimeout,
		func(ctx context.Context) (any, error) {
			return conn.RegisterNamespace(ctx, input)
		})
	if err != nil {
		response.Diagnostics.AddError("creating Redshift Namespace Registration", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *namespaceRegistrationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data namespaceRegistrationResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().RedshiftClient(ctx)
	serverlessConn := r.Meta().RedshiftServerlessClient(ctx)

	_, err := findNamespaceRegistrationByID(ctx, conn, serverlessConn,
		data.ConsumerIdentifier.ValueString(),
		data.NamespaceType.ValueString(),
		data.ServerlessNamespaceIdentifier.ValueString(),
		data.ServerlessWorkgroupIdentifier.ValueString(),
		data.ProvisionedClusterIdentifier.ValueString())

	if tfresource.NotFound(err) {
		response.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		response.Diagnostics.AddError("reading Redshift Namespace Registration", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *namespaceRegistrationResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data namespaceRegistrationResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().RedshiftClient(ctx)

	input := &redshift.DeregisterNamespaceInput{
		ConsumerIdentifiers: []string{data.ConsumerIdentifier.ValueString()},
	}

	if data.NamespaceType.ValueString() == "serverless" {
		input.NamespaceIdentifier = &awstypes.NamespaceIdentifierUnionMemberServerlessIdentifier{
			Value: awstypes.ServerlessIdentifier{
				NamespaceIdentifier: fwflex.StringFromFramework(ctx, data.ServerlessNamespaceIdentifier),
				WorkgroupIdentifier: fwflex.StringFromFramework(ctx, data.ServerlessWorkgroupIdentifier),
			},
		}
	} else {
		input.NamespaceIdentifier = &awstypes.NamespaceIdentifierUnionMemberProvisionedIdentifier{
			Value: awstypes.ProvisionedIdentifier{
				ClusterIdentifier: fwflex.StringFromFramework(ctx, data.ProvisionedClusterIdentifier),
			},
		}
	}

	_, err := tfresource.RetryWhenIsA[any, *awstypes.InvalidClusterStateFault](ctx, namespaceRegistrationInvalidClusterStateFaultTimeout,
		func(ctx context.Context) (any, error) {
			return conn.DeregisterNamespace(ctx, input)
		})
	if err != nil {
		if errs.IsA[*awstypes.ClusterNotFoundFault](err) || errs.IsA[*awstypes.InvalidNamespaceFault](err) {
			return
		}
		response.Diagnostics.AddError("deleting Redshift Namespace Registration", err.Error())
		return
	}
}

func (r *namespaceRegistrationResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(names.AttrID), request, response)
}

type namespaceRegistrationResourceModel struct {
	framework.WithRegionModel
	ConsumerIdentifier            types.String `tfsdk:"consumer_identifier"`
	ID                            types.String `tfsdk:"id"`
	NamespaceType                 types.String `tfsdk:"namespace_type"`
	ProvisionedClusterIdentifier  types.String `tfsdk:"provisioned_cluster_identifier"`
	ServerlessNamespaceIdentifier types.String `tfsdk:"serverless_namespace_identifier"`
	ServerlessWorkgroupIdentifier types.String `tfsdk:"serverless_workgroup_identifier"`
}
