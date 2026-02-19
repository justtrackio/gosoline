package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

func init() {
	AddOutputFactory(OutputTypeFile, newFileOutputFromConfig)
	AddOutputFactory(OutputTypeInMemory, newInMemoryOutputFromConfig)
	AddOutputFactory(OutputTypeKafka, newKafkaOutputFromConfig)
	AddOutputFactory(OutputTypeKinesis, newKinesisOutputFromConfig)
	AddOutputFactory(OutputTypeMultiple, NewConfigurableMultiOutput)
	AddOutputFactory(OutputTypeNoOp, newNoOpOutput)
	AddOutputFactory(OutputTypeRedis, newRedisListOutputFromConfig)
	AddOutputFactory(OutputTypeSns, newSnsOutputFromConfig)
	AddOutputFactory(OutputTypeSqs, newSqsOutputFromConfig)
}

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

var outputFactories = map[string]OutputFactory{}

func AddOutputFactory(name string, factory OutputFactory) {
	outputFactories[name] = factory
}

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

func NewConfigurableOutput(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error) {
	key := fmt.Sprintf("%s.type", ConfigurableOutputKey(name))
	typ, err := config.GetString(key)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get type for output %s: %w", name, err)
	}

	var ok bool
	var factory OutputFactory
	var output Output
	var outputCapabilities *OutputCapabilities

	if factory, ok = outputFactories[typ]; !ok {
		return nil, nil, fmt.Errorf("invalid output %s of type %s", name, typ)
	}

	if output, outputCapabilities, err = factory(ctx, config, logger, name); err != nil {
		return nil, nil, fmt.Errorf("can not create output %s: %w", name, err)
	}

	outputWithTracer, err := NewOutputTracer(ctx, config, logger, output, name)
	if err != nil {
		return nil, nil, fmt.Errorf("can not create output with tracer %s: %w", name, err)
	}

	return outputWithTracer, outputCapabilities, nil
}

func newFileOutputFromConfig(_ context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error) {
	key := ConfigurableOutputKey(name)
	settings := &FileOutputSettings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal file output settings for key %q in newFileOutputFromConfig: %w", key, err)
	}

	if settings.Filename == "" {
		settings.Filename = fmt.Sprintf("stream-output-%s", name)
	}

	return NewFileOutput(config, logger, settings), DefaultOutputCapabilities, nil
}

type InMemoryOutputConfiguration struct {
	BaseOutputConfiguration
	Type string `cfg:"type" default:"inMemory"`
}

func newInMemoryOutputFromConfig(_ context.Context, _ cfg.Config, _ log.Logger, name string) (Output, *OutputCapabilities, error) {
	return ProvideInMemoryOutput(name), DefaultOutputCapabilities, nil
}

type KafkaOutputConfiguration struct {
	BaseOutputConfiguration
	Type       string       `cfg:"type" default:"kafka"`
	Identity   cfg.Identity `cfg:"identity"`
	TopicId    string       `cfg:"topic_id"`
	Connection string       `cfg:"connection" default:"default"`

	// LingerTimeout is the max time the producer will wait for new records before flushing the current batch.
	// When set to 0s, batches will be sent out as fast as possible (or when the size limits are reached with enough back pressure).
	// The kafka library recommends to increase this only when batching with low volume.
	LingerTimeout  time.Duration `cfg:"linger_timeout" default:"0s"`
	RequestTimeout time.Duration `cfg:"request_timeout" default:"10s"`

	MaxBatchSize  int   `cfg:"max_batch_size" default:"10000"`
	MaxBatchBytes int32 `cfg:"max_batch_bytes" default:"1000012"`
}

func newKafkaOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error) {
	key := ConfigurableOutputKey(name)
	configuration := &KafkaOutputConfiguration{}
	if err := config.UnmarshalKey(key, configuration); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal kafka output settings for key %q in newKafkaOutputFromConfig: %w", key, err)
	}

	producerSettings, err := readProducerSettings(config, name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read producer settings for %q: %w", name, err)
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

	outputCapabilities := &OutputCapabilities{
		// we are not using the partitioned producer daemon aggregator.
		// but the kafka library will partition by the AttributeKafkaKey in the message attributes if it is set.
		IsPartitionedOutput: false,
		ProvidesCompression: true,
		// when using the schema registry, we can not aggregate.
		// otherwise, we would write something that does not match the schema.
		// unfortunately, we can also not aggregate when not using the schema registry,
		// because the producer daemon starts running as a module before the schema registry can be initialized
		// and therefore the producer daemon can not know if the schema registry is being used.
		SupportsAggregation: false,
		MaxBatchSize:        mdl.Box(configuration.MaxBatchSize),
		MaxMessageSize:      mdl.Box(int(configuration.MaxBatchBytes)),
		// the kafka library has an internal process for batching and flushing messages.
		// so we always use the size restrictions from the library to prevent it from re-batching and breaking up what we already batched
		// and to have just one place for the batch settings.
		IgnoreProducerDaemonBatchSettings: true,
	}

	output, err := NewKafkaOutput(ctx, config, logger, &kafkaProducer.Settings{
		Identity:       configuration.Identity,
		Connection:     configuration.Connection,
		TopicId:        configuration.TopicId,
		Compression:    compression,
		MaxBatchSize:   configuration.MaxBatchSize,
		MaxBatchBytes:  configuration.MaxBatchBytes,
		LingerTimeout:  configuration.LingerTimeout,
		RequestTimeout: configuration.RequestTimeout,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("can not create kafka output %s: %w", name, err)
	}

	return output, outputCapabilities, nil
}

type KinesisOutputConfiguration struct {
	BaseOutputConfiguration
	Type       string       `cfg:"type" default:"kinesis"`
	Identity   cfg.Identity `cfg:"identity"`
	ClientName string       `cfg:"client_name" default:"default"`
	StreamName string       `cfg:"stream_name"`
}

func newKinesisOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error) {
	key := ConfigurableOutputKey(name)
	configuration := &KinesisOutputConfiguration{}
	if err := config.UnmarshalKey(key, configuration); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal kinesis output settings for key %q in newKinesisOutputFromConfig: %w", key, err)
	}

	outputCapabilities := &OutputCapabilities{
		IsPartitionedOutput:               true,
		ProvidesCompression:               false,
		SupportsAggregation:               true,
		MaxBatchSize:                      mdl.Box(500),
		MaxMessageSize:                    mdl.Box(1024 * 1024),
		IgnoreProducerDaemonBatchSettings: false,
	}

	output, err := NewKinesisOutput(ctx, config, logger, &KinesisOutputSettings{
		Identity:   configuration.Identity,
		ClientName: configuration.ClientName,
		StreamName: configuration.StreamName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("can not create kinesis output %s: %w", name, err)
	}

	return output, outputCapabilities, nil
}

