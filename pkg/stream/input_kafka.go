package stream

import (
	"context"
	"errors"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
)

type KafkaInput struct {
	consumer *kafkaConsumer.Consumer
	data     chan *Message
	pool     coffin.Coffin
}

var _ AcknowledgeableInput = &KafkaInput{}

func NewKafkaInput(ctx context.Context, config cfg.Config, logger log.Logger, key string) (*KafkaInput, error) {
	consumer, err := kafkaConsumer.NewConsumer(ctx, config, logger, key)
	if err != nil {
		return nil, fmt.Errorf("failed to init consumer: %w", err)
	}

	return NewKafkaInputWithInterfaces(consumer)
}

func NewKafkaInputWithInterfaces(consumer *kafkaConsumer.Consumer) (*KafkaInput, error) {
	return &KafkaInput{
		consumer: consumer,
		data:     make(chan *Message, cap(consumer.Data())),
		pool:     coffin.New(),
	}, nil
}

// Run provides a steady stream of messages, returned via Data. Run does not return until Stop is called and thus
// should be called in its own go routine. The only exception to this is if we either fail to produce messages and
// return an error or if the input is depleted (like an InMemoryInput).
//
// Run should only be called once, not all inputs can be resumed.
func (i *KafkaInput) Run(ctx context.Context) error {
	i.pool.GoWithContext(ctx, i.consumer.Run)

	defer close(i.data)

	for msg := range i.consumer.Data() {
		if len(msg.Value) == 6 && msg.Value[0] == 0 && msg.Value[1] == 0 && msg.Value[2] == 0 && msg.Value[3] == 0 {
			// this is a control batch indicating an aborted transactional message.
			// the kafka-go library does not support transactions currently and is not handling this correctly (https://github.com/segmentio/kafka-go/issues/1348).
			continue
		}

		i.data <- KafkaToGosoMessage(msg)
	}

	return ctx.Err()
}

// Stop causes Run to return as fast as possible. Calling Stop is preferable to canceling the context passed to Run
// as it allows Run to shut down cleaner (and might take a bit longer, e.g., to finish processing the current batch
// of messages).
func (i *KafkaInput) Stop() {
	i.pool.Kill(errors.New("asked to stop"))
}

func (i *KafkaInput) IsHealthy() bool {
	return i.consumer.IsHealthy()
}

// Data returns a channel containing the messages produced by this input.
func (i *KafkaInput) Data() <-chan *Message {
	return i.data
}

// Ack acknowledges a message. If possible, prefer calling Ack with a batch as it is more efficient.
func (i *KafkaInput) Ack(ctx context.Context, msg *Message, _ bool) error {
	return i.consumer.Commit(ctx, GosoToKafkaMessage(msg))
}

// AckBatch does the same as calling Ack for every single message would, but it might use fewer calls to an external
// service.
func (i *KafkaInput) AckBatch(ctx context.Context, msgs []*Message, _ []bool) error {
	return i.consumer.Commit(ctx, GosoToKafkaMessages(msgs...)...)
}
