package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
)

const (
	OutputTypeFile     = "file"
	OutputTypeInMemory = "inMemory"
	OutputTypeKafka    = "kafka"
	OutputTypeKinesis  = "kinesis"
	OutputTypeMultiple = "multiple"
	OutputTypeRedis    = "redis"
	OutputTypeSns      = "sns"
	OutputTypeSqs      = "sqs"
)

func NewConfigurableOutput(config cfg.Config, logger mon.Logger, name string) Output {
	key := fmt.Sprintf("%s.type", ConfigurableOutputKey(name))
	t := config.GetString(key)

	switch t {
	case OutputTypeFile:
		return newFileOutputFromConfig(config, logger, name)
	case OutputTypeInMemory:
		return ProvideInMemoryOutput(name)
	case OutputTypeKinesis:
		return newKinesisOutputFromConfig(config, logger, name)
	case OutputTypeMultiple:
		return newMultipleOutput(config, logger, name)
	case OutputTypeRedis:
		return newRedisListOutputFromConfig(config, logger, name)
	case OutputTypeSns:
		return newSnsOutputFromConfig(config, logger, name)
	case OutputTypeSqs:
		return newSqsOutputFromConfig(config, logger, name)
	default:
		logger.Fatalf(fmt.Errorf("invalid input %s of type %s", name, t), "invalid input %s of type %s", name, t)
	}

	return nil
}

func newFileOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := ConfigurableOutputKey(name)
	settings := &FileOutputSettings{}
	config.UnmarshalKey(key, settings)

	return NewFileOutput(config, logger, settings)
}

func newKafkaOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := ConfigurableOutputKey(name)
	settings := &KafkaOutputSettings{}
	config.UnmarshalKey(key, settings)

	return NewKafkaOutput(logger, settings)
}

type kinesisOutputConfiguration struct {
	StreamName string               `cfg:"stream_name"`
	Backoff    exec.BackoffSettings `cfg:"backoff"`
}

func newKinesisOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := ConfigurableOutputKey(name)
	settings := &kinesisOutputConfiguration{}
	config.UnmarshalKey(key, settings)

	return NewKinesisOutput(config, logger, &KinesisOutputSettings{
		StreamName: settings.StreamName,
		Backoff:    settings.Backoff,
	})
}

func newMultipleOutput(config cfg.Config, logger mon.Logger, name string) Output {
	return NewConfigurableMultiOutput(config, logger, name)
}

type redisListOutputConfiguration struct {
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
	ServerName  string `cfg:"server_name" default:"default" validate:"required,min=1"`
	Key         string `cfg:"key" validate:"required,min=1"`
	BatchSize   int    `cfg:"batch_size" default:"100"`
}

func newRedisListOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
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
	Type        string               `cfg:"type"`
	Project     string               `cfg:"project"`
	Family      string               `cfg:"family"`
	Application string               `cfg:"application"`
	TopicId     string               `cfg:"topic_id" validate:"required"`
	Client      cloud.ClientSettings `cfg:"client"`
	Backoff     exec.BackoffSettings `cfg:"backoff"`
}

func newSnsOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := ConfigurableOutputKey(name)

	configuration := SnsOutputConfiguration{}
	config.UnmarshalKey(key, &configuration)

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
	Project           string               `cfg:"project"`
	Family            string               `cfg:"family"`
	Application       string               `cfg:"application"`
	QueueId           string               `cfg:"queue_id" validate:"required"`
	VisibilityTimeout int                  `cfg:"visibility_timeout" default:"30" validate:"gt=0"`
	RedrivePolicy     sqs.RedrivePolicy    `cfg:"redrive_policy"`
	Client            cloud.ClientSettings `cfg:"client"`
	Backoff           exec.BackoffSettings `cfg:"backoff"`
	Fifo              sqs.FifoSettings     `cfg:"fifo"`
}

func newSqsOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := ConfigurableOutputKey(name)

	configuration := sqsOutputConfiguration{}
	config.UnmarshalKey(key, &configuration)

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
