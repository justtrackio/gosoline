package stream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/twmb/franz-go/pkg/kerr"
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
	executor              exec.Executor
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

	res := &exec.ExecutableResource{
		Type: "kafka",
		Name: settings.TopicId,
	}
	executor := exec.NewBackoffExecutor(
		logger,
		res,
		&settings.Backoff,
		[]exec.ErrorChecker{
			CheckKafkaRetryableError(reader),
		},
		exec.WithElapsedTimeTrackerFactory(func() exec.ElapsedTimeTracker {
			return exec.NewErrorTriggeredElapsedTimeTracker()
		}),
	)

	return NewKafkaInputWithInterfaces(logger, *conn, healthCheckTimer, partitionManager, reader, schemaRegistryService, executor, settings.MaxPollRecords, data), nil
}

func NewKafkaInputWithInterfaces(
	logger log.Logger,
	connection connection.Settings,
	healthCheckTimer clock.HealthCheckTimer,
	partitionManager kafkaConsumer.PartitionManager,
	reader kafkaConsumer.Reader,
	schemaRegistryService schemaRegistry.Service,
	executor exec.Executor,
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
		executor:              executor,
		maxPollRecords:        maxPollRecords,
		data:                  data,
	}
}

// CheckKafkaRetryableError is an exec.ErrorChecker that classifies Kafka errors.
// It returns ErrorTypeRetryable for transient errors (connection issues, broker errors)
// and ErrorTypeUnknown for other errors (letting them fail).
func CheckKafkaRetryableError(kafkaReader kafkaConsumer.Reader) func(_ any, err error) exec.ErrorType {
	return func(_ any, err error) exec.ErrorType {
		errType := exec.ErrorTypeUnknown

		defer func() {
			if errType == exec.ErrorTypeRetryable {
				// we should allow a rebalance between executor retries to avoid getting kicked out of the group
				// if we are blocking a rebalance for too long
				kafkaReader.AllowRebalance()
			}
		}()

		switch {
		case err == nil:
			errType = exec.ErrorTypeOk
		case exec.IsConnectionError(err): // Check for network-level connection errors (connection refused, reset, EOF, etc.)
			errType = exec.ErrorTypeRetryable
		case kgo.IsRetryableBrokerErr(err): // Check if franz-go considers this a retryable broker error
			errType = exec.ErrorTypeRetryable
		case kerr.IsRetriable(err): // Check if this is a retryable Kafka protocol error
			errType = exec.ErrorTypeRetryable
		case exec.IsDNSNotFoundError(err): // Check for "no such host" errors. This might be temporary in some environments if a broker restarts.
			errType = exec.ErrorTypeRetryable
		default:
		}

		return errType
	}
}

// pollRecords wraps the PollRecords call with error handling for use with the executor.
func (i *kafkaInput) pollRecords(ctx context.Context) (any, error) {
	fetches := i.reader.PollRecords(ctx, i.maxPollRecords)

	if fetches.IsClientClosed() || exec.IsRequestCanceled(fetches.Err0()) {
		return fetches, nil
	}

	// Collect all non-data-loss errors
	var errs error

	for _, fetchError := range fetches.Errors() {
		var errDataLoss *kgo.ErrDataLoss

		if errors.As(fetchError.Err, &errDataLoss) {
			// the kafka library declares this error as informational (as it will reset and retry) but worth logging and investigating.
			// so, we just log this as a warning and then try polling again.
			i.logger.Warn(ctx, "%s", fetchError.Err.Error())

			continue
		}

		// Collect the error - the executor will decide if it's retryable
		errs = errors.Join(errs, fmt.Errorf("failed to fetch records (topic: %s, partition: %d): %w",
			fetchError.Topic, fetchError.Partition, fetchError.Err))
	}

	return fetches, errs
}

// processPartitions handles the fetched records from each partition.
func (i *kafkaInput) processPartitions(fetches kgo.Fetches) {
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
}

func (i *kafkaInput) Run(ctx context.Context) error {
	defer i.partitionManager.Stop(ctx)

	for {
		// while we are polling messages, we can't get unhealthy
		// (as this code is outside our control to add code to mark us as healthy)
		i.pollingOrRebalancing.Store(true)

		// Use the executor to poll records with automatic retry on transient errors
		result, err := i.executor.Execute(ctx, i.pollRecords)

		// mark us as healthy as soon as we got records to ensure we stay healthy while we process the records
		// (unless we take too long to send the messages to the i.data channel)
		i.healthCheckTimer.MarkHealthy()
		i.pollingOrRebalancing.Store(false)

		// Handle executor errors (permanent failures after exhausting retries)
		if err != nil {
			if exec.IsRequestCanceled(err) {
				return nil
			}

			return err
		}

		fetches := result.(kgo.Fetches)

		if fetches.IsClientClosed() || exec.IsRequestCanceled(fetches.Err0()) {
			return nil
		}

		i.processPartitions(fetches)

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
