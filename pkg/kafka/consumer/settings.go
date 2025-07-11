package consumer

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

type Settings struct {
	ConnectionName string `cfg:"connection" default:"default"`
	connection     *connection.Settings

	Topic   string `cfg:"topic" validate:"required"`
	GroupID string `cfg:"group_id"`
	// FQTopic is the fully-qualified topic name (with prefix).
	FQTopic string
	// FQGroupID is the fully-qualified group id (with prefix).
	FQGroupID    string
	BatchSize    int           `cfg:"batch_size" default:"1"`
	BatchTimeout time.Duration `cfg:"idle_timeout" default:"1s"`
	StartOffset  string        `cfg:"start_offset" default:"last" validate:"oneof=first last"`

	Healthcheck health.HealthCheckSettings `cfg:"healthcheck"`
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
	config.UnmarshalKey(key, settings)

	appID, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get app ID from config: %w", err)
	}
	settings.connection = connection.ParseSettings(config, settings.ConnectionName)
	settings.FQGroupID = kafka.FQGroupId(config, appID, settings.GroupID)
	settings.FQTopic = kafka.FQTopicName(config, appID, settings.Topic)

	return settings, nil
}
