package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	kafkaProducer "github.com/justtrackio/gosoline/pkg/kafka/producer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type kafkaOutput struct {
	writer kafkaProducer.Writer
}

var _ SizeRestrictedOutput = &kafkaOutput{}

func NewKafkaOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *kafkaProducer.Settings) (Output, error) {
	writer, err := kafkaProducer.NewWriter(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka writer: %w", err)
	}

	return NewKafkaOutputWithInterfaces(writer)
}

func NewKafkaOutputWithInterfaces(writer kafkaProducer.Writer) (Output, error) {
	return &kafkaOutput{writer: writer}, nil
}

func (o *kafkaOutput) WriteOne(ctx context.Context, m WritableMessage) error {
	message, err := NewKafkaMessage(m)
	if err != nil {
		return fmt.Errorf("failed to build kafka message: %w", err)
	}

	return o.writer.ProduceSync(ctx, message).FirstErr()
}

func (o *kafkaOutput) Write(ctx context.Context, ms []WritableMessage) error {
	messages, err := NewKafkaMessages(ms)
	if err != nil {
		return fmt.Errorf("failed to build kafka messages: %w", err)
	}

	return o.writer.ProduceSync(ctx, messages...).FirstErr()
}

func (o *kafkaOutput) ProvidesCompression() bool {
	return true
}

func (o *kafkaOutput) SupportsAggregation() bool {
	return false
}

func (o *kafkaOutput) IsPartitionedOutput() bool {
	// we are not using the partitioned producer daemon aggregator.
	// but the kafka library will partition by the AttributeKafkaKey in the message attributes if it is set.
	return false
}

func (o *kafkaOutput) GetMaxMessageSize() *int {
	return mdl.Box(1000 * 1000)
}

func (o *kafkaOutput) GetMaxBatchSize() *int {
	return mdl.Box(500)
}
