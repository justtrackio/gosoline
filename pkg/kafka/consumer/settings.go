package consumer

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka"

	"github.com/justtrackio/gosoline/pkg/kafka/connection"
)

type Settings struct {
	ConnectionName string `cfg:"connection" validate:"required"`
	connection     *connection.Settings

	Topic string `cfg:"topic" validate:"required"`
	// FQTopic is the fully-qualified topic name (with prefixes).
	FQTopic      string
	GroupID      string
	BatchSize    int           `cfg:"batch_size" default:"1"`
	BatchTimeout time.Duration `cfg:"idle_timeout" default:"1s"`
}

func (s *Settings) Connection() *connection.Settings {
	return s.connection
}

func (s *Settings) WithConnection(conn *connection.Settings) *Settings {
	s.connection = conn
	return s
}

func ParseSettings(c cfg.Config, key string) *Settings {
	settings := &Settings{}
	c.UnmarshalKey(key, settings)

	settings.connection = connection.ParseSettings(c, settings.ConnectionName)
	settings.GroupID = cfg.GetAppIdFromConfig(c).Application
	settings.FQTopic = kafka.FQTopicName(cfg.GetAppIdFromConfig(c), settings.Topic)

	return settings
}
