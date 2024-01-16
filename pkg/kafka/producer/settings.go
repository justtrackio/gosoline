package producer

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"

	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
)

type Settings struct {
	ConnectionName string `cfg:"connection" validate:"required"`
	Topic          string `cfg:"topic" validate:"required"`
	// FQTopic is the fully-qualified topic name (with prefixes).
	FQTopic      string
	BatchSize    int           `cfg:"batch_size"`
	BatchTimeout time.Duration `cfg:"idle_timeout"`
	AsyncWrites  bool          `cfg:"async_writes"`
	connection   *connection.Settings
}

func (s *Settings) Connection() *connection.Settings {
	return s.connection
}

func (s *Settings) WithConnection(conn *connection.Settings) *Settings {
	s.connection = conn
	return s
}

func ParseSettings(config cfg.Config, key string) *Settings {
	settings := &Settings{}
	config.UnmarshalKey(key, settings)

	settings.connection = connection.ParseSettings(config, settings.ConnectionName)
	settings.FQTopic = kafka.FQTopicName(config, cfg.GetAppIdFromConfig(config), settings.Topic)

	return settings
}
