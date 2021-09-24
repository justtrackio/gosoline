package ssm

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Client
type Client interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
}

type ClientConfig struct {
	Settings    ClientSettings
	LoadOptions []func(options *awsCfg.LoadOptions) error
}

type ClientOption func(cfg *ClientConfig)

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*ssm.Client, error) {
	client, err := appctx.Provide(ctx, clientAppCtxKey(name), func() (interface{}, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
	if err != nil {
		return nil, err
	}

	return client.(*ssm.Client), nil
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*ssm.Client, error) {
	clientCfg := &ClientConfig{}
	gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "ssm", name)

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg.Settings.ClientSettings, clientCfg.LoadOptions...); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := ssm.NewFromConfig(awsConfig)

	return client, nil
}
