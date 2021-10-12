package instancemetadataservice

import (
	"context"
	"fmt"

	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Client
type Client interface {
	GetMetadata(ctx context.Context, input *imds.GetMetadataInput, optFns ...func(options *imds.Options)) (*imds.GetMetadataOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
}

type ClientConfig struct {
	Settings    ClientSettings
	LoadOptions []func(options *awsCfg.LoadOptions) error
}

type (
	ClientOption    func(cfg *ClientConfig)
	clientAppCtxKey string
)

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*imds.Client, error) {
	client, err := appctx.GetSet(ctx, clientAppCtxKey(name), func() (interface{}, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
	if err != nil {
		return nil, err
	}

	return client.(*imds.Client), nil
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*imds.Client, error) {
	clientCfg := &ClientConfig{}
	gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "imds", name)

	for _, opt := range optFns {
		opt(clientCfg)
	}

	awsConfig, err := gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg.Settings.ClientSettings, clientCfg.LoadOptions...)
	if err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := imds.NewFromConfig(awsConfig)

	return client, nil
}
