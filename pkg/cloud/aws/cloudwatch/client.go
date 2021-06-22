package cloudwatch

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
)

type Settings struct {
	gosoAws.ClientSettings
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...func(options *awsCfg.LoadOptions) error) (*cloudwatch.Client, error) {
	key := fmt.Sprintf("cloud.aws.cloudwatch.clients.%s", name)
	settings := &Settings{}
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("cloud.aws.defaults", "."))

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, logger, clock.Provider, settings.ClientSettings, optFns...); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := cloudwatch.NewFromConfig(awsConfig)

	return client, nil
}
