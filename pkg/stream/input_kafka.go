package stream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type kafkaInput struct {
	logger                log.Logger
	connection            connection.Settings
	healthCheckTimer      clock.HealthCheckTimer
	polling               atomic.Bool
	partitionManager      kafkaConsumer.PartitionManager
	reader                kafkaConsumer.Reader
	schemaRegistryService schemaRegistry.Service
	maxPollRecords        int
	data                  chan *Message
}

var _ SchemaRegistryAwareInput = &kafkaInput{}

func NewKafkaInput(ctx context.Context, config cfg.Config, logger log.Logger, settings kafkaConsumer.Settings) (Input, error) {
	data := make(chan *Message)
	messageHandler := NewKafkaMessageHandler(data)
	partitionManager := kafkaConsumer.NewPartitionManager(logger, messageHandler)

	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	var opts []kgo.Opt

	if !conn.IsReadOnly {
		opts = append(opts, []kgo.Opt{
			kgo.OnPartitionsAssigned(partitionManager.OnPartitionsAssigned),
			kgo.OnPartitionsRevoked(partitionManager.OnPartitionsLostOrRevoked),
			kgo.OnPartitionsLost(partitionManager.OnPartitionsLostOrRevoked),
		}...)
	}

	reader, err := kafkaConsumer.NewReader(ctx, config, logger, settings, conn.IsReadOnly, opts...)
	if err != nil {
		return nil, fmt.Errorf("can not create kafka reader: %w", err)
	}

	schemaRegistryService, err := schemaRegistry.NewService(*conn)
	if err != nil {
		return nil, fmt.Errorf("can not create schema registry service: %w", err)
	}

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.Healthcheck.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	return NewKafkaInputWithInterfaces(logger, *conn, healthCheckTimer, *partitionManager, reader, schemaRegistryService, settings.MaxPollRecords, data)
}

func NewKafkaInputWithInterfaces(
	logger log.Logger,
	connection connection.Settings,
	healthCheckTimer clock.HealthCheckTimer,
	partitionManager kafkaConsumer.PartitionManager,
	reader kafkaConsumer.Reader,
	schemaRegistryService schemaRegistry.Service,
	maxPollRecords int,
	data chan *Message,
) (Input, error) {
	return &kafkaInput{
		logger:                logger,
		connection:            connection,
		healthCheckTimer:      healthCheckTimer,
		partitionManager:      partitionManager,
		reader:                reader,
		schemaRegistryService: schemaRegistryService,
		maxPollRecords:        maxPollRecords,
		data:                  data,
	}, nil
}

func (i *kafkaInput) Run(ctx context.Context) error {
	for {
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

		var err error

		fetchErrors := fetches.Errors()
		for _, fetchError := range fetchErrors {
			var errDataLoss *kgo.ErrDataLoss

			switch {
			case errors.As(fetchError.Err, &errDataLoss):
				// the kafka library declares this error as informational (as it will reset and retry) but worth logging and investigating.
				// so, we just log this as a warning and then try polling again.
				i.logger.Warn(ctx, "%s", fetchError.Err.Error())
			default:
				err = multierror.Append(err, fmt.Errorf("failed to fetch records (topic: %s. partition: %d): %w", fetchError.Topic, fetchError.Partition, fetchError.Err))
			}
		}

		if err != nil {
			return err
		}

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			if i.connection.IsReadOnly {
				i.partitionManager.HandleWithoutCommit(p.Records)

				return
			}

			i.partitionManager.Handle(p.Topic, p.Partition, p.Records)
		})

		i.reader.AllowRebalance()
	}
}

func (i *kafkaInput) Stop(_ context.Context) {
	defer close(i.data)
	i.reader.CloseAllowingRebalance()
}

func (i *kafkaInput) Data() <-chan *Message {
	return i.data
}

func (i *kafkaInput) IsHealthy() bool {
	return i.healthCheckTimer.IsHealthy() || i.polling.Load()
}

func (i *kafkaInput) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	return InitKafkaSchemaRegistry(ctx, settings, i.schemaRegistryService)
}
