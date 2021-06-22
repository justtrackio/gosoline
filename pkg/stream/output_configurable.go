package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/sqs"
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
)

type BaseOutputSettings struct {
	Backoff exec.BackoffSettings `cfg:"backoff"`
	Tracing struct {
		Enabled bool `cfg:"enabled" default:"true"`
	} `cfg:"tracing"`
}

func NewConfigurableOutput(config cfg.Config, logger log.Logger, name string) (Output, error) {
	var outputFactories = map[string]OutputFactory{
		OutputTypeFile:     newFileOutputFromConfig,
		OutputTypeInMemory: newInMemoryOutputFromConfig,
		OutputTypeKinesis:  newKinesisOutputFromConfig,
		OutputTypeMultiple: NewConfigurableMultiOutput,
		OutputTypeNoOp:     newNoOpOutput,
		OutputTypeRedis:    newRedisListOutputFromConfig,
		OutputTypeSns:      newSnsOutputFromConfig,
		OutputTypeSqs:      newSqsOutputFromConfig,
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

	if output, err = factory(config, logger, name); err != nil {
		return nil, fmt.Errorf("can not create output %s: %w", name, err)
	}

	return NewOutputTracer(config, logger, output, name)
}

func newFileOutputFromConfig(config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	settings := &FileOutputSettings{}
	config.UnmarshalKey(key, settings)

	return NewFileOutput(config, logger, settings), nil
}

func newInMemoryOutputFromConfig(_ cfg.Config, _ log.Logger, name string) (Output, error) {
	return ProvideInMemoryOutput(name), nil
}

type kinesisOutputConfiguration struct {
	StreamName string               `cfg:"stream_name"`
	Backoff    exec.BackoffSettings `cfg:"backoff"`
}

func newKinesisOutputFromConfig(config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)
	settings := &kinesisOutputConfiguration{}
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey(ConfigKeyStreamBackoff, "backoff"))

	return NewKinesisOutput(config, logger, &KinesisOutputSettings{
		StreamName: settings.StreamName,
		Backoff:    settings.Backoff,
	})
}

type redisListOutputConfiguration struct {
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
	ServerName  string `cfg:"server_name" default:"default" validate:"required,min=1"`
	Key         string `cfg:"key" validate:"required,min=1"`
	BatchSize   int    `cfg:"batch_size" default:"100"`
}

func newRedisListOutputFromConfig(config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)

	configuration := redisListOutputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	return NewRedisListOutput(config, logger, &RedisListOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		ServerName: configuration.ServerName,
		Key:        configuration.Key,
		BatchSize:  configuration.BatchSize,
	})
}

type SnsOutputConfiguration struct {
	BaseOutputSettings
	Type        string               `cfg:"type"`
	Project     string               `cfg:"project"`
	Family      string               `cfg:"family"`
	Application string               `cfg:"application"`
	TopicId     string               `cfg:"topic_id" validate:"required"`
	Client      cloud.ClientSettings `cfg:"client"`
}

func newSnsOutputFromConfig(config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)

	configuration := SnsOutputConfiguration{}
	config.UnmarshalKey(key, &configuration, cfg.UnmarshalWithDefaultsFromKey(ConfigKeyStreamBackoff, "backoff"))

	return NewSnsOutput(config, logger, SnsOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		TopicId: configuration.TopicId,
		Client:  configuration.Client,
		Backoff: configuration.Backoff,
	})
}

type sqsOutputConfiguration struct {
	BaseOutputSettings
	Project           string               `cfg:"project"`
	Family            string               `cfg:"family"`
	Application       string               `cfg:"application"`
	QueueId           string               `cfg:"queue_id" validate:"required"`
	VisibilityTimeout int                  `cfg:"visibility_timeout" default:"30" validate:"gt=0"`
	RedrivePolicy     sqs.RedrivePolicy    `cfg:"redrive_policy"`
	Client            cloud.ClientSettings `cfg:"client"`
	Fifo              sqs.FifoSettings     `cfg:"fifo"`
}

func newSqsOutputFromConfig(config cfg.Config, logger log.Logger, name string) (Output, error) {
	key := ConfigurableOutputKey(name)

	configuration := sqsOutputConfiguration{}
	config.UnmarshalKey(key, &configuration, cfg.UnmarshalWithDefaultsFromKey(ConfigKeyStreamBackoff, "backoff"))

	return NewSqsOutput(config, logger, SqsOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		QueueId:           configuration.QueueId,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RedrivePolicy:     configuration.RedrivePolicy,
		Client:            configuration.Client,
		Backoff:           configuration.Backoff,
		Fifo:              configuration.Fifo,
	})
}

func ConfigurableOutputKey(name string) string {
	return fmt.Sprintf("stream.output.%s", name)
}
