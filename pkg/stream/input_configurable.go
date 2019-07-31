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
	config.Unmarshal(key, &settings)

	return NewFileInput(config, logger, settings)
}

type kinesisInputConfiguration struct {
	StreamName      string `mapstructure:"streamName"`
	ApplicationName string `mapstructure:"applicationName"`
}

func newKinesisInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

	settings := kinesisInputConfiguration{}
	config.Unmarshal(key, &settings)

	readerSettings := KinsumerSettings{
		StreamName:      settings.StreamName,
		ApplicationName: settings.ApplicationName,
	}

	return NewKinsumerInput(config, logger, NewKinsumer, readerSettings)
}

type redisInputConfiguration struct {
	Project     string        `mapstructure:"project"`
	Family      string        `mapstructure:"family"`
	Application string        `mapstructure:"application"`
	ServerName  string        `mapstructure:"serverName"`
	Key         string        `mapstructure:"key"`
	WaitTime    time.Duration `mapstructure:"waitTime"`
}

func newRedisInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

	configuration := redisInputConfiguration{}
	config.Unmarshal(key, &configuration)

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
	Family      string `mapstructure:"family"`
	Application string `mapstructure:"application"`
	TopicId     string `mapstructure:"topic_id"`
}

type snsInputConfiguration struct {
	ConsumerId        string            `mapstructure:"id"`
	WaitTime          int64             `mapstructure:"wait_time"`
	VisibilityTimeout int               `mapstructure:"visibility_timeout"`
	RedrivePolicy     sqs.RedrivePolicy `mapstructure:"redrive_policy"`
	Targets           []snsInputTarget  `mapstructure:"targets"`
}

func newSnsInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

	configuration := snsInputConfiguration{}
	config.Unmarshal(key, &configuration)

	settings := SnsInputSettings{
		QueueId:           configuration.ConsumerId,
		WaitTime:          configuration.WaitTime,
		RedrivePolicy:     configuration.RedrivePolicy,
		VisibilityTimeout: configuration.VisibilityTimeout,
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
	Family            string            `mapstructure:"target_family"`
	Application       string            `mapstructure:"target_application"`
	QueueId           string            `mapstructure:"target_queue_id"`
	WaitTime          int64             `mapstructure:"wait_time"`
	VisibilityTimeout int               `mapstructure:"visibility_timeout"`
	RedrivePolicy     sqs.RedrivePolicy `mapstructure:"redrive_policy"`
}

func newSqsInputFromConfig(config cfg.Config, logger mon.Logger, name string) Input {
	key := getConfigurableInputKey(name)

	configuration := sqsInputConfiguration{}
	config.Unmarshal(key, &configuration)

	settings := SqsInputSettings{
		AppId: cfg.AppId{
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		QueueId:           configuration.QueueId,
		WaitTime:          configuration.WaitTime,
		VisibilityTimeout: configuration.VisibilityTimeout,
		RedrivePolicy:     configuration.RedrivePolicy,
	}

	return NewSqsInput(config, logger, settings)
}

func getConfigurableInputKey(name string) string {
	return fmt.Sprintf("input_%s_settings", name)
}
