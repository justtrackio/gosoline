package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
)

type kafkaOutput struct {
	logger                log.Logger
	connection            connection.Settings
	schemaRegistryService schemaRegistry.Service
	writer                kafkaProducer.Writer
	maxBatchBytes         int32
	maxBatchSize          int
}

var _ SchemaRegistryAwareOutput = &kafkaOutput{}

func NewKafkaOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *kafkaProducer.Settings) (Output, error) {
	writer, err := kafkaProducer.NewWriter(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka writer: %w", err)
	}

	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	schemaRegistryService, err := schemaRegistry.NewService(*conn)
	if err != nil {
		return nil, fmt.Errorf("can not create schema registry service: %w", err)
	}

	return NewKafkaOutputWithInterfaces(logger, *conn, schemaRegistryService, writer, settings.MaxBatchBytes, settings.MaxBatchSize), nil
}

func NewKafkaOutputWithInterfaces(
	logger log.Logger,
	connection connection.Settings,
	schemaRegistryService schemaRegistry.Service,
	writer kafkaProducer.Writer,
	maxBatchBytes int32,
	maxBatchSize int,
) Output {
	return &kafkaOutput{
		logger:                logger,
		connection:            connection,
		schemaRegistryService: schemaRegistryService,
		writer:                writer,
		maxBatchBytes:         maxBatchBytes,
		maxBatchSize:          maxBatchSize,
	}
}

func (o *kafkaOutput) WriteOne(ctx context.Context, msg WritableMessage) error {
	message, err := NewKafkaMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to build kafka message: %w", err)
	}

	if o.connection.IsReadOnly {
		o.logger.Warn(ctx, "dropping message that was written to a read-only output")

		return nil
	}

	return o.writer.ProduceSync(ctx, message).FirstErr()
}

func (o *kafkaOutput) Write(ctx context.Context, batch []WritableMessage) error {
	messages, err := NewKafkaMessages(batch)
	if err != nil {
		return fmt.Errorf("failed to build kafka messages: %w", err)
	}

	if o.connection.IsReadOnly {
		o.logger.Warn(ctx, "dropping messages that were written to a read-only output")

		return nil
	}

	return o.writer.ProduceSync(ctx, messages...).FirstErr()
}

func (o *kafkaOutput) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	return InitKafkaSchemaRegistry(ctx, settings, o.schemaRegistryService)
}
