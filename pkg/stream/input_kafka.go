package stream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
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
	pollingOrRebalancing  atomic.Bool
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

	reader, err := kafkaConsumer.NewReader(ctx, config, logger, settings, partitionManager, conn.IsReadOnly)
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

	return NewKafkaInputWithInterfaces(logger, *conn, healthCheckTimer, partitionManager, reader, schemaRegistryService, settings.MaxPollRecords, data), nil
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
) Input {
	return &kafkaInput{
		logger:                logger,
		connection:            connection,
		healthCheckTimer:      healthCheckTimer,
		partitionManager:      partitionManager,
		reader:                reader,
		schemaRegistryService: schemaRegistryService,
		maxPollRecords:        maxPollRecords,
		data:                  data,
	}
}

func (i *kafkaInput) Run(ctx context.Context) error {
	defer i.partitionManager.Stop(ctx)

	for {
		// while we are polling messages, we can't get unhealthy
		// (as this code is outside our control to add code to mark us as healthy)
		i.pollingOrRebalancing.Store(true)
		fetches := i.reader.PollRecords(ctx, i.maxPollRecords)
		// mark us as healthy as soon as we got records to ensure we stay healthy while we process the records
		// (unless we take too long to send the messages to the i.data channel)
		i.healthCheckTimer.MarkHealthy()
		i.pollingOrRebalancing.Store(false)

		if fetches.IsClientClosed() || exec.IsRequestCanceled(fetches.Err0()) {
			return nil
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
			} else {
				i.partitionManager.Handle(p.Topic, p.Partition, p.Records)
			}

			// mark us as healthy in case there is backpressure from the consumer, and we take a long time
			// to feed the partitions to the partition manager
			i.healthCheckTimer.MarkHealthy()
		})

		// we can't get unhealthy here as the rebalance may take some time and is out of our control
		i.pollingOrRebalancing.Store(true)
		i.reader.AllowRebalance()
		// mark us as healthy now, in case the rebalance took too long
		i.healthCheckTimer.MarkHealthy()
		i.pollingOrRebalancing.Store(false)
	}
}

func (i *kafkaInput) Stop(_ context.Context) {
	i.reader.CloseAllowingRebalance()
}

func (i *kafkaInput) Data() <-chan *Message {
	return i.data
}

func (i *kafkaInput) IsHealthy() bool {
	return i.healthCheckTimer.IsHealthy() || i.pollingOrRebalancing.Load()
}

func (i *kafkaInput) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	return InitKafkaSchemaRegistry(ctx, settings, i.schemaRegistryService)
}
