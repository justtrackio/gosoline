package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
	"time"
)

const (
	InputTypeFile    = "file"
	InputTypeKinesis = "kinesis"
	InputTypeRedis   = "redis"
	InputTypeSns     = "sns"
	InputTypeSqs     = "sqs"
)

func NewConfigurableInput(config cfg.Config, logger mon.Logger, name string) Input {
	key := fmt.Sprintf("stream.input.%s.type", name)
	t := config.GetString(key)

	switch t {
	case InputTypeFile:
		return newFileInputFromConfig(config, logger, name)
	case InputTypeKinesis:
		return newKinesisInputFromConfig(config, logger, name)
	case InputTypeRedis:
		return newRedisInputFromConfig(config, logger, name)
	case InputTypeSns:
		return newSnsInputFromConfig(config, logger, name)
	case InputTypeSqs:
		return newSqsInputFromConfig(config, logger, name)
	default:
		logger.Fatalf(fmt.Errorf("invalid input %s of type %s", name, t), "invalid input %s of type %s", name, t)
	}

	return nil
}

func newFileInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)
	settings := FileSettings{}
	config.UnmarshalKey(key, &settings)

	return NewFileInput(config, logger, settings)
}

type kinesisInputConfiguration struct {
	StreamName      string `cfg:"stream_name" validate:"required"`
	ApplicationName string `cfg:"application_name" validate:"required"`
}

func newKinesisInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

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
	key := getConfigurableInputKey(name)

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

type snsInputTarget struct {
	Family      string `cfg:"family"`
	Application string `cfg:"application" validate:"required"`
	TopicId     string `cfg:"topic_id" validate:"required"`
}

type snsInputConfiguration struct {
	ConsumerId        string                `cfg:"id" validate:"required"`
	Targets           []snsInputTarget      `cfg:"targets" validate:"min=1"`
	WaitTime          int64                 `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout int                   `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount       int                   `cfg:"runner_count" default:"1" validate:"min=1"`
	RedrivePolicy     sqs.RedrivePolicy     `cfg:"redrive_policy"`
	Client            cloud.ClientSettings  `cfg:"client"`
	Backoff           cloud.BackoffSettings `cfg:"backoff"`
}

func newSnsInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

	configuration := snsInputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	settings := SnsInputSettings{
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
	Family            string                `cfg:"target_family"`
	Application       string                `cfg:"target_application"`
	QueueId           string                `cfg:"target_queue_id" validate:"min=1"`
	WaitTime          int64                 `cfg:"wait_time" default:"3" validate:"min=1"`
	VisibilityTimeout int                   `cfg:"visibility_timeout" default:"30" validate:"min=1"`
	RunnerCount       int                   `cfg:"runner_count" default:"1" validate:"min=1"`
	Fifo              sqs.FifoSettings      `cfg:"fifo"`
	RedrivePolicy     sqs.RedrivePolicy     `cfg:"redrive_policy"`
	Client            cloud.ClientSettings  `cfg:"client"`
	Backoff           cloud.BackoffSettings `cfg:"backoff"`
}

func newSqsInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

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
	}

	return NewSqsInput(config, logger, settings)
}

func getConfigurableInputKey(name string) string {
	return fmt.Sprintf("stream.input.%s", name)
}
