package sagemakerruntime

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sagemakerruntime"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	InvokeEndpoint(ctx context.Context, body []byte) ([]byte, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
	Identity    cfg.Identity `cfg:"identity"`
	ContentType string       `cfg:"content_type" default:"application/json"`
	Accept      string       `cfg:"accept" default:"application/json"`
}

type ClientConfig struct {
	Settings     ClientSettings
	LoadOptions  []func(options *awsCfg.LoadOptions) error
	RetryOptions []func(*retry.StandardOptions)
}

func (c ClientConfig) GetSettings() gosoAws.ClientSettings {
	return c.Settings.ClientSettings
}

func (c ClientConfig) GetLoadOptions() []func(options *awsCfg.LoadOptions) error {
	return c.LoadOptions
}

func (c ClientConfig) GetRetryOptions() []func(*retry.StandardOptions) {
	return c.RetryOptions
}

type ClientOption func(cfg *ClientConfig)

type clientAppCtxKey string

type client struct {
	sdkClient    *sagemakerruntime.Client
	settings     ClientSettings
	endpointName string
}

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (Client, error) {
	clientCfg := &ClientConfig{}
	if err := gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "sagemakerruntime", name); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SageMaker Runtime client settings: %w", err)
	}

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	sdkClient := sagemakerruntime.NewFromConfig(awsConfig, func(options *sagemakerruntime.Options) {
		options.BaseEndpoint = gosoAws.NilIfEmpty(clientCfg.Settings.Endpoint)
	})

	endpointName, err := GetEndpointName(config, EndpointNameSettings{
		Identity:   clientCfg.Settings.Identity,
		ClientName: name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get SageMaker endpoint name: %w", err)
	}

	gosoAws.LogNewClientCreated(ctx, logger, "sagemakerruntime", name, clientCfg.Settings.ClientSettings)

	return &client{
		sdkClient:    sdkClient,
		settings:     clientCfg.Settings,
		endpointName: endpointName,
	}, nil
}

func (c *client) InvokeEndpoint(ctx context.Context, body []byte) ([]byte, error) {
	input := &sagemakerruntime.InvokeEndpointInput{
		EndpointName: mdl.Box(c.endpointName),
		ContentType:  mdl.Box(c.settings.ContentType),
		Accept:       mdl.Box(c.settings.Accept),
		Body:         body,
	}

	output, err := c.sdkClient.InvokeEndpoint(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke SageMaker endpoint %s: %w", c.endpointName, err)
	}

	return output.Body, nil
}
