package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
	"time"
)

const (
	InputTypeFile     = "file"
	InputTypeInMemory = "inMemory"
	InputTypeKinesis  = "kinesis"
	InputTypeRedis    = "redis"
	InputTypeSns      = "sns"
	InputTypeSqs      = "sqs"
)

type InputFactory func(config cfg.Config, logger mon.Logger, name string) Input

var inputFactories = map[string]InputFactory{
	InputTypeFile:     newFileInputFromConfig,
	InputTypeInMemory: newInMemoryInputFromConfig,
	InputTypeKinesis:  newKinesisInputFromConfig,
	InputTypeRedis:    newRedisInputFromConfig,
	InputTypeSns:      newSnsInputFromConfig,
	InputTypeSqs:      newSqsInputFromConfig,
}

func SetInputFactory(typ string, factory InputFactory) {
	inputFactories[typ] = factory
}

var inputs = map[string]Input{}

func ProvideConfigurableInput(config cfg.Config, logger mon.Logger, name string) Input {
	if input, ok := inputs[name]; ok {
		return input
	}

	inputs[name] = NewConfigurableInput(config, logger, name)

	return inputs[name]
}

func NewConfigurableInput(config cfg.Config, logger mon.Logger, name string) Input {
	key := fmt.Sprintf("stream.input.%s.type", name)
	t := config.GetString(key)

	factory, ok := inputFactories[t]

	if !ok {
		logger.Fatalf(fmt.Errorf("invalid input %s of type %s", name, t), "invalid input %s of type %s", name, t)
	}

	return factory(config, logger, name)
}

func newFileInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := ConfigurableInputKey(name)
	settings := FileSettings{}
	config.UnmarshalKey(key, &settings)

	return NewFileInput(config, logger, settings)
}

func newInMemoryInputFromConfig(config cfg.Config, _ mon.Logger, name string) Input {
	key := ConfigurableInputKey(name)
	settings := &InMemorySettings{}
	config.UnmarshalKey(key, settings)

	return ProvideInMemoryInput(name, settings)
}

type kinesisInputConfiguration struct {
	StreamName      string `cfg:"stream_name" validate:"required"`
	ApplicationName string `cfg:"application_name" validate:"required"`
}

func newKinesisInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := ConfigurableInputKey(name)

	settings := kinesisInputConfiguration{}
	config.UnmarshalKey(key, &settings)

	readerSettings := KinsumerSettings{
		StreamName:      settings.StreamName,
		ApplicationName: settings.ApplicationName,
	}

	return NewKinsumerInput(config, logger, NewKinsumer, readerSettings)
}

type redisInputConfiguration struct {
	Project     string        `cfg:"project"`
	Family      string        `cfg:"family"`
	Application string        `cfg:"application"`
	ServerName  string        `cfg:"server_name" default:"default" validate:"min=1"`
	Key         string        `cfg:"key" validate:"required,min=1"`
	WaitTime    time.Duration `cfg:"wait_time" default:"3s"`
}

func newRedisInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := ConfigurableInputKey(name)

	configuration := redisInputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	settings := &RedisListInputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		ServerName: configuration.ServerName,
		Key:        configuration.Key,
		WaitTime:   configuration.WaitTime,
	}

	return NewRedisListInput(config, logger, settings)
}

type SnsInputTargetConfiguration struct {
	Family      string `cfg:"family"`
	Application string `cfg:"application" validate:"required"`
	TopicId     string `cfg:"topic_id" validate:"required"`
}

type SnsInputConfiguration struct {
	Type              string                        `cfg:"type" default:"sns"`
	ConsumerId        string                        `cfg:"id" validate:"required"`
	Family            string                        `cfg:"family" default:""`
	Application       string                        `cfg:"application" default:""`
	Targets           []SnsInputTargetConfiguration `cfg:"targets" validate:"min=1"`
	WaitTime          int64                         `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout int                           `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount       int                           `cfg:"runner_count" default:"1" validate:"min=1"`
	RedrivePolicy     sqs.RedrivePolicy             `cfg:"redrive_policy"`
	Client            cloud.ClientSettings          `cfg:"client"`
	Backoff           exec.BackoffSettings          `cfg:"backoff"`
}

func newSnsInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := ConfigurableInputKey(name)

	configuration := &SnsInputConfiguration{}
	config.UnmarshalKey(key, configuration)

	settings := SnsInputSettings{
		AppId: cfg.AppId{
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		QueueId:           configuration.ConsumerId,
		WaitTime:          configuration.WaitTime,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RunnerCount:       configuration.RunnerCount,
		RedrivePolicy:     configuration.RedrivePolicy,
		Client:            configuration.Client,
		Backoff:           configuration.Backoff,
	}

	targets := make([]SnsInputTarget, len(configuration.Targets))
	for i, t := range configuration.Targets {
		targets[i] = SnsInputTarget{
			AppId: cfg.AppId{
				Family:      t.Family,
				Application: t.Application,
			},
			TopicId: t.TopicId,
		}
	}

	return NewSnsInput(config, logger, settings, targets)
}

type sqsInputConfiguration struct {
	Family            string               `cfg:"target_family"`
	Application       string               `cfg:"target_application"`
	QueueId           string               `cfg:"target_queue_id" validate:"min=1"`
	WaitTime          int64                `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout int                  `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount       int                  `cfg:"runner_count" default:"1" validate:"min=1"`
	Fifo              sqs.FifoSettings     `cfg:"fifo"`
	RedrivePolicy     sqs.RedrivePolicy    `cfg:"redrive_policy"`
	Client            cloud.ClientSettings `cfg:"client"`
	Backoff           exec.BackoffSettings `cfg:"backoff"`
	Unmarshaller      string               `cfg:"unmarshaller" default:"msg"`
}

func newSqsInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := ConfigurableInputKey(name)

	configuration := sqsInputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	settings := SqsInputSettings{
		AppId: cfg.AppId{
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		QueueId:           configuration.QueueId,
		WaitTime:          configuration.WaitTime,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RunnerCount:       configuration.RunnerCount,
		Fifo:              configuration.Fifo,
		RedrivePolicy:     configuration.RedrivePolicy,
		Client:            configuration.Client,
		Backoff:           configuration.Backoff,
		Unmarshaller:      configuration.Unmarshaller,
	}

	return NewSqsInput(config, logger, settings)
}

func ConfigurableInputKey(name string) string {
	return fmt.Sprintf("stream.input.%s", name)
}
