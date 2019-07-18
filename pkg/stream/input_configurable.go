package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

const (
	InputTypeFile    = "file"
	InputTypeKinesis = "kinesis"
	InputTypeRedis   = "redis"
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

func getConfigurableInputKey(name string) string {
	return fmt.Sprintf("input_%s_settings", name)
}
