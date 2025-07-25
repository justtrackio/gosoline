package stream

import (
	"context"
	"errors"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type kafkaInput struct {
	logger           log.Logger
	partitionManager kafkaConsumer.PartitionManager
	reader           kafkaConsumer.Reader
	data             chan *Message
}

func NewKafkaInput(ctx context.Context, config cfg.Config, logger log.Logger, key string) (Input, error) {
	data := make(chan *Message)
	messageHandler := NewKafkaMessageHandler(data)
	partitionManager := kafkaConsumer.NewPartitionManager(logger, messageHandler)

	opts := []kgo.Opt{
		kgo.OnPartitionsAssigned(partitionManager.OnPartitionsAssigned),
		kgo.OnPartitionsRevoked(partitionManager.OnPartitionsLostOrRevoked),
		kgo.OnPartitionsLost(partitionManager.OnPartitionsLostOrRevoked),
	}

	reader, err := kafkaConsumer.NewReader(ctx, config, logger, key, opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka reader: %w", err)
	}

	return NewKafkaInputWithInterfaces(logger, *partitionManager, reader, data)
}

func NewKafkaInputWithInterfaces(logger log.Logger, partitionManager kafkaConsumer.PartitionManager, reader kafkaConsumer.Reader, data chan *Message) (Input, error) {
	return &kafkaInput{
		logger:           logger,
		partitionManager: partitionManager,
		reader:           reader,
		data:             data,
	}, nil
}

func (i *kafkaInput) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fetches := i.reader.PollRecords(ctx, 100) // todo: make configurable
		if fetches.IsClientClosed() {
			return nil
		}
		if errors.Is(fetches.Err0(), context.Canceled) {
			return ctx.Err()
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			// todo: proper error handling
			i.logger.WithContext(ctx).Error("failed to fetch record (topic: %s. partition: %d): %w", topic, partition, err)
		})

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			i.partitionManager.AssignRecords(p.Topic, p.Partition, p.Records)
		})

		i.reader.AllowRebalance()
	}
}

func (i *kafkaInput) Stop() {
	defer close(i.data)
	i.reader.CloseAllowingRebalance()
}

func (i *kafkaInput) Data() <-chan *Message {
	return i.data
}

func (i *kafkaInput) IsHealthy() bool {
	return true // todo: health check
}
