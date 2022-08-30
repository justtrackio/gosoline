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

	Topic   string `cfg:"topic" validate:"required"`
	GroupID string `cfg:"group_id"`
	// FQTopic is the fully-qualified topic name (with prefix).
	FQTopic string
	// FQGroupID is the fully-qualified group id (with prefix).
	FQGroupID    string
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

func ParseSettings(config cfg.Config, key string) *Settings {
	settings := &Settings{}
	config.UnmarshalKey(key, settings)

	appID := cfg.GetAppIdFromConfig(config)
	settings.connection = connection.ParseSettings(config, settings.ConnectionName)
	settings.FQGroupID = kafka.FQGroupId(config, appID, settings.GroupID)
	settings.FQTopic = kafka.FQTopicName(config, appID, settings.Topic)

	return settings
}
