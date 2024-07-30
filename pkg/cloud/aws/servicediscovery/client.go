package servicediscovery

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Client
type Client interface {
	CreateHttpNamespace(ctx context.Context, params *servicediscovery.CreateHttpNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateHttpNamespaceOutput, error)
	CreatePrivateDnsNamespace(
		ctx context.Context,
		params *servicediscovery.CreatePrivateDnsNamespaceInput,
		optFns ...func(*servicediscovery.Options),
	) (*servicediscovery.CreatePrivateDnsNamespaceOutput, error)
	CreatePublicDnsNamespace(
		ctx context.Context,
		params *servicediscovery.CreatePublicDnsNamespaceInput,
		optFns ...func(*servicediscovery.Options),
	) (*servicediscovery.CreatePublicDnsNamespaceOutput, error)
	CreateService(ctx context.Context, params *servicediscovery.CreateServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.CreateServiceOutput, error)
	DeleteNamespace(ctx context.Context, params *servicediscovery.DeleteNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteNamespaceOutput, error)
	DeleteService(ctx context.Context, params *servicediscovery.DeleteServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeleteServiceOutput, error)
	DeregisterInstance(ctx context.Context, params *servicediscovery.DeregisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DeregisterInstanceOutput, error)
	DiscoverInstances(ctx context.Context, params *servicediscovery.DiscoverInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.DiscoverInstancesOutput, error)
	GetInstance(ctx context.Context, params *servicediscovery.GetInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetInstanceOutput, error)
	GetInstancesHealthStatus(
		ctx context.Context,
		params *servicediscovery.GetInstancesHealthStatusInput,
		optFns ...func(*servicediscovery.Options),
	) (*servicediscovery.GetInstancesHealthStatusOutput, error)
	GetNamespace(ctx context.Context, params *servicediscovery.GetNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetNamespaceOutput, error)
	GetOperation(ctx context.Context, params *servicediscovery.GetOperationInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetOperationOutput, error)
	GetService(ctx context.Context, params *servicediscovery.GetServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.GetServiceOutput, error)
	ListInstances(ctx context.Context, params *servicediscovery.ListInstancesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListInstancesOutput, error)
	ListNamespaces(ctx context.Context, params *servicediscovery.ListNamespacesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListNamespacesOutput, error)
	ListOperations(ctx context.Context, params *servicediscovery.ListOperationsInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListOperationsOutput, error)
	ListServices(ctx context.Context, params *servicediscovery.ListServicesInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListServicesOutput, error)
	ListTagsForResource(ctx context.Context, params *servicediscovery.ListTagsForResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.ListTagsForResourceOutput, error)
	RegisterInstance(ctx context.Context, params *servicediscovery.RegisterInstanceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.RegisterInstanceOutput, error)
	TagResource(ctx context.Context, params *servicediscovery.TagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.TagResourceOutput, error)
	UntagResource(ctx context.Context, params *servicediscovery.UntagResourceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.UntagResourceOutput, error)
	UpdateHttpNamespace(ctx context.Context, params *servicediscovery.UpdateHttpNamespaceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.UpdateHttpNamespaceOutput, error)
	UpdateInstanceCustomHealthStatus(
		ctx context.Context,
		params *servicediscovery.UpdateInstanceCustomHealthStatusInput,
		optFns ...func(*servicediscovery.Options),
	) (*servicediscovery.UpdateInstanceCustomHealthStatusOutput, error)
	UpdatePrivateDnsNamespace(
		ctx context.Context,
		params *servicediscovery.UpdatePrivateDnsNamespaceInput,
		optFns ...func(*servicediscovery.Options),
	) (*servicediscovery.UpdatePrivateDnsNamespaceOutput, error)
	UpdatePublicDnsNamespace(
		ctx context.Context,
		params *servicediscovery.UpdatePublicDnsNamespaceInput,
		optFns ...func(*servicediscovery.Options),
	) (*servicediscovery.UpdatePublicDnsNamespaceOutput, error)
	UpdateService(ctx context.Context, params *servicediscovery.UpdateServiceInput, optFns ...func(*servicediscovery.Options)) (*servicediscovery.UpdateServiceOutput, error)
}

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (Client, error) {
	clientCfg := &ClientConfig{}
	gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "servicediscovery", name)

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := servicediscovery.NewFromConfig(awsConfig)

	gosoAws.LogNewClientCreated(ctx, logger, "servicediscovery", name, clientCfg.Settings.ClientSettings)

	return client, nil
}
