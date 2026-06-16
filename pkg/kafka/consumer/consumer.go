package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaErrors "github.com/justtrackio/gosoline/pkg/kafka/errors"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	metricNameRecordsConsumed = "RecordsConsumed"
	metricNamePollCount       = "PollCount"
	metricNamePollDuration    = "PollDuration"
)

// ReaderFactory creates a Reader using the run context and the partition manager.
// It is called at the start of Run to create the kgo client with the correct lifecycle context.
type ReaderFactory func(ctx context.Context, partitionManager *PartitionManager) (Reader, error)

//go:generate go run github.com/vektra/mockery/v2 --name Consumer
type Consumer interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context)
	IsHealthy() bool
}

type consumer struct {
	logger               log.Logger
	clock                clock.Clock
	healthCheckTimer     clock.HealthCheckTimer
	pollingOrRebalancing atomic.Bool
	partitionManager     *PartitionManager
	readerFactory        ReaderFactory
	reader               Reader
	executor             exec.Executor
	settings             Settings
	stopped              chan struct{}
	name                 string
	metricWriter         metric.Writer
	fullTopicName        string
	isReadOnly           bool
}

func NewConsumer(ctx context.Context, config cfg.Config, logger log.Logger, handler KafkaMessageHandler, settings Settings, name string) (Consumer, error) {
	conn, err := connection.ParseSettings(config, settings.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kafka connection settings for connection name %q: %w", settings.Connection, err)
	}

	fullTopicName, err := kafka.BuildFullTopicName(config, settings.ToIdentity(), settings.TopicId)
	if err != nil {
		return nil, fmt.Errorf("failed to build full kafka topic name: %w", err)
	}

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.Healthcheck.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	defaults := getConsumerDefaultMetrics(name, fullTopicName)
	metricWriter := metric.NewWriter(defaults...)

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManagerConsumer(name, fullTopicName, conn.Brokers)); err != nil {
		return nil, fmt.Errorf("failed to add kafka consumer lifecycle manager: %w", err)
	}

	readerFactory := func(ctx context.Context, partitionManager *PartitionManager) (Reader, error) {
		return NewReader(ctx, config, logger, settings, partitionManager, conn.IsReadOnly, name)
	}

	return &consumer{
		logger:           logger,
		clock:            clock.Provider,
		healthCheckTimer: healthCheckTimer,
		partitionManager: NewPartitionManager(logger, clock.Provider, metricWriter, handler, name),
		readerFactory:    readerFactory,
		settings:         settings,
		stopped:          make(chan struct{}),
		name:             name,
		metricWriter:     metricWriter,
		fullTopicName:    fullTopicName,
		isReadOnly:       conn.IsReadOnly,
	}, nil
}

func NewConsumerWithInterfaces(
	logger log.Logger,
	clk clock.Clock,
	healthCheckTimer clock.HealthCheckTimer,
	handler KafkaMessageHandler,
	readerFactory ReaderFactory,
	settings Settings,
	metricWriter metric.Writer,
	fullTopicName string,
	isReadOnly bool,
	name string,
) Consumer {
	return &consumer{
		logger:           logger,
		clock:            clk,
		healthCheckTimer: healthCheckTimer,
		partitionManager: NewPartitionManager(logger, clk, metricWriter, handler, name),
		readerFactory:    readerFactory,
		settings:         settings,
		stopped:          make(chan struct{}),
		name:             name,
		metricWriter:     metricWriter,
		fullTopicName:    fullTopicName,
		isReadOnly:       isReadOnly,
	}
}

func newExecutor(logger log.Logger, reader Reader, settings *Settings) exec.Executor {
	res := &exec.ExecutableResource{
		Type: "kafka",
		Name: settings.TopicId,
	}

	return exec.NewBackoffExecutor(
		logger,
		res,
		&settings.Backoff,
		[]exec.ErrorChecker{
			CheckKafkaUnknownTopicError,
			CheckKafkaRetryableError(reader),
		},
		exec.WithElapsedTimeTrackerFactory(func() exec.ElapsedTimeTracker {
			return exec.NewErrorTriggeredElapsedTimeTracker()
		}),
	)
}

// CheckKafkaUnknownTopicError is an exec.ErrorChecker that fails fast when the consumer is configured
// for a topic that does not exist. franz-go surfaces these as UNKNOWN_TOPIC_OR_PARTITION (or
// UNKNOWN_TOPIC_ID) once KeepRetryableFetchErrors is enabled.
func CheckKafkaUnknownTopicError(_ any, err error) exec.ErrorType {
	if isUnknownTopicError(err) {
		return exec.ErrorTypePermanent
	}

	return exec.ErrorTypeUnknown
}

func isUnknownTopicError(err error) bool {
	return errors.Is(err, kerr.UnknownTopicOrPartition) || errors.Is(err, kerr.UnknownTopicID)
}

// CheckKafkaRetryableError is an exec.ErrorChecker that classifies Kafka errors.
// It returns ErrorTypeRetryable for transient errors (connection issues, broker errors)
// and ErrorTypeUnknown for other errors (letting them fail).
func CheckKafkaRetryableError(kafkaReader Reader) func(_ any, err error) exec.ErrorType {
	return func(_ any, err error) exec.ErrorType {
		switch {
		case err == nil:
			return exec.ErrorTypeOk
		case kafkaErrors.IsRetryableKafkaError(err):
			kafkaReader.AllowRebalance()

			return exec.ErrorTypeRetryable
		default:
			return exec.ErrorTypeUnknown
		}
	}
}

