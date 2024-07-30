package cloudwatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Client
type Client interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(options *cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(options *cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*cloudwatch.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*cloudwatch.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*cloudwatch.Client, error) {
	clientCfg := &ClientConfig{}
	gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "cloudwatch", name)

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := cloudwatch.NewFromConfig(awsConfig)

	gosoAws.LogNewClientCreated(ctx, logger, "cloudwatch", name, clientCfg.Settings.ClientSettings)

	return client, nil
}