type redisListOutputConfiguration struct {
	ServerName string `cfg:"server_name" default:"default" validate:"required,min=1"`
	Key        string `cfg:"key" validate:"required,min=1"`
	BatchSize  int    `cfg:"batch_size" default:"100"`
}

func newRedisListOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error) {
	key := ConfigurableOutputKey(name)

	configuration := redisListOutputConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal redis list output settings for key %q in newRedisListOutputFromConfig: %w", key, err)
	}

	output, err := NewRedisListOutput(ctx, config, logger, &RedisListOutputSettings{
		ServerName: configuration.ServerName,
		Key:        configuration.Key,
		BatchSize:  configuration.BatchSize,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("can not create redis output %s: %w", name, err)
	}

	return output, DefaultOutputCapabilities, nil
}

type SnsOutputConfiguration struct {
	BaseOutputConfiguration
	Type       string       `cfg:"type" default:"sns"`
	Identity   cfg.Identity `cfg:"identity"`
	TopicId    string       `cfg:"topic_id" validate:"required"`
	ClientName string       `cfg:"client_name" default:"default"`
}

func newSnsOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error) {
	key := ConfigurableOutputKey(name)
	configuration := SnsOutputConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal sns output settings for key %q in newSnsOutputFromConfig: %w", key, err)
	}

	outputCapabilities := &OutputCapabilities{
		IsPartitionedOutput:               false,
		ProvidesCompression:               false,
		SupportsAggregation:               true,
		MaxBatchSize:                      mdl.Box(10),
		MaxMessageSize:                    mdl.Box(256 * 1024),
		IgnoreProducerDaemonBatchSettings: false,
	}

	output, err := NewSnsOutput(ctx, config, logger, &SnsOutputSettings{
		Identity:   configuration.Identity,
		TopicId:    configuration.TopicId,
		ClientName: configuration.ClientName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("can not create sns output %s: %w", name, err)
	}

	return output, outputCapabilities, nil
}

type SqsOutputConfiguration struct {
	BaseOutputConfiguration
	Type              string            `cfg:"type" default:"sqs"`
	Identity          cfg.Identity      `cfg:"identity"`
	QueueId           string            `cfg:"queue_id" validate:"required"`
	VisibilityTimeout int               `cfg:"visibility_timeout" default:"30" validate:"gt=0"`
	RedrivePolicy     sqs.RedrivePolicy `cfg:"redrive_policy"`
	Fifo              sqs.FifoSettings  `cfg:"fifo"`
	ClientName        string            `cfg:"client_name" default:"default"`
}

func newSqsOutputFromConfig(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Output, *OutputCapabilities, error) {
	key := ConfigurableOutputKey(name)
	configuration := SqsOutputConfiguration{}
	if err := config.UnmarshalKey(key, &configuration); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal sqs output settings for key %q in newSqsOutputFromConfig: %w", key, err)
	}

	outputCapabilities := &OutputCapabilities{
		IsPartitionedOutput:               false,
		ProvidesCompression:               false,
		SupportsAggregation:               true,
		MaxBatchSize:                      mdl.Box(10),
		MaxMessageSize:                    mdl.Box(256 * 1024),
		IgnoreProducerDaemonBatchSettings: false,
	}

	output, err := NewSqsOutput(ctx, config, logger, &SqsOutputSettings{
		Identity:          configuration.Identity,
		QueueId:           configuration.QueueId,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RedrivePolicy:     configuration.RedrivePolicy,
		Fifo:              configuration.Fifo,
		ClientName:        configuration.ClientName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("can not create sqs output %s: %w", name, err)
	}

	return output, outputCapabilities, nil
}

func ConfigurableOutputKey(name string) string {
	return fmt.Sprintf("stream.output.%s", name)
}
