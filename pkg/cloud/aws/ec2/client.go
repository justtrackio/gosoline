package ec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Client
type Client interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

type clientAppCtxKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*ec2.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*ec2.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*ec2.Client, error) {
	clientCfg := &ClientConfig{}
	gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "ec2", name)

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := ec2.NewFromConfig(awsConfig)

	gosoAws.LogNewClientCreated(ctx, logger, "ec2", name, clientCfg.Settings.ClientSettings)

	return client, nil
}
