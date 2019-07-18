package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	OutputTypeFile    = "file"
	OutputTypeKinesis = "kinesis"
	OutputTypeRedis   = "redis"
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
	default:
		logger.Fatalf(fmt.Errorf("invalid input %s of type %s", name, t), "invalid input %s of type %s", name, t)
	}

	return nil
}

func newFileOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := getConfigurableOutputKey(name)
	settings := &FileOutputSettings{}
	config.Unmarshal(key, settings)

	return NewFileOutput(config, logger, settings)
}

type kinesisOutputConfiguration struct {
	StreamName string `mapstructure:"streamName"`
}

func newKinesisOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := getConfigurableOutputKey(name)
	settings := &kinesisOutputConfiguration{}
	config.Unmarshal(key, settings)

	return NewKinesisOutput(config, logger, &KinesisOutputSettings{
		StreamName: settings.StreamName,
	})
}

type redisListOutputConfiguration struct {
	Project     string `mapstructure:"project"`
	Family      string `mapstructure:"family"`
	Application string `mapstructure:"application"`
	ServerName  string `mapstructure:"serverName"`
	Key         string `mapstructure:"key"`
	BatchSize   int    `mapstructure:"batchSize"`
}

func newRedisListOutputFromConfig(config cfg.Config, logger mon.Logger, name string) Output {
	key := getConfigurableOutputKey(name)

	configuration := redisListOutputConfiguration{}
	config.Unmarshal(key, &configuration)

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

func getConfigurableOutputKey(name string) string {
	return fmt.Sprintf("output_%s_settings", name)
}
