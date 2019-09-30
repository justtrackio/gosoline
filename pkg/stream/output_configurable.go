package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
)

const (
	OutputTypeFile    = "file"
	OutputTypeKinesis = "kinesis"
	OutputTypeRedis   = "redis"
	OutputTypeSns     = "sns"
	OutputTypeSqs     = "sqs"
)

func NewConfigurableOutput(config cfg.Config, logger mon.Logger, name string) Output {
	key := fmt.Sprintf("output_%s_type", name)
	t := config.GetString(key)

	switch t {
	case OutputTypeFile:
		return newFileOutputFromConfig(config, logger, name)
	case OutputTypeKinesis:
		return newKinesisOutputFromConfig(config, logger, name)
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
	key := getConfigurableOutputKey(name)
	settings := &FileOutputSettings{}
	config.UnmarshalKey(key, settings)

	return NewFileOutput(config, logger, settings)
}

type kinesisOutputConfiguration struct {
	StreamName string `cfg:"streamName"`
}

func newKinesisOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := getConfigurableOutputKey(name)
	settings := &kinesisOutputConfiguration{}
	config.UnmarshalKey(key, settings)

	return NewKinesisOutput(config, logger, &KinesisOutputSettings{
		StreamName: settings.StreamName,
	})
}

type redisListOutputConfiguration struct {
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
	ServerName  string `cfg:"serverName"`
	Key         string `cfg:"key"`
	BatchSize   int    `cfg:"batchSize"`
}

func newRedisListOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := getConfigurableOutputKey(name)

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

type snsOutputConfiguration struct {
	Project     string `cfg:"project"`
	Family      string `cfg:"family"`
	Application string `cfg:"application"`
	TopicId     string `cfg:"topic_id"`
}

func newSnsOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := getConfigurableOutputKey(name)

	configuration := snsOutputConfiguration{}
	config.UnmarshalKey(key, &configuration)

	return NewSnsOutput(config, logger, SnsOutputSettings{
		AppId: cfg.AppId{
			Project:     configuration.Project,
			Family:      configuration.Family,
			Application: configuration.Application,
		},
		TopicId: configuration.TopicId,
	})
}

type sqsOutputConfiguration struct {
	Project           string            `cfg:"project"`
	Family            string            `cfg:"family"`
	Application       string            `cfg:"application"`
	QueueId           string            `cfg:"queue_id"`
	VisibilityTimeout int               `cfg:"visibility_timeout"`
	RedrivePolicy     sqs.RedrivePolicy `cfg:"redrive_policy"`
}

func newSqsOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := getConfigurableOutputKey(name)

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
	})
}

func getConfigurableOutputKey(name string) string {
	return fmt.Sprintf("output_%s_settings", name)
}
