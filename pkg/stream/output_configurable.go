package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	OutputTypeFile     = "file"
	OutputTypeInMemory = "inMemory"
	OutputTypeKinesis  = "kinesis"
	OutputTypeMultiple = "multiple"
	OutputTypeNoOp     = "noop"
	OutputTypeRedis    = "redis"
	OutputTypeSns      = "sns"
	OutputTypeSqs      = "sqs"
	OutputTypeKafka    = "kafka"
)

type BaseOutputConfigurationAware interface {
	SetTracing(enabled bool)
}

type BaseOutputConfiguration struct {
	Tracing BaseOutputConfigurationTracing `cfg:"tracing"`
}

func (b *BaseOutputConfiguration) SetTracing(enabled bool) {
	b.Tracing.Enabled = enabled
}

type BaseOutputConfigurationTracing struct {
	Enabled bool `cfg:"enabled" default:"true"`
}

func NewConfigurableOutput(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	outputFactories := map[string]OutputFactory{
		OutputTypeFile:     newFileOutputFromConfig,
		OutputTypeInMemory: newInMemoryOutputFromConfig,
		OutputTypeKinesis:  newKinesisOutputFromConfig,
		OutputTypeMultiple: NewConfigurableMultiOutput,
		OutputTypeNoOp:     newNoOpOutput,
		OutputTypeRedis:    newRedisListOutputFromConfig,
		OutputTypeSns:      newSnsOutputFromConfig,
		OutputTypeSqs:      newSqsOutputFromConfig,
		OutputTypeKafka:    newKafkaOutputFromConfig,
	}

	key := fmt.Sprintf("%s.type", ConfigurableOutputKey(name))
	typ := config.GetString(key)

	var ok bool
	var err error
	var factory OutputFactory
	var output Output

	if factory, ok = outputFactories[typ]; !ok {
		return nil, fmt.Errorf("invalid output %s of type %s", name, typ)
	}

	if output, err = factory(ctx, config, logger, name); err != nil {
		return nil, fmt.Errorf("can not create output %s: %w", name, err)
	}

	return NewOutputTracer(config, logger, output, name)
}

func newFileOutputFromConfig(_ context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	settings := &FileOutputSettings{}
	config.UnmarshalKey(key, settings)

	if settings.Filename == "" {
		settings.Filename = fmt.Sprintf("stream-output-%s", name)
	}

	return NewFileOutput(config, logger, settings), nil
}

func newKafkaOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	return NewKafkaOutput(ctx, config, logger, key)
}

type InMemoryOutputConfiguration struct {
	BaseOutputConfiguration
	Type string `cfg:"type" default:"inMemory"`
}

func newInMemoryOutputFromConfig(_ context.Context, _ cfg.Config, _ log.Logger, name string) (Output, error) {
	return ProvideInMemoryOutput(name), nil
}

type KinesisOutputConfiguration struct {
	BaseOutputConfiguration
	Type        string `cfg:"type" default:"kinesis"`
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Group       string `cfg:"group"`
	Application string `cfg:"application"`
	ClientName  string `cfg:"client_name" default:"default"`
	StreamName  string `cfg:"stream_name"`
}

func newKinesisOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	configuration := &KinesisOutputConfiguration{}
	config.UnmarshalKey(key, configuration)

	return NewKinesisOutput(ctx, config, logger, &KinesisOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		ClientName: configuration.ClientName,
		StreamName: configuration.StreamName,
	})
}

type redisListOutputConfiguration struct {
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Group       string `cfg:"group"`
	Application string `cfg:"application"`
	ServerName  string `cfg:"server_name" default:"default" validate:"required,min=1"`
	Key         string `cfg:"key" validate:"required,min=1"`
	BatchSize   int    `cfg:"batch_size" default:"100"`
}

func newRedisListOutputFromConfig(_ context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)

	configuration := redisListOutputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	return NewRedisListOutput(config, logger, &RedisListOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		ServerName: configuration.ServerName,
		Key:        configuration.Key,
		BatchSize:  configuration.BatchSize,
	})
}

type SnsOutputConfiguration struct {
	BaseOutputConfiguration
	Type        string `cfg:"type" default:"sns"`
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Group       string `cfg:"group"`
	Application string `cfg:"application"`
	TopicId     string `cfg:"topic_id" validate:"required"`
	ClientName  string `cfg:"client_name" default:"default"`
}

func newSnsOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	configuration := SnsOutputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	return NewSnsOutput(ctx, config, logger, &SnsOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		TopicId:    configuration.TopicId,
		ClientName: configuration.ClientName,
	})
}

type SqsOutputConfiguration struct {
	BaseOutputConfiguration
	Type              string            `cfg:"type" default:"sqs"`
	Project           string            `cfg:"project"`
	Family            string            `cfg:"family"`
	Group             string            `cfg:"group"`
	Application       string            `cfg:"application"`
	QueueId           string            `cfg:"queue_id" validate:"required"`
	VisibilityTimeout int               `cfg:"visibility_timeout" default:"30" validate:"gt=0"`
	RedrivePolicy     sqs.RedrivePolicy `cfg:"redrive_policy"`
	Fifo              sqs.FifoSettings  `cfg:"fifo"`
	ClientName        string            `cfg:"client_name" default:"default"`
}

func newSqsOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	configuration := SqsOutputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	return NewSqsOutput(ctx, config, logger, &SqsOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Group:       configuration.Group,
			Application: configuration.Application,
		},
		QueueId:           configuration.QueueId,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RedrivePolicy:     configuration.RedrivePolicy,
		Fifo:              configuration.Fifo,
		ClientName:        configuration.ClientName,
	})
}

func ConfigurableOutputKey(name string) string {
	return fmt.Sprintf("stream.output.%s", name)
}