func (c *consumer) Run(ctx context.Context) error {
	reader, err := c.readerFactory(ctx, c.partitionManager)
	if err != nil {
		return fmt.Errorf("failed to create kafka reader: %w", err)
	}

	c.reader = reader
	c.executor = newExecutor(c.logger, reader, &c.settings)

	defer c.partitionManager.Stop(ctx)
	defer c.reader.CloseAllowingRebalance()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopped:
			return nil
		default:
		}

		c.pollingOrRebalancing.Store(true)

		start := c.clock.Now()
		result, err := c.executor.Execute(ctx, c.pollRecords)

		c.healthCheckTimer.MarkHealthy()
		c.pollingOrRebalancing.Store(false)

		pollDuration := float64(c.clock.Since(start).Milliseconds())

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

		c.writeMetrics(ctx, pollDuration, countRecords(fetches))
		c.processPartitions(ctx, fetches)

		c.pollingOrRebalancing.Store(true)
		c.reader.AllowRebalance()
		c.healthCheckTimer.MarkHealthy()
		c.pollingOrRebalancing.Store(false)

		if len(fetches) == 0 {
			c.clock.Sleep(c.settings.IdleWaitTime)
		}
	}
}

func countRecords(fetches kgo.Fetches) int {
	var count int

	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		count += len(p.Records)
	})

	return count
}

func (c *consumer) Stop(_ context.Context) {
	close(c.stopped)
}

func (c *consumer) IsHealthy() bool {
	return c.healthCheckTimer.IsHealthy() || c.pollingOrRebalancing.Load()
}

func (c *consumer) pollRecords(ctx context.Context) (any, error) {
	select {
	case <-ctx.Done():
		return kgo.Fetches{}, ctx.Err()
	case <-c.stopped:
		return kgo.Fetches{}, nil
	default:
	}

	//nolint:staticcheck // We pass a nil context to prevent PollRecords from blocking when waiting for new messages.
	fetches := c.reader.PollRecords(nil, c.settings.MaxPollRecords)

	if fetches.IsClientClosed() || exec.IsRequestCanceled(fetches.Err0()) {
		return fetches, nil
	}

	var errs error

	for _, fetchError := range fetches.Errors() {
		var errDataLoss *kgo.ErrDataLoss

		if errors.As(fetchError.Err, &errDataLoss) {
			c.logger.Warn(ctx, "%s", fetchError.Err.Error())

			continue
		}

		// KeepRetryableFetchErrors surfaces missing-topic errors so we can fail fast, but also exposes other
		// retryable per-partition errors that franz-go recovers from internally. Ignore those and keep the
		// records returned alongside them, rather than discarding the fetch (which could drop records).
		if kerr.IsRetriable(fetchError.Err) && !isUnknownTopicError(fetchError.Err) {
			c.logger.Warn(ctx, "ignoring retryable kafka fetch error (topic: %s, partition: %d): %s",
				fetchError.Topic, fetchError.Partition, fetchError.Err.Error())

			continue
		}

		errs = errors.Join(errs, fmt.Errorf("failed to fetch records (topic: %s, partition: %d): %w",
			fetchError.Topic, fetchError.Partition, fetchError.Err))
	}

	return fetches, errs
}

func (c *consumer) processPartitions(ctx context.Context, fetches kgo.Fetches) {
	fetches.EachPartition(func(p kgo.FetchTopicPartition) {
		if c.isReadOnly {
			c.partitionManager.HandleWithoutCommit(p.Records)
		} else {
			c.partitionManager.Handle(ctx, p.Topic, p.Partition, p.Records)
		}

		c.healthCheckTimer.MarkHealthy()
	})
}

func (c *consumer) writeMetrics(ctx context.Context, pollDurationMs float64, recordCount int) {
	dims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: c.name, kafka.DimensionTopic: c.fullTopicName}

	c.metricWriter.Write(ctx, metric.Data{
		metric.NewMetricDatum(metricNamePollCount, dims, 1.0, metric.UnitCount, metric.PriorityHigh),
		metric.NewMetricDatum(metricNamePollDuration, dims, pollDurationMs, metric.UnitMillisecondsAverage, metric.PriorityHigh),
		metric.NewMetricDatum(metricNameRecordsConsumed, dims, float64(recordCount), metric.UnitCount, metric.PriorityHigh),
	})
}

func getConsumerDefaultMetrics(name, topicName string) metric.Data {
	dims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: name, kafka.DimensionTopic: topicName}
	partitionDims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: name, kafka.DimensionTopic: topicName, kafka.DimensionPartition: metric.DimensionDefault}

	return metric.Data{
		{Priority: metric.PriorityHigh, MetricName: metricNameRecordsConsumed, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameRecordsConsumedFailed, Dimensions: partitionDims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNamePollCount, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNamePollDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameProcessDuration, Dimensions: partitionDims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameWaitDuration, Dimensions: partitionDims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameCommitDuration, Dimensions: partitionDims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameCommitFailures, Dimensions: partitionDims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameRebalanceCount, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
	}
}
