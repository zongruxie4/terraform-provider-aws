// Code generated by internal/generate/servicepackage/main.go; DO NOT EDIT.

package ecs

import (
	"context"
	"unique"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	inttypes "github.com/hashicorp/terraform-provider-aws/internal/types"
	"github.com/hashicorp/terraform-provider-aws/internal/vcr"
	"github.com/hashicorp/terraform-provider-aws/names"
)

type servicePackage struct{}

func (p *servicePackage) FrameworkDataSources(ctx context.Context) []*inttypes.ServicePackageFrameworkDataSource {
	return []*inttypes.ServicePackageFrameworkDataSource{
		{
			Factory:  newClustersDataSource,
			TypeName: "aws_ecs_clusters",
			Name:     "Clusters",
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
	}
}

func (p *servicePackage) FrameworkResources(ctx context.Context) []*inttypes.ServicePackageFrameworkResource {
	return []*inttypes.ServicePackageFrameworkResource{}
}

func (p *servicePackage) SDKDataSources(ctx context.Context) []*inttypes.ServicePackageSDKDataSource {
	return []*inttypes.ServicePackageSDKDataSource{
		{
			Factory:  dataSourceCluster,
			TypeName: "aws_ecs_cluster",
			Name:     "Cluster",
			Tags:     unique.Make(inttypes.ServicePackageResourceTags{}),
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  dataSourceContainerDefinition,
			TypeName: "aws_ecs_container_definition",
			Name:     "Container Definition",
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  dataSourceService,
			TypeName: "aws_ecs_service",
			Name:     "Service",
			Tags:     unique.Make(inttypes.ServicePackageResourceTags{}),
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  dataSourceTaskDefinition,
			TypeName: "aws_ecs_task_definition",
			Name:     "Task Definition",
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  dataSourceTaskExecution,
			TypeName: "aws_ecs_task_execution",
			Name:     "Task Execution",
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
	}
}

func (p *servicePackage) SDKResources(ctx context.Context) []*inttypes.ServicePackageSDKResource {
	return []*inttypes.ServicePackageSDKResource{
		{
			Factory:  resourceAccountSettingDefault,
			TypeName: "aws_ecs_account_setting_default",
			Name:     "Account Setting Default",
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  resourceCapacityProvider,
			TypeName: "aws_ecs_capacity_provider",
			Name:     "Capacity Provider",
			Tags: unique.Make(inttypes.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			}),
			Region: unique.Make(inttypes.ResourceRegionDefault()),
			Identity: inttypes.RegionalARNIdentity(
				inttypes.WithIdentityDuplicateAttrs(names.AttrID),
				inttypes.WithV6_0SDKv2Fix(),
			),
			Import: inttypes.SDKv2Import{
				WrappedImport: true,
			},
		},
		{
			Factory:  resourceCluster,
			TypeName: "aws_ecs_cluster",
			Name:     "Cluster",
			Tags: unique.Make(inttypes.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			}),
			Region: unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  resourceClusterCapacityProviders,
			TypeName: "aws_ecs_cluster_capacity_providers",
			Name:     "Cluster Capacity Providers",
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  resourceService,
			TypeName: "aws_ecs_service",
			Name:     "Service",
			Tags: unique.Make(inttypes.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			}),
			Region: unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  resourceTag,
			TypeName: "aws_ecs_tag",
			Name:     "ECS Resource Tag",
			Region:   unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  resourceTaskDefinition,
			TypeName: "aws_ecs_task_definition",
			Name:     "Task Definition",
			Tags: unique.Make(inttypes.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			}),
			Region: unique.Make(inttypes.ResourceRegionDefault()),
		},
		{
			Factory:  resourceTaskSet,
			TypeName: "aws_ecs_task_set",
			Name:     "Task Set",
			Tags: unique.Make(inttypes.ServicePackageResourceTags{
				IdentifierAttribute: names.AttrARN,
			}),
			Region: unique.Make(inttypes.ResourceRegionDefault()),
		},
	}
}

func (p *servicePackage) ServicePackageName() string {
	return names.ECS
}

// NewClient returns a new AWS SDK for Go v2 client for this service package's AWS API.
func (p *servicePackage) NewClient(ctx context.Context, config map[string]any) (*ecs.Client, error) {
	cfg := *(config["aws_sdkv2_config"].(*aws.Config))
	optFns := []func(*ecs.Options){
		ecs.WithEndpointResolverV2(newEndpointResolverV2()),
		withBaseEndpoint(config[names.AttrEndpoint].(string)),
		func(o *ecs.Options) {
			if region := config[names.AttrRegion].(string); o.Region != region {
				tflog.Info(ctx, "overriding provider-configured AWS API region", map[string]any{
					"service":         p.ServicePackageName(),
					"original_region": o.Region,
					"override_region": region,
				})
				o.Region = region
			}
		},
		func(o *ecs.Options) {
			if inContext, ok := conns.FromContext(ctx); ok && inContext.VCREnabled() {
				tflog.Info(ctx, "overriding retry behavior to immediately return VCR errors")
				o.Retryer = conns.AddIsErrorRetryables(cfg.Retryer().(aws.RetryerV2), retry.IsErrorRetryableFunc(vcr.InteractionNotFoundRetryableFunc))
			}
		},
		withExtraOptions(ctx, p, config),
	}

	return ecs.NewFromConfig(cfg, optFns...), nil
}

// withExtraOptions returns a functional option that allows this service package to specify extra API client options.
// This option is always called after any generated options.
func withExtraOptions(ctx context.Context, sp conns.ServicePackage, config map[string]any) func(*ecs.Options) {
	if v, ok := sp.(interface {
		withExtraOptions(context.Context, map[string]any) []func(*ecs.Options)
	}); ok {
		optFns := v.withExtraOptions(ctx, config)

		return func(o *ecs.Options) {
			for _, optFn := range optFns {
				optFn(o)
			}
		}
	}

	return func(*ecs.Options) {}
}

func ServicePackage(ctx context.Context) conns.ServicePackage {
	return &servicePackage{}
}
