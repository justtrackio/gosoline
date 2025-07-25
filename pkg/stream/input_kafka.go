package stream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type kafkaInput struct {
	logger           log.Logger
	healthCheckTimer clock.HealthCheckTimer
	polling          atomic.Bool
	partitionManager kafkaConsumer.PartitionManager
	reader           kafkaConsumer.Reader
	maxPollRecords   int
	data             chan *Message
}

func NewKafkaInput(ctx context.Context, config cfg.Config, logger log.Logger, settings kafkaConsumer.Settings) (Input, error) {
	data := make(chan *Message)
	messageHandler := NewKafkaMessageHandler(data)
	partitionManager := kafkaConsumer.NewPartitionManager(logger, messageHandler)

	opts := []kgo.Opt{
		kgo.OnPartitionsAssigned(partitionManager.OnPartitionsAssigned),
		kgo.OnPartitionsRevoked(partitionManager.OnPartitionsLostOrRevoked),
		kgo.OnPartitionsLost(partitionManager.OnPartitionsLostOrRevoked),
	}

	reader, err := kafkaConsumer.NewReader(ctx, config, logger, settings, opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka reader: %w", err)
	}

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.Healthcheck.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	return NewKafkaInputWithInterfaces(logger, healthCheckTimer, *partitionManager, reader, settings.MaxPollRecords, data)
}

func NewKafkaInputWithInterfaces(
	logger log.Logger,
	healthCheckTimer clock.HealthCheckTimer,
	partitionManager kafkaConsumer.PartitionManager,
	reader kafkaConsumer.Reader,
	maxPollRecords int,
	data chan *Message,
) (Input, error) {
	return &kafkaInput{
		logger:           logger,
		healthCheckTimer: healthCheckTimer,
		partitionManager: partitionManager,
		reader:           reader,
		maxPollRecords:   maxPollRecords,
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

		// while we are polling messages, we can't get unhealthy
		// (as this code is outside our control to add code to mark us as healthy)
		i.polling.Store(true)
		fetches := i.reader.PollRecords(ctx, i.maxPollRecords)
		// mark us as healthy as soon as we got records to ensure we stay healthy while we process the records
		// (unless we take too long to send the messages to the i.data channel)
		i.healthCheckTimer.MarkHealthy()
		i.polling.Store(false)

		if fetches.IsClientClosed() {
			return nil
		}
		if errors.Is(fetches.Err0(), context.Canceled) {
			return ctx.Err()
		}

		fetches.EachError(func(topic string, partition int32, err error) {
			var errDataLoss *kgo.ErrDataLoss

			switch {
			case errors.As(err, &errDataLoss):
				// the kafka library declares this error as informational (as it will reset and retry) but worth logging and investigating.
				// so, we log this as a warning.
				i.logger.WithContext(ctx).Warn("%s", err.Error())
			default:
				i.logger.WithContext(ctx).Error("failed to fetch records (topic: %s. partition: %d): %w", topic, partition, err)
			}
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
	return i.healthCheckTimer.IsHealthy() || i.polling.Load()
}
