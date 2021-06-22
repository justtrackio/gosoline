package dynamodb

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Settings struct {
	gosoAws.ClientSettings
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...func(options *awsCfg.LoadOptions) error) (*dynamodb.Client, error) {
	key := fmt.Sprintf("cloud.aws.dynamodb.clients.%s", name)
	settings := &Settings{}
	config.UnmarshalKey(key, settings,
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults", "."),
		cfg.UnmarshalWithDefaultsFromKey("cloud.aws.dynamodb.clients.default", "."),
	)

	return NewClientWithInterfaces(ctx, config, logger, clock.Provider, settings, optFns...)
}

func NewClientWithInterfaces(ctx context.Context, config cfg.Config, logger log.Logger, clock clock.Clock, settings *Settings, optFns ...func(options *awsCfg.LoadOptions) error) (*dynamodb.Client, error) {
	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, logger, clock, settings.ClientSettings, optFns...); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := dynamodb.NewFromConfig(awsConfig)

	return client, nil
}
