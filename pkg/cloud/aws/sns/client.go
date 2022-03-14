package sns

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Client
type Client interface {
	CreateTopic(ctx context.Context, params *sns.CreateTopicInput, optFns ...func(options *sns.Options)) (*sns.CreateTopicOutput, error)
	GetSubscriptionAttributes(ctx context.Context, params *sns.GetSubscriptionAttributesInput, optFns ...func(*sns.Options)) (*sns.GetSubscriptionAttributesOutput, error)
	ListSubscriptionsByTopic(ctx context.Context, params *sns.ListSubscriptionsByTopicInput, optFns ...func(*sns.Options)) (*sns.ListSubscriptionsByTopicOutput, error)
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(options *sns.Options)) (*sns.PublishOutput, error)
	PublishBatch(ctx context.Context, input *sns.PublishBatchInput, optFns ...func(options *sns.Options)) (*sns.PublishBatchOutput, error)
	Subscribe(ctx context.Context, params *sns.SubscribeInput, optFns ...func(options *sns.Options)) (*sns.SubscribeOutput, error)
	Unsubscribe(ctx context.Context, params *sns.UnsubscribeInput, optFns ...func(*sns.Options)) (*sns.UnsubscribeOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
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

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*sns.Client, error) {
	client, err := appctx.Provide(ctx, clientAppCtxKey(name), func() (interface{}, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
	if err != nil {
		return nil, err
	}

	return client.(*sns.Client), nil
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*sns.Client, error) {
	clientCfg := &ClientConfig{}
	gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "sns", name)

	for _, opt := range optFns {
		opt(clientCfg)
	}

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := sns.NewFromConfig(awsConfig)

	return client, nil
}
