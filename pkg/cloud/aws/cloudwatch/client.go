package cloudwatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(options *cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(options *cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
	// Whether to disable automatic request compression for supported operations.
	DisableRequestCompression bool `cfg:"disable_request_compression" default:"false"`
	// The minimum request body size, in bytes, at which compression should occur. The
	// default value is 10 KiB. Values must fall within [0, 1MiB].
	RequestMinCompressSizeBytes int64 `cfg:"request_min_compress_size_bytes" default:"0"`
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

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*cloudwatch.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*cloudwatch.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*cloudwatch.Client, error) {
	clientCfg := &ClientConfig{}
	if err := gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "cloudwatch", name); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudWatch client settings: %w", err)
	}

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := cloudwatch.NewFromConfig(awsConfig, func(options *cloudwatch.Options) {
		options.BaseEndpoint = gosoAws.NilIfEmpty(clientCfg.Settings.Endpoint)
		options.DisableRequestCompression = clientCfg.Settings.DisableRequestCompression
		options.RequestMinCompressSizeBytes = clientCfg.Settings.RequestMinCompressSizeBytes
	})

	gosoAws.LogNewClientCreated(ctx, logger, "cloudwatch", name, clientCfg.Settings.ClientSettings)

	return client, nil
}
