package consumer

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/twmb/franz-go/pkg/kgo"
)

type (
	Balancer    string
	StartOffset string
)

const (
	FirstOffset StartOffset = "first"
	LastOffset  StartOffset = "last"

	CooperativeSticky  Balancer = "cooperative-sticky"
	Sticky             Balancer = "sticky"
	RoundRobinBalancer Balancer = "round-robin"
	Range              Balancer = "range"
)

type Settings struct {
	ConnectionName string `cfg:"connection" default:"default"`
	connection     *connection.Settings

	Topic   string `cfg:"topic" validate:"required"`
	GroupID string `cfg:"group_id"`
	// FQTopic is the fully-qualified topic name (with prefix).
	FQTopic string
	// FQGroupID is the fully-qualified group id (with prefix).
	FQGroupID   string
	StartOffset StartOffset `cfg:"start_offset" default:"last" validate:"oneof=first last"`
	Balancer    Balancer    `cfg:"balancer" default:"cooperative_sticky" validate:"oneof=cooperative_sticky sticky round_robin range"`

	Healthcheck health.HealthCheckSettings `cfg:"healthcheck"`
}

func (s *Settings) Connection() *connection.Settings {
	return s.connection
}

func (s *Settings) WithConnection(conn *connection.Settings) *Settings {
	s.connection = conn

	return s
}

func (s *Settings) GetStartOffset() kgo.Offset {
	startOffset := kgo.NewOffset().AtStart()
	if s.StartOffset == LastOffset {
		startOffset = kgo.NewOffset().AtEnd()
	}

	return startOffset
}

func (s *Settings) GetBalancer() kgo.GroupBalancer {
	balancer := kgo.CooperativeStickyBalancer()
	switch s.Balancer {
	case Sticky:
		balancer = kgo.StickyBalancer()
	case RoundRobinBalancer:
		balancer = kgo.RoundRobinBalancer()
	case Range:
		balancer = kgo.RangeBalancer()
	}

	return balancer
}

func ParseSettings(config cfg.Config, key string) (*Settings, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kafka consumer settings for key %q in ParseSettings: %w", key, err)
	}

	appID, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get app id from config: %w", err)
	}
	conn, err := connection.ParseSettings(config, settings.ConnectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection %q in ParseSettings: %w", settings.ConnectionName, err)
	}
	settings.connection = conn
	fqGroupID, err := kafka.FQGroupId(config, appID, settings.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get FQGroupId in ParseSettings: %w", err)
	}
	settings.FQGroupID = fqGroupID
	fqTopic, err := kafka.FQTopicName(config, appID, settings.Topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get FQTopicName in ParseSettings: %w", err)
	}
	settings.FQTopic = fqTopic

	return settings, nil
}
