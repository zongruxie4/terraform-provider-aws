// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package redshift

import (
	"context"
	"log"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	awstypes "github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_redshift_endpoint_access", name="Endpoint Access")
func resourceEndpointAccess() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceEndpointAccessCreate,
		ReadWithoutTimeout:   resourceEndpointAccessRead,
		UpdateWithoutTimeout: resourceEndpointAccessUpdate,
		DeleteWithoutTimeout: resourceEndpointAccessDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			names.AttrAddress: {
				Type:     schema.TypeString,
				Computed: true,
			},
			names.AttrClusterIdentifier: {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"endpoint_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 30),
					validation.StringMatch(regexache.MustCompile(`^[0-9a-z-]+$`), "must contain only lowercase alphanumeric characters and hyphens"),
				),
			},
			names.AttrPort: {
				Type:     schema.TypeInt,
				Computed: true,
			},
			names.AttrResourceOwner: {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
				Computed: true,
			},
			"subnet_group_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"vpc_endpoint": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_interface": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									names.AttrAvailabilityZone: {
										Type:     schema.TypeString,
										Computed: true,
									},
									names.AttrNetworkInterfaceID: {
										Type:     schema.TypeString,
										Computed: true,
									},
									"private_ip_address": {
										Type:     schema.TypeString,
										Computed: true,
									},
									names.AttrSubnetID: {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						names.AttrVPCEndpointID: {
							Type:     schema.TypeString,
							Computed: true,
						},
						names.AttrVPCID: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			names.AttrVPCSecurityGroupIDs: {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceEndpointAccessCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RedshiftClient(ctx)

	createOpts := redshift.CreateEndpointAccessInput{
		EndpointName:    aws.String(d.Get("endpoint_name").(string)),
		SubnetGroupName: aws.String(d.Get("subnet_group_name").(string)),
	}

	if v, ok := d.GetOk(names.AttrVPCSecurityGroupIDs); ok && v.(*schema.Set).Len() > 0 {
		createOpts.VpcSecurityGroupIds = flex.ExpandStringValueSet(v.(*schema.Set))
	}

	if v, ok := d.GetOk(names.AttrClusterIdentifier); ok {
		createOpts.ClusterIdentifier = aws.String(v.(string))
	}

	if v, ok := d.GetOk(names.AttrResourceOwner); ok {
		createOpts.ResourceOwner = aws.String(v.(string))
	}

	_, err := conn.CreateEndpointAccess(ctx, &createOpts)
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating Redshift endpoint access: %s", err)
	}

	d.SetId(aws.ToString(createOpts.EndpointName))
	log.Printf("[INFO] Redshift endpoint access ID: %s", d.Id())

	if _, err := waitEndpointAccessActive(ctx, conn, d.Id()); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for Redshift Endpoint Access (%s) to be active: %s", d.Id(), err)
	}

	return append(diags, resourceEndpointAccessRead(ctx, d, meta)...)
}

func resourceEndpointAccessRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RedshiftClient(ctx)

	endpoint, err := findEndpointAccessByName(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] Redshift endpoint access (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading Redshift endpoint access (%s): %s", d.Id(), err)
	}

	d.Set("endpoint_name", endpoint.EndpointName)
	d.Set("subnet_group_name", endpoint.SubnetGroupName)
	d.Set(names.AttrVPCSecurityGroupIDs, vpcSgsIdsToSlice(endpoint.VpcSecurityGroups))
	d.Set(names.AttrResourceOwner, endpoint.ResourceOwner)
	d.Set(names.AttrClusterIdentifier, endpoint.ClusterIdentifier)
	d.Set(names.AttrPort, endpoint.Port)
	d.Set(names.AttrAddress, endpoint.Address)

	if err := d.Set("vpc_endpoint", flattenVPCEndpoint(endpoint.VpcEndpoint)); err != nil {
		return sdkdiag.AppendErrorf(diags, "setting vpc_endpoint: %s", err)
	}

	return diags
}

func resourceEndpointAccessUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RedshiftClient(ctx)

	if d.HasChanges(names.AttrVPCSecurityGroupIDs) {
		_, n := d.GetChange(names.AttrVPCSecurityGroupIDs)
		if n == nil {
			n = new(schema.Set)
		}
		ns := n.(*schema.Set)

		var sIds []string
		for _, s := range ns.List() {
			sIds = append(sIds, s.(string))
		}

		_, err := conn.ModifyEndpointAccess(ctx, &redshift.ModifyEndpointAccessInput{
			EndpointName:        aws.String(d.Id()),
			VpcSecurityGroupIds: sIds,
		})

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating Redshift endpoint access (%s): %s", d.Id(), err)
		}

		if _, err := waitEndpointAccessActive(ctx, conn, d.Id()); err != nil {
			return sdkdiag.AppendErrorf(diags, "waiting for Redshift Endpoint Access (%s) to be active: %s", d.Id(), err)
		}
	}

	return append(diags, resourceEndpointAccessRead(ctx, d, meta)...)
}

func resourceEndpointAccessDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).RedshiftClient(ctx)

	_, err := conn.DeleteEndpointAccess(ctx, &redshift.DeleteEndpointAccessInput{
		EndpointName: aws.String(d.Id()),
	})

	if err != nil {
		if errs.IsA[*awstypes.EndpointNotFoundFault](err) {
			return diags
		}
		return sdkdiag.AppendErrorf(diags, "deleting Redshift Endpoint Access (%s): %s", d.Id(), err)
	}

	if _, err := waitEndpointAccessDeleted(ctx, conn, d.Id()); err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting Redshift Endpoint Access (%s): waiting for completion: %s", d.Id(), err)
	}

	return diags
}

func vpcSgsIdsToSlice(vpsSgsIds []awstypes.VpcSecurityGroupMembership) []string {
	VpcSgsSlice := make([]string, 0, len(vpsSgsIds))
	for _, s := range vpsSgsIds {
		VpcSgsSlice = append(VpcSgsSlice, aws.ToString(s.VpcSecurityGroupId))
	}
	return VpcSgsSlice
}

func flattenVPCEndpoint(apiObject *awstypes.VpcEndpoint) []any {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]any{}

	if v := apiObject.NetworkInterfaces; v != nil {
		tfMap["network_interface"] = flattenNetworkInterfaces(v)
	}

	if v := apiObject.VpcEndpointId; v != nil {
		tfMap[names.AttrVPCEndpointID] = aws.ToString(v)
	}

	if v := apiObject.VpcId; v != nil {
		tfMap[names.AttrVPCID] = aws.ToString(v)
	}

	return []any{tfMap}
}

func flattenNetworkInterface(apiObject awstypes.NetworkInterface) map[string]any {
	tfMap := map[string]any{}

	if v := apiObject.AvailabilityZone; v != nil {
		tfMap[names.AttrAvailabilityZone] = aws.ToString(v)
	}

	if v := apiObject.NetworkInterfaceId; v != nil {
		tfMap[names.AttrNetworkInterfaceID] = aws.ToString(v)
	}

	if v := apiObject.PrivateIpAddress; v != nil {
		tfMap["private_ip_address"] = aws.ToString(v)
	}

	if v := apiObject.SubnetId; v != nil {
		tfMap[names.AttrSubnetID] = aws.ToString(v)
	}

	return tfMap
}

func flattenNetworkInterfaces(apiObjects []awstypes.NetworkInterface) []any {
	if len(apiObjects) == 0 {
		return nil
	}

	var tfList []any

	for _, apiObject := range apiObjects {
		tfList = append(tfList, flattenNetworkInterface(apiObject))
	}

	return tfList
}
