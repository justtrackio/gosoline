package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type kafkaOutput struct {
	connection            connection.Settings
	schemaRegistryService schemaRegistry.Service
	writer                kafkaProducer.Writer
}

var (
	_ SizeRestrictedOutput      = &kafkaOutput{}
	_ SchemaRegistryAwareOutput = &kafkaOutput{}
)

func NewKafkaOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *kafkaProducer.Settings) (Output, error) {
	writer, err := kafkaProducer.NewWriter(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka writer: %w", err)
	}

	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	service, err := schemaRegistry.NewService(*conn)
	if err != nil {
		return nil, fmt.Errorf("can not create schema registry service: %w", err)
	}

	return NewKafkaOutputWithInterfaces(*conn, service, writer)
}

func NewKafkaOutputWithInterfaces(
	connection connection.Settings,
	schemaRegistryService schemaRegistry.Service,
	writer kafkaProducer.Writer,
) (Output, error) {
	return &kafkaOutput{
		connection:            connection,
		schemaRegistryService: schemaRegistryService,
		writer:                writer,
	}, nil
}

func (o *kafkaOutput) WriteOne(ctx context.Context, m WritableMessage) error {
	message, err := NewKafkaMessage(m)
	if err != nil {
		return fmt.Errorf("failed to build kafka message: %w", err)
	}

	if o.connection.IsReadOnly {
		return nil
	}

	return o.writer.ProduceSync(ctx, message).FirstErr()
}

func (o *kafkaOutput) Write(ctx context.Context, ms []WritableMessage) error {
	messages, err := NewKafkaMessages(ms)
	if err != nil {
		return fmt.Errorf("failed to build kafka messages: %w", err)
	}

	if o.connection.IsReadOnly {
		return nil
	}

	return o.writer.ProduceSync(ctx, messages...).FirstErr()
}

func (o *kafkaOutput) ProvidesCompression() bool {
	return true
}

func (o *kafkaOutput) SupportsAggregation() bool {
	// when using the schema registry, we can not aggregate.
	// otherwise, we would write something that does not match the schema.
	// unfortunately, we can also not aggregate when not using the schema registry,
	// because the producer daemon starts running as a module before the schema registry can be initialized
	// and therefore the producer daemon can not know if the schema registry is being used.
	return false
}

func (o *kafkaOutput) IsPartitionedOutput() bool {
	// we are not using the partitioned producer daemon aggregator.
	// but the kafka library will partition by the AttributeKafkaKey in the message attributes if it is set.
	return false
}

func (o *kafkaOutput) GetMaxMessageSize() *int {
	return mdl.Box(1_000_000)
}

func (o *kafkaOutput) GetMaxBatchSize() *int {
	// the kafka library will take care of batching.
	// we just need the producer daemon to send messages to the output in the background.
	return mdl.Box(1)
}

func (o *kafkaOutput) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	return InitKafkaSchemaRegistry(ctx, settings, o.schemaRegistryService)
}
