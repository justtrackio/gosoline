package consumer

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type PartitionConsumer struct {
	logger         log.Logger
	topic          string
	partition      int32
	messageHandler KafkaMessageHandler
	kafkaClient    *kgo.Client
	fetchBatches   chan []*kgo.Record
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
		fetchBatches:   make(chan []*kgo.Record, 10), // todo: make configurable?
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}
}

func (c PartitionConsumer) Consume(ctx context.Context) {
	defer c.logger.WithContext(ctx).Debug("done consuming partition %d of topic %s", c.partition, c.topic)
	defer close(c.done)

	for {
		select {
		case <-c.stop:
			return
		case records := <-c.fetchBatches:
			for _, record := range records {
				if record == nil {
					continue
				}

				c.messageHandler.Handle(*record)
			}

			err := c.kafkaClient.CommitRecords(ctx, records...)
			if err != nil {
				c.logger.WithContext(ctx).WithFields(map[string]any{
					"topic":     c.topic,
					"partition": c.partition,
					"offset":    records[len(records)-1].Offset + 1,
				}).Error("failed to commit offsets to kafka:", err)
			}
		}
	}
}

func (c PartitionConsumer) Stop() {
	close(c.stop)
}
