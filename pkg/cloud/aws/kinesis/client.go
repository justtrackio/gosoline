package kinesis

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name Client
type Client interface {
	AddTagsToStream(ctx context.Context, params *kinesis.AddTagsToStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.AddTagsToStreamOutput, error)
	CreateStream(ctx context.Context, params *kinesis.CreateStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.CreateStreamOutput, error)
	DecreaseStreamRetentionPeriod(ctx context.Context, params *kinesis.DecreaseStreamRetentionPeriodInput, optFns ...func(*kinesis.Options)) (*kinesis.DecreaseStreamRetentionPeriodOutput, error)
	DeleteStream(ctx context.Context, params *kinesis.DeleteStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DeleteStreamOutput, error)
	DeregisterStreamConsumer(ctx context.Context, params *kinesis.DeregisterStreamConsumerInput, optFns ...func(*kinesis.Options)) (*kinesis.DeregisterStreamConsumerOutput, error)
	DescribeLimits(ctx context.Context, params *kinesis.DescribeLimitsInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeLimitsOutput, error)
	DescribeStream(ctx context.Context, params *kinesis.DescribeStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamOutput, error)
	DescribeStreamConsumer(ctx context.Context, params *kinesis.DescribeStreamConsumerInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamConsumerOutput, error)
	DescribeStreamSummary(ctx context.Context, params *kinesis.DescribeStreamSummaryInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamSummaryOutput, error)
	DisableEnhancedMonitoring(ctx context.Context, params *kinesis.DisableEnhancedMonitoringInput, optFns ...func(*kinesis.Options)) (*kinesis.DisableEnhancedMonitoringOutput, error)
	EnableEnhancedMonitoring(ctx context.Context, params *kinesis.EnableEnhancedMonitoringInput, optFns ...func(*kinesis.Options)) (*kinesis.EnableEnhancedMonitoringOutput, error)
	GetRecords(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.GetRecordsOutput, error)
	GetShardIterator(ctx context.Context, params *kinesis.GetShardIteratorInput, optFns ...func(*kinesis.Options)) (*kinesis.GetShardIteratorOutput, error)
	IncreaseStreamRetentionPeriod(ctx context.Context, params *kinesis.IncreaseStreamRetentionPeriodInput, optFns ...func(*kinesis.Options)) (*kinesis.IncreaseStreamRetentionPeriodOutput, error)
	ListShards(ctx context.Context, params *kinesis.ListShardsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListShardsOutput, error)
	ListStreamConsumers(ctx context.Context, params *kinesis.ListStreamConsumersInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamConsumersOutput, error)
	ListStreams(ctx context.Context, params *kinesis.ListStreamsInput, optFns ...func(*kinesis.Options)) (*kinesis.ListStreamsOutput, error)
	ListTagsForStream(ctx context.Context, params *kinesis.ListTagsForStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.ListTagsForStreamOutput, error)
	MergeShards(ctx context.Context, params *kinesis.MergeShardsInput, optFns ...func(*kinesis.Options)) (*kinesis.MergeShardsOutput, error)
	PutRecord(ctx context.Context, params *kinesis.PutRecordInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error)
	PutRecords(ctx context.Context, params *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error)
	RegisterStreamConsumer(ctx context.Context, params *kinesis.RegisterStreamConsumerInput, optFns ...func(*kinesis.Options)) (*kinesis.RegisterStreamConsumerOutput, error)
	RemoveTagsFromStream(ctx context.Context, params *kinesis.RemoveTagsFromStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.RemoveTagsFromStreamOutput, error)
	SplitShard(ctx context.Context, params *kinesis.SplitShardInput, optFns ...func(*kinesis.Options)) (*kinesis.SplitShardOutput, error)
	StartStreamEncryption(ctx context.Context, params *kinesis.StartStreamEncryptionInput, optFns ...func(*kinesis.Options)) (*kinesis.StartStreamEncryptionOutput, error)
	StopStreamEncryption(ctx context.Context, params *kinesis.StopStreamEncryptionInput, optFns ...func(*kinesis.Options)) (*kinesis.StopStreamEncryptionOutput, error)
	SubscribeToShard(ctx context.Context, params *kinesis.SubscribeToShardInput, optFns ...func(*kinesis.Options)) (*kinesis.SubscribeToShardOutput, error)
	UpdateShardCount(ctx context.Context, params *kinesis.UpdateShardCountInput, optFns ...func(*kinesis.Options)) (*kinesis.UpdateShardCountOutput, error)
	UpdateStreamMode(ctx context.Context, params *kinesis.UpdateStreamModeInput, optFns ...func(*kinesis.Options)) (*kinesis.UpdateStreamModeOutput, error)
}

type ClientSettings struct {
	gosoAws.ClientSettings
	ReadProvisionedThroughputDelay time.Duration `cfg:"read_provisioned_throughput_exceeded_delay" default:"1s"`
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

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*kinesis.Client, error) {
	return appctx.Provide(ctx, clientAppCtxKey(name), func() (*kinesis.Client, error) {
		return NewClient(ctx, config, logger, name, optFns...)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger, name string, optFns ...ClientOption) (*kinesis.Client, error) {
	clientCfg := &ClientConfig{}
	if err := gosoAws.UnmarshalClientSettings(config, &clientCfg.Settings, "kinesis", name); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Kinesis client settings: %w", err)
	}

	settings := clientCfg.Settings
	backoffDelayer := NewBackoffDelayer(settings.Backoff.InitialInterval, settings.Backoff.MaxInterval, settings.ReadProvisionedThroughputDelay)
	clientCfg.RetryOptions = []func(*retry.StandardOptions){
		gosoAws.RetryWithBackoff(backoffDelayer),
	}

	for _, opt := range optFns {
		opt(clientCfg)
	}

	logger = logger.WithChannel(fmt.Sprintf("kinesis-%s", name))

	var err error
	var awsConfig aws.Config

	if awsConfig, err = gosoAws.DefaultClientConfig(ctx, config, logger, clientCfg); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	client := kinesis.NewFromConfig(awsConfig, func(options *kinesis.Options) {
		options.BaseEndpoint = gosoAws.NilIfEmpty(clientCfg.Settings.Endpoint)
	})

	gosoAws.LogNewClientCreated(ctx, logger, "kinesis", name, clientCfg.Settings.ClientSettings)

	return client, nil
}
