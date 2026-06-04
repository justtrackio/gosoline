package producer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/admin"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKeyProducers = "kafka.producers"

type ProducerMetadata struct {
	BootstrapServers []string `json:"bootstrap_servers"`
	Name             string   `json:"name"`
	Topic            string   `json:"topic"`
}

type LifecycleManagerProducer interface {
	reslife.LifeCycleer
	reslife.Creator
	reslife.Registerer
}

type lifecycleManagerProducer struct {
	service        *admin.Service
	name           string
	connectionName string
	topic          string
	brokers        []string
}

var _ LifecycleManagerProducer = &lifecycleManagerProducer{}

func NewLifecycleManagerProducer(name string, connectionName string, topic string, brokers []string) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		conn, err := connection.ParseSettings(config, connectionName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", connectionName, err)
		}

		service, err := admin.NewService(ctx, logger, topic, conn.Brokers)
		if err != nil {
			return nil, fmt.Errorf("could not create kafka producer lifecycle manager: %w", err)
		}

		return &lifecycleManagerProducer{
			service:        service,
			name:           name,
			connectionName: connectionName,
			topic:          topic,
			brokers:        brokers,
		}, nil
	}
}

func (l *lifecycleManagerProducer) GetId() string {
	return fmt.Sprintf("kafka/%s/producer", l.topic)
}

func (l *lifecycleManagerProducer) Create(ctx context.Context) error {
	return l.service.CreateTopic(ctx)
}

func (l *lifecycleManagerProducer) Register(_ context.Context) (key string, metadata any, err error) {
	return MetadataKeyProducers, ProducerMetadata{
		BootstrapServers: l.brokers,
		Name:             l.name,
		Topic:            l.topic,
	}, nil
}
