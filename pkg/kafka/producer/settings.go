package producer

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
)

type Settings struct {
	ConnectionName string `cfg:"connection" default:"default"`
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

func ParseSettings(config cfg.Config, key string) (*Settings, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kafka producer settings for key %q in ParseSettings: %w", key, err)
	}

	conn, err := connection.ParseSettings(config, settings.ConnectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection %q in ParseSettings: %w", settings.ConnectionName, err)
	}
	settings.connection = conn
	settings.FQTopic, err = kafka.FQTopicName(config, cfg.GetAppIdFromConfig(config), settings.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get fully qualified topic name for topic %q in ParseSettings: %w", settings.Topic, err)
	}

	return settings, nil
}
