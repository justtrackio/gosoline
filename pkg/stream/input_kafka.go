package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
)

type kafkaInput struct {
	consumer              kafkaConsumer.Consumer
	schemaRegistryService schemaRegistry.Service
	schemaRegistryReady   atomic.Bool
	channel               chan *Message
}

var (
	_ SchemaRegistryAwareInput = &kafkaInput{}
	_ AcknowledgeableInput     = &kafkaInput{}
)

func NewKafkaInput(ctx context.Context, config cfg.Config, logger log.Logger, settings kafkaConsumer.Settings, name string) (Input, error) {
	channel := make(chan *Message)
	handler := NewKafkaMessageHandler(channel)

	consumer, err := kafkaConsumer.NewConsumer(ctx, config, logger, handler, settings, name)
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

	return NewKafkaInputWithInterfaces(consumer, schemaRegistryService, channel), nil
}

func NewKafkaInputWithInterfaces(consumer kafkaConsumer.Consumer, schemaRegistryService schemaRegistry.Service, channel chan *Message) Input {
	inp := &kafkaInput{
		consumer:              consumer,
		schemaRegistryService: schemaRegistryService,
		channel:               channel,
	}

	// initialize ready as we don't know yet if we will use the schema registry
	// the schema is only parsed later from the implementing consumer
	inp.schemaRegistryReady.Store(true)

	return inp
}

func (i *kafkaInput) Run(ctx context.Context) error {
	return i.consumer.Run(ctx)
}

func (i *kafkaInput) Stop(ctx context.Context) {
	i.consumer.Stop(ctx)
}

func (i *kafkaInput) Data() <-chan *Message {
	return i.channel
}

// Ack marks the message as processed so that the offsets of the batch it belongs to can be committed once all of its
// messages are done. Kafka commits offsets sequentially, so a negative acknowledgement records a failure but does not
// prevent the batch from being committed.
func (i *kafkaInput) Ack(_ context.Context, msg *Message, ack bool) error {
	if msg == nil {
		return nil
	}

	completion, ok := msg.metaData[metaDataKafkaBatchCompletion].(*kafkaBatchCompletion)
	if !ok {
		return nil
	}

	completion.complete(ack)

	return nil
}

func (i *kafkaInput) AckBatch(ctx context.Context, msgs []*Message, acks []bool) error {
	if len(msgs) != len(acks) {
		return fmt.Errorf("the number of messages (%d) does not match the number of acknowledgements (%d)", len(msgs), len(acks))
	}

	for idx, msg := range msgs {
		if err := i.Ack(ctx, msg, acks[idx]); err != nil {
			return err
		}
	}

	return nil
}

func (i *kafkaInput) IsHealthy() bool {
	return i.consumer.IsHealthy() && i.schemaRegistryReady.Load()
}

func (i *kafkaInput) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	i.schemaRegistryReady.Store(false)

	encoder, err := InitKafkaSchemaRegistry(ctx, settings, i.schemaRegistryService)
	if err != nil {
		return nil, err
	}

	i.schemaRegistryReady.Store(true)

	return encoder, nil
}
