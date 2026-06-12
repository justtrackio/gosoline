package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/admin"
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
	service *admin.Service
	name    string
	topic   string
	brokers []string
}

var _ LifecycleManagerConsumer = &lifecycleManagerConsumer{}

func NewLifecycleManagerConsumer(name string, topic string, brokers []string) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		service, err := admin.NewService(ctx, logger, brokers)
		if err != nil {
			return nil, fmt.Errorf("could not create kafka admin service: %w", err)
		}

		return &lifecycleManagerConsumer{
			service: service,
			name:    name,
			topic:   topic,
			brokers: brokers,
		}, nil
	}
}

func (l *lifecycleManagerConsumer) GetId() string {
	return fmt.Sprintf("kafka/%s/consumer", l.topic)
}

func (l *lifecycleManagerConsumer) Create(ctx context.Context) error {
	return l.service.CreateTopic(ctx, l.topic)
}

func (l *lifecycleManagerConsumer) Register(_ context.Context) (key string, metadata any, err error) {
	return MetadataKeyConsumers, ConsumerMetadata{
		BootstrapServers: l.brokers,
		Name:             l.name,
		Topic:            l.topic,
	}, nil
}
