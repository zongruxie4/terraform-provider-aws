// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package redshift

import ( // nosemgrep:ci.semgrep.aws.multiple-service-imports
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	awstypes "github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/aws/aws-sdk-go-v2/service/redshiftserverless"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
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
	framework.ResourceWithModel[namespaceRegistrationResourceModel]
	framework.WithNoUpdate
	framework.WithImportByIdentity
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

	// Wait for the internal data share to be created
	if data.NamespaceType.ValueString() == "serverless" {
		// Get the namespace ID (UUID) for building the data share ARN
		// serverless_namespace_identifier can be either name or ID
		serverlessConn := r.Meta().RedshiftServerlessClient(ctx)
		namespaceIdentifier := data.ServerlessNamespaceIdentifier.ValueString()

		namespace, err := serverlessConn.GetNamespace(ctx, &redshiftserverless.GetNamespaceInput{
			NamespaceName: aws.String(namespaceIdentifier),
		})

		var namespaceID string
		if err != nil {
			// If GetNamespace fails, assume the identifier is already the namespace ID
			namespaceID = namespaceIdentifier
		} else {
			namespaceID = aws.ToString(namespace.Namespace.NamespaceId)
		}

		_, err = waitInternalDataShareCreated(ctx, conn,
			namespaceID,
			r.Meta().AccountID(ctx),
			r.Meta().Region(ctx),
			namespaceRegistrationInvalidClusterStateFaultTimeout)
		if err != nil {
			response.Diagnostics.AddError("waiting for Redshift internal data share creation", err.Error())
			return
		}
	} else if data.NamespaceType.ValueString() == "provisioned" {
		// Get the namespace ID from the cluster
		cluster, err := findClusterByID(ctx, conn, data.ProvisionedClusterIdentifier.ValueString())
		if err != nil {
			response.Diagnostics.AddError("reading Redshift Cluster for namespace ID", err.Error())
			return
		}

		// Extract namespace ID from ClusterNamespaceArn
		// Format: arn:aws:redshift:region:account:namespace:namespace-id
		namespaceArn := aws.ToString(cluster.ClusterNamespaceArn)
		parts := strings.Split(namespaceArn, ":")
		if len(parts) < 7 {
			response.Diagnostics.AddError("parsing cluster namespace ARN", fmt.Sprintf("invalid ARN format: %s", namespaceArn))
			return
		}
		namespaceID := parts[6]

		_, err = waitInternalDataShareCreated(ctx, conn,
			namespaceID,
			r.Meta().AccountID(ctx),
			r.Meta().Region(ctx),
			namespaceRegistrationInvalidClusterStateFaultTimeout)
		if err != nil {
			response.Diagnostics.AddError("waiting for Redshift internal data share creation", err.Error())
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	readResp := resource.ReadResponse{State: response.State}
	r.Read(ctx, resource.ReadRequest{State: response.State}, &readResp)
	response.Diagnostics.Append(readResp.Diagnostics...)
	response.State = readResp.State
}

func (r *namespaceRegistrationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data namespaceRegistrationResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	// For serverless namespaces, if the identifier is a UUID (namespace_id),
	// we cannot verify the registration status because GetNamespace requires namespace_name.
	// In this case, we trust the state and skip verification.
	if data.NamespaceType.ValueString() == "serverless" {
		identifier := data.ServerlessNamespaceIdentifier.ValueString()
		// Check if it looks like a UUID (8-4-4-4-12 format)
		if len(identifier) == 36 && identifier[8] == '-' && identifier[13] == '-' {
			// Assume it's a UUID, skip verification
			response.Diagnostics.Append(response.State.Set(ctx, &data)...)
			return
		}
	}

	conn := r.Meta().RedshiftClient(ctx)
	serverlessConn := r.Meta().RedshiftServerlessClient(ctx)

	// LakehouseRegistrationStatus is not immediately populated by AWS after registration,
	// particularly for provisioned clusters. We verify the namespace/cluster exists but cannot
	// reliably detect if the registration has been removed outside Terraform.
	err := findNamespaceRegistrationByID(ctx, conn, serverlessConn,
		data.ConsumerIdentifier.ValueString(),
		data.NamespaceType.ValueString(),
		data.ServerlessNamespaceIdentifier.ValueString(),
		data.ServerlessWorkgroupIdentifier.ValueString(),
		data.ProvisionedClusterIdentifier.ValueString())

	if retry.NotFound(err) {
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

type namespaceRegistrationResourceModel struct {
	framework.WithRegionModel
	ConsumerIdentifier            types.String `tfsdk:"consumer_identifier"`
	ID                            types.String `tfsdk:"id"`
	NamespaceType                 types.String `tfsdk:"namespace_type"`
	ProvisionedClusterIdentifier  types.String `tfsdk:"provisioned_cluster_identifier"`
	ServerlessNamespaceIdentifier types.String `tfsdk:"serverless_namespace_identifier"`
	ServerlessWorkgroupIdentifier types.String `tfsdk:"serverless_workgroup_identifier"`
}
