package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/kafka"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	OutputTypeFile     = "file"
	OutputTypeInMemory = "inMemory"
	OutputTypeKafka    = "kafka"
	OutputTypeKinesis  = "kinesis"
	OutputTypeMultiple = "multiple"
	OutputTypeNoOp     = "noop"
	OutputTypeRedis    = "redis"
	OutputTypeSns      = "sns"
	OutputTypeSqs      = "sqs"
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
		OutputTypeKafka:    newKafkaOutputFromConfig,
		OutputTypeKinesis:  newKinesisOutputFromConfig,
		OutputTypeMultiple: NewConfigurableMultiOutput,
		OutputTypeNoOp:     newNoOpOutput,
		OutputTypeRedis:    newRedisListOutputFromConfig,
		OutputTypeSns:      newSnsOutputFromConfig,
		OutputTypeSqs:      newSqsOutputFromConfig,
	}

	key := fmt.Sprintf("%s.type", ConfigurableOutputKey(name))
	typ, err := config.GetString(key)
	if err != nil {
		return nil, fmt.Errorf("could not get type for output %s: %w", name, err)
	}

	var ok bool
	var factory OutputFactory
	var output Output

	if factory, ok = outputFactories[typ]; !ok {
		return nil, fmt.Errorf("invalid output %s of type %s", name, typ)
	}

	if output, err = factory(ctx, config, logger, name); err != nil {
		return nil, fmt.Errorf("can not create output %s: %w", name, err)
	}

	return NewOutputTracer(ctx, config, logger, output, name)
}

func newFileOutputFromConfig(_ context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	settings := &FileOutputSettings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file output settings for key %q in newFileOutputFromConfig: %w", key, err)
	}

	if settings.Filename == "" {
		settings.Filename = fmt.Sprintf("stream-output-%s", name)
	}

	return NewFileOutput(config, logger, settings), nil
}

type InMemoryOutputConfiguration struct {
	BaseOutputConfiguration
	Type string `cfg:"type" default:"inMemory"`
}

func newInMemoryOutputFromConfig(_ context.Context, _ cfg.Config, _ log.Logger, name string) (Output, error) {
	return ProvideInMemoryOutput(name), nil
}

type KafkaOutputConfiguration struct {
	BaseOutputConfiguration
	Type        string `cfg:"type" default:"kafka"`
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Group       string `cfg:"group"`
	Application string `cfg:"application"`
	TopicId     string `cfg:"topic_id"`
	Connection  string `cfg:"connection" default:"default"`
}

func newKafkaOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	configuration := &KafkaOutputConfiguration{}
	if err := config.UnmarshalKey(key, configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kafka output settings for key %q in newKafkaOutputFromConfig: %w", key, err)
	}

	appId := cfg.AppId{
		Project:     configuration.Project,
		Family:      configuration.Family,
		Group:       configuration.Group,
		Application: configuration.Application,
	}

	topic, err := kafka.BuildFullTopicName(config, appId, configuration.TopicId)
	if err != nil {
		return nil, fmt.Errorf("failed to build full topic name for topic id %q: %w", configuration.TopicId, err)
	}

	producerSettings, err := readProducerSettings(config, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read producer settings for %q: %w", name, err)
	}

	compression := kafkaProducer.CompressionNone

	switch producerSettings.Compression {
	case CompressionGZip:
		compression = kafkaProducer.CompressionGZip
	case CompressionSnappy:
		compression = kafkaProducer.CompressionSnappy
	case CompressionLZ4:
		compression = kafkaProducer.CompressionLZ4
	case CompressionZstd:
		compression = kafkaProducer.CompressionZstd
	}

	return NewKafkaOutput(ctx, config, logger, &kafkaProducer.Settings{
		Connection:  configuration.Connection,
		Topic:       topic,
		Compression: compression,
	})
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
	if err := config.UnmarshalKey(key, configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kinesis output settings for key %q in newKinesisOutputFromConfig: %w", key, err)
	}

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

func newRedisListOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)

	configuration := redisListOutputConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal redis list output settings for key %q in newRedisListOutputFromConfig: %w", key, err)
	}

	return NewRedisListOutput(ctx, config, logger, &RedisListOutputSettings{
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
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sns output settings for key %q in newSnsOutputFromConfig: %w", key, err)
	}

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
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sqs output settings for key %q in newSqsOutputFromConfig: %w", key, err)
	}

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
