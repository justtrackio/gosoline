package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type kafkaInput struct {
	consumer              kafkaConsumer.Consumer
	schemaRegistryService schemaRegistry.Service
}

var _ SchemaRegistryAwareInput = &kafkaInput{}

func NewKafkaInput(ctx context.Context, config cfg.Config, logger log.Logger, settings *kafkaConsumer.Settings, name string) (Input, error) {
	consumer, err := kafkaConsumer.NewConsumer(ctx, config, logger, settings, name)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka consumer: %w", err)
	}

	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	schemaRegistryService, err := schemaRegistry.NewService(config, logger, settings.Connection, *conn)
	if err != nil {
		return nil, fmt.Errorf("can not create schema registry service: %w", err)
	}

	return NewKafkaInputWithInterfaces(consumer, schemaRegistryService), nil
}

func NewKafkaInputWithInterfaces(consumer kafkaConsumer.Consumer, schemaRegistryService schemaRegistry.Service) Input {
	return &kafkaInput{
		consumer:              consumer,
		schemaRegistryService: schemaRegistryService,
	}
}

func (i *kafkaInput) Run(ctx context.Context, process InputProcess) error {
	return i.consumer.Run(ctx, func(ctx context.Context, record *kgo.Record) bool {
		msg := KafkaToGosoMessage(*record)

		return process(ctx, msg)
	})
}

func (i *kafkaInput) Stop(ctx context.Context) {
	i.consumer.Stop(ctx)
}

func (i *kafkaInput) IsHealthy() bool {
	return i.consumer.IsHealthy()
}

func (i *kafkaInput) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	return InitKafkaSchemaRegistry(ctx, settings, i.schemaRegistryService)
}
