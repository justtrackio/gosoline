package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
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
	key := fmt.Sprintf("input_%s_type", name)
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
	StreamName      string `cfg:"streamName"`
	ApplicationName string `cfg:"applicationName"`
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
	ServerName  string        `cfg:"serverName"`
	Key         string        `cfg:"key"`
	WaitTime    time.Duration `cfg:"waitTime"`
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
		WaitTime:   configuration.WaitTime * time.Second,
	}

	return NewRedisListInput(config, logger, settings)
}

type snsInputTarget struct {
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
	TopicId     string `cfg:"topic_id"`
}

type snsInputConfiguration struct {
	ConsumerId        string            `cfg:"id"`
	WaitTime          int64             `cfg:"wait_time"`
	VisibilityTimeout int               `cfg:"visibility_timeout"`
	RedrivePolicy     sqs.RedrivePolicy `cfg:"redrive_policy"`
	RunnerCount       int               `cfg:"runner_count"`
	Targets           []snsInputTarget  `cfg:"targets"`
}

func newSnsInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

	configuration := snsInputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	settings := SnsInputSettings{
		QueueId:           configuration.ConsumerId,
		WaitTime:          configuration.WaitTime,
		RedrivePolicy:     configuration.RedrivePolicy,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RunnerCount:       configuration.RunnerCount,
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
	Family            string            `cfg:"target_family"`
	Application       string            `cfg:"target_application"`
	QueueId           string            `cfg:"target_queue_id"`
	Fifo              sqs.FifoSettings  `cfg:"fifo"`
	WaitTime          int64             `cfg:"wait_time"`
	VisibilityTimeout int               `cfg:"visibility_timeout"`
	RedrivePolicy     sqs.RedrivePolicy `cfg:"redrive_policy"`
	RunnerCount       int               `cfg:"runner_count"`
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
		Fifo:              configuration.Fifo,
		WaitTime:          configuration.WaitTime,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RedrivePolicy:     configuration.RedrivePolicy,
		RunnerCount:       configuration.RunnerCount,
	}

	return NewSqsInput(config, logger, settings)
}

func getConfigurableInputKey(name string) string {
	return fmt.Sprintf("input_%s_settings", name)
}
