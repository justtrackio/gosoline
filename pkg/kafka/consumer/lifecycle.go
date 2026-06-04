package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/admin"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKeyConsumers = "kafka.consumers"

type ConsumerMetadata struct {
	BootstrapServers []string `json:"bootstrap_servers"`
	Name             string   `json:"name"`
	Topic            string   `json:"topic"`
}

type LifecycleManagerConsumer interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Registerer
}

type lifecycleManagerConsumer struct {
	service        *admin.Service
	name           string
	connectionName string
	topic          string
	brokers        []string
}

var _ LifecycleManagerConsumer = &lifecycleManagerConsumer{}

func NewLifecycleManagerConsumer(name string, connectionName string, topic string, brokers []string) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		conn, err := connection.ParseSettings(config, connectionName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", connectionName, err)
		}

		service, err := admin.NewService(ctx, logger, topic, conn.Brokers)
		if err != nil {
			return nil, fmt.Errorf("could not create kafka consumer lifecycle manager: %w", err)
		}

		return &lifecycleManagerConsumer{
			service:        service,
			name:           name,
			connectionName: connectionName,
			topic:          topic,
			brokers:        brokers,
		}, nil
	}
}

func (l *lifecycleManagerConsumer) GetId() string {
	return fmt.Sprintf("kafka/%s/consumer", l.topic)
}

func (l *lifecycleManagerConsumer) Create(ctx context.Context) error {
	return l.service.CreateTopic(ctx)
}

func (l *lifecycleManagerConsumer) Register(_ context.Context) (key string, metadata any, err error) {
	return MetadataKeyConsumers, ConsumerMetadata{
		BootstrapServers: l.brokers,
		Name:             l.name,
		Topic:            l.topic,
	}, nil
}
