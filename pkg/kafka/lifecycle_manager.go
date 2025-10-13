package kafka

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/admin"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKeyKafkaTopics = "kafka.topics"

type Metadata struct {
	connectionName string
	Topic          string
}

type lifecycleManager struct {
	service        *admin.Service
	connectionName string
	topic          string
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Registerer
}

var _ LifecycleManager = &lifecycleManager{}

func NewLifecycleManager(connectionName string, topic string) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		conn, err := connection.ParseSettings(config, connectionName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", connectionName, err)
		}

		service, err := admin.NewService(ctx, logger, topic, conn.Brokers)
		if err != nil {
			return nil, fmt.Errorf("could not create kafka lifecycle manager: %w", err)
		}

		return &lifecycleManager{
			service:        service,
			connectionName: connectionName,
			topic:          topic,
		}, nil
	}
}

func (l *lifecycleManager) GetId() string {
	return fmt.Sprintf("kafka/%s", l.topic)
}

func (l *lifecycleManager) Create(ctx context.Context) error {
	return l.service.CreateTopic(ctx)
}

func (l *lifecycleManager) Register(_ context.Context) (key string, metadata any, err error) {
	return MetadataKeyKafkaTopics, Metadata{
		connectionName: l.connectionName,
		Topic:          l.topic,
	}, nil
}
