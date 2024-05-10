package athena

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ClientSettings struct {
	gosoAws.ClientSettings
	Database       string        `cfg:"database"`
	OutputLocation string        `cfg:"output_location"`
	PollFrequency  time.Duration `cfg:"poll_frequency" default:"3s"`
}

type ClientConfig struct {
	Settings    ClientSettings
	LoadOptions []func(options *awsCfg.LoadOptions) error
}

func (c ClientConfig) GetSettings() gosoAws.ClientSettings {
	return c.Settings.ClientSettings
}

func (c ClientConfig) GetLoadOptions() []func(options *awsCfg.LoadOptions) error {
	return c.LoadOptions
}

func (c ClientConfig) GetRetryOptions() []func(*retry.StandardOptions) {
	return nil
}

type ClientOption func(cfg *ClientConfig)

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*athena.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*athena.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*athena.Client, error) {
	var err error
	var clientCfg *ClientConfig
	var awsConfig aws.Config

	if clientCfg, awsConfig, err = getConfigs(ctx, config, logger, name, optFns...); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := athena.NewFromConfig(awsConfig)

	gosoAws.LogNewClientCreated(ctx, logger, "athena", name, clientCfg.Settings.ClientSettings)

	return client, nil
}

func getConfigs(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*ClientConfig, aws.Config, error) {
	clientCfg := &ClientConfig{}
	if err := gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "athena", name); err != nil {
		return nil, aws.Config{}, fmt.Errorf("failed to unmarshal Athena client settings: %w", err)
	}

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, awsConfig, fmt.Errorf("can not initialize config: %w", err)
	}

	return clientCfg, awsConfig, nil
}
