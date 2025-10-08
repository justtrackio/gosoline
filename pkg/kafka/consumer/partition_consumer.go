package consumer

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type PartitionConsumer struct {
	logger         log.Logger
	topic          string
	partition      int32
	messageHandler KafkaMessageHandler
	kafkaClient    *kgo.Client
	assignedBatch  chan []*kgo.Record
	stop           chan struct{}
	done           chan struct{}
}

func NewPartitionConsumer(logger log.Logger, topic string, partition int32, messageHandler KafkaMessageHandler, kafkaClient *kgo.Client) *PartitionConsumer {
	return &PartitionConsumer{
		logger:         logger,
		topic:          topic,
		partition:      partition,
		messageHandler: messageHandler,
		kafkaClient:    kafkaClient,
		assignedBatch:  make(chan []*kgo.Record),
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}
}

func (c PartitionConsumer) Consume(ctx context.Context) error {
	defer c.logger.Debug(ctx, "done consuming partition %d of topic %s", c.partition, c.topic)
	defer close(c.done)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stop:
			return nil
		case records := <-c.assignedBatch:
			if len(records) == 0 {
				continue
			}

			c.messageHandler.Handle(records)

			// we immediately commit so we can continue processing the next records and leave retry handling to some retry input like an SQS queue
			err := c.kafkaClient.CommitRecords(ctx, records...)
			if err != nil {
				offset := records[len(records)-1].Offset + 1

				return fmt.Errorf("failed to commit offset %d for partition %d of topic %s: %w", offset, c.partition, c.topic, err)
			}
		}
	}
}

func (c PartitionConsumer) Stop() {
	close(c.stop)
}
