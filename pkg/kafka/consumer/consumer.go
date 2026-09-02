package consumer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kafka"
	"github.com/justtrackio/gosoline/pkg/kafka/connection"
	kafkaErrors "github.com/justtrackio/gosoline/pkg/kafka/errors"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sync/semaphore"
)

const (
	metricNameCommitDuration        = "CommitDuration"
	metricNamePollCount             = "PollCount"
	metricNamePollDuration          = "PollDuration"
	metricNameProcessDuration       = "ProcessDuration"
	metricNameRecordsConsumed       = "RecordsConsumed"
	metricNameRecordsConsumedFailed = "RecordsConsumedFailed"
	metricNameSleepDuration         = "SleepDuration"
)

// healthCheckKeepAliveFallback is the keep alive interval used while waiting out a consume delay if no health check
// timeout is configured. Callers going through the config always have one (health.HealthCheckSettings defaults to 5m),
// but a Settings literal may leave it at zero, and a non positive ticker interval is not valid.
const healthCheckKeepAliveFallback = time.Second

// ReaderFactory creates a Reader using the run context and the partition manager.
// It is called at the start of Run to create the kgo client with the correct lifecycle context.
type ReaderFactory func(ctx context.Context, partitionManager *PartitionManager) (Reader, error)

//go:generate go run github.com/vektra/mockery/v2 --name Consumer
type Consumer interface {
	Run(ctx context.Context, process func(ctx context.Context, record *kgo.Record) bool) error
	Stop(ctx context.Context)
	IsHealthy() bool
}

type consumer struct {
	logger               log.Logger
	clock                clock.Clock
	metricWriter         metric.Writer
	healthCheckTimer     clock.HealthCheckTimer
	pollingOrRebalancing atomic.Bool
	started              atomic.Bool
	partitionManager     *PartitionManager
	readerFactory        ReaderFactory
	reader               Reader
	executorPolling      exec.Executor
	executorCommitting   exec.Executor
	settings             *Settings
	shutdownCtx          context.Context
	shutdown             context.CancelFunc
	name                 string
	fullTopicName        string
}

func NewConsumer(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings, name string) (Consumer, error) {
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
		return NewReader(ctx, config, logger, settings, partitionManager, name)
	}

	return NewConsumerWithInterfaces(logger, clock.Provider, healthCheckTimer, readerFactory, settings, metricWriter, fullTopicName, name), nil
}

func NewConsumerWithInterfaces(
	logger log.Logger,
	clk clock.Clock,
	healthCheckTimer clock.HealthCheckTimer,
	readerFactory ReaderFactory,
	settings *Settings,
	metricWriter metric.Writer,
	fullTopicName string,
	name string,
) Consumer {
	shutdownCtx, shutdown := context.WithCancel(context.Background())

	return &consumer{
		logger:           logger,
		clock:            clk,
		healthCheckTimer: healthCheckTimer,
		partitionManager: NewPartitionManager(logger, metricWriter, name),
		readerFactory:    readerFactory,
		settings:         settings,
		name:             name,
		metricWriter:     metricWriter,
		fullTopicName:    fullTopicName,
		shutdownCtx:      shutdownCtx,
		shutdown:         shutdown,
	}
}

func (c *consumer) Run(ctx context.Context, process func(ctx context.Context, record *kgo.Record) bool) error {
	if !c.started.CompareAndSwap(false, true) {
		return fmt.Errorf("can not run a kafka consumer a second time")
	}

	var err error

	runCtx, cancel := context.WithCancel(ctx)
	stopShutdownPropagation := context.AfterFunc(c.shutdownCtx, cancel)
	defer cancel()
	defer stopShutdownPropagation()

	if c.shutdownCtx.Err() != nil {
		// AfterFunc runs asynchronously, so cancel explicitly if shutdown already happened.
		cancel()
	}

	// Only a context which is already cancelled before the consumer started doing any work is reported as an
	// error: the caller asked us to run with a context that can never produce records. Once the poll loop is
	// running, cancellation is the regular shutdown path and returns nil (see runLoop).
	select {
	case <-runCtx.Done():
		if c.shutdownCtx.Err() != nil {
			return nil
		}

		return ctx.Err()
	default:
	}

	readerCtx, cancelReader := context.WithCancel(context.WithoutCancel(runCtx))
	defer cancelReader()

	reader, err := c.readerFactory(readerCtx, c.partitionManager)
	if err != nil {
		return fmt.Errorf("failed to create kafka reader: %w", err)
	}
	c.reader = reader
	defer c.reader.CloseAllowingRebalance()

	c.executorPolling = newExecutorPolling(c.logger, c.allowRebalance, c.settings)
	c.executorCommitting = newExecutorCommitting(c.logger, c.settings)

	return c.runLoop(runCtx, process)
}

// runLoop polls and processes records until runCtx is cancelled, the client is closed or an error occurs.
//
// runCtx is cancelled by Stop as well as by the cancellation of the context Run was called with. Both are
// regular shutdown triggers, so every cancellation aware branch below returns nil rather than the context
// error: which of them observes the cancellation first only depends on where the loop happens to be at that
// moment, and a shutdown must not report an error just because it interrupted an idle wait instead of a poll.
func (c *consumer) runLoop(runCtx context.Context, process func(ctx context.Context, record *kgo.Record) bool) error {
	for {
		if c.isStopping(runCtx) {
			return nil
		}

		fetches, pollDuration, err := c.poll(runCtx)
		if err != nil {
			if exec.IsRequestCanceled(err) {
				return nil
			}

			return err
		}
		if fetches.IsClientClosed() {
			return nil
		}

		processDuration, err := c.process(runCtx, fetches, process)
		if err != nil {
			if exec.IsRequestCanceled(err) {
				return nil
			}

			return err
		}

		c.writeProcessingMetrics(runCtx, pollDuration, processDuration, fetches.NumRecords())
		c.allowRebalance()

		if err := c.waitForRecords(runCtx, fetches); err != nil {
			return err
		}
	}
}

func (c *consumer) poll(ctx context.Context) (kgo.Fetches, float64, error) {
	start := c.clock.Now()
	result, err := c.executorPolling.Execute(ctx, c.pollRecords)
	if err != nil {
		return kgo.Fetches{}, 0, fmt.Errorf("failed to poll records: %w", err)
	}

	fetches, ok := result.(kgo.Fetches)
	if !ok {
		return kgo.Fetches{}, 0, fmt.Errorf("unexpected poll executor result type %T", result)
	}

	return fetches, float64(c.clock.Since(start).Milliseconds()), nil
}

func (c *consumer) process(ctx context.Context, fetches kgo.Fetches, process func(ctx context.Context, record *kgo.Record) bool) (float64, error) {
	start := c.clock.Now()
	if err := c.processRecords(ctx, fetches, process); err != nil {
		return 0, fmt.Errorf("failed to process partitions: %w", err)
	}

	return float64(c.clock.Since(start).Milliseconds()), nil
}

// waitForRecords idles for IdleWaitTime if the last poll returned nothing, so an empty topic does not turn the
// poll loop into a busy loop. It returns early once runCtx is cancelled.
func (c *consumer) waitForRecords(runCtx context.Context, fetches kgo.Fetches) error {
	if fetches.NumRecords() > 0 {
		return nil
	}

	// Use a real timer here because a fake clock would block the poll loop unless tests explicitly advance it.
	timer := time.NewTimer(c.settings.IdleWaitTime)
	defer timer.Stop()

	select {
	case <-runCtx.Done():
		return nil
	case <-timer.C:
		return nil
	}
}

func (c *consumer) Stop(_ context.Context) {
	c.shutdown()
}

func (c *consumer) IsHealthy() bool {
	return c.healthCheckTimer.IsHealthy() || c.pollingOrRebalancing.Load()
}

func (c *consumer) allowRebalance() {
	c.pollingOrRebalancing.Store(true)
	defer c.pollingOrRebalancing.Store(false)

	c.reader.AllowRebalance()
	c.healthCheckTimer.MarkHealthy()
}

func (c *consumer) pollRecords(ctx context.Context) (any, error) {
	select {
	case <-ctx.Done():
		return kgo.Fetches{}, ctx.Err()
	default:
	}

	c.healthCheckTimer.MarkHealthy()

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
		if kafkaErrors.IsRetryableKafkaError(fetchError.Err) && !isUnknownTopicError(fetchError.Err) {
			c.logger.Warn(ctx, "ignoring retryable kafka fetch error (topic: %s, partition: %d): %s",
				fetchError.Topic, fetchError.Partition, fetchError.Err.Error())

			continue
		}

		errs = errors.Join(errs, fmt.Errorf("failed to fetch records (topic: %s, partition: %d): %w",
			fetchError.Topic, fetchError.Partition, fetchError.Err))
	}

	return fetches, errs
}

func (c *consumer) processRecords(ctx context.Context, fetches kgo.Fetches, process func(ctx context.Context, record *kgo.Record) bool) error {
	if fetches.NumRecords() == 0 {
		return nil
	}

	workUnits := make([][]*kgo.Record, 0, fetches.NumRecords())

	switch c.settings.ProcessingMode {
	case "", ProcessingModeUnordered:
		// fetches.Records() appends the records of each partition in the order franz-go received them,
		// so the resulting slice is ascending by offset within every topic partition. processRecordWorkUnits
		// relies on that to keep the committed offsets free of gaps.
		for _, record := range fetches.Records() {
			workUnits = append(workUnits, []*kgo.Record{record})
		}

		return c.processRecordWorkUnits(ctx, workUnits, process)

	case ProcessingModeOrdered:
		partitionRecords := make(map[topicPartition][]*kgo.Record)
		fetches.EachPartition(func(partition kgo.FetchTopicPartition) {
			key := topicPartition{topic: partition.Topic, partition: partition.Partition}
			partitionRecords[key] = append(partitionRecords[key], partition.Records...)
		})

		for _, records := range partitionRecords {
			workUnits = append(workUnits, records)
		}

		return c.processRecordWorkUnits(ctx, workUnits, process)

	default:
		return fmt.Errorf("unknown processing mode %s", c.settings.ProcessingMode)
	}
}

// processRecordWorkUnits processes the given work units with at most runnerCount of them in flight at the
// same time and commits every record that was handed to the processing callback.
//
// Committing is only safe because of two invariants which have to be preserved by any change to this
// function:
//
//  1. Work units are admitted strictly in order and cancellation is checked before every admission, so the
//     set of admitted work units is always a prefix of workUnits.
//  2. Every admitted work unit processes and records at least its first record and stops only at a record
//     boundary afterwards. This is why processRecordWorkUnit skips the cancellation check for the first
//     record and why callbacks run on a drain context instead of ctx.
//
// Together they guarantee that the recorded records form a gap free prefix per topic partition. This matters
// because CommitRecords commits the highest offset per partition: committing a set with a hole in it would
// mark the records below that hole as consumed even though they were never processed. Invariant 1 also
// relies on fetches.Records() returning the records of a partition in ascending offset order (see
// processRecords).
func (c *consumer) processRecordWorkUnits(ctx context.Context, workUnits [][]*kgo.Record, process func(ctx context.Context, record *kgo.Record) bool) error {
	sem := semaphore.NewWeighted(int64(c.runnerCount()))
	result := &recordBatchResult{}

	// In flight processing must not be cut short by cancellation, otherwise a record would be committed
	// without having been processed. Callbacks are drained instead and only cancelled once the caller's
	// processing deadline expired.
	processingCtx, stopDraining := c.drainContext(ctx)
	defer stopDraining()

	cfn := coffin.New()
	cfn.Go(func() error {
		for _, records := range workUnits {
			if c.isStopping(ctx) {
				return nil
			}

			if err := sem.Acquire(ctx, 1); err != nil {
				if exec.IsRequestCanceled(err) {
					return nil
				}

				return fmt.Errorf("can not acquire semaphore: %w", err)
			}

			// Waiting for a free runner is where we spend most of the time, so cancellation is most likely
			// to happen while we were blocked above. Re-check to avoid admitting one more work unit than
			// necessary during a shutdown.
			if c.isStopping(ctx) {
				sem.Release(1)

				return nil
			}

			cfn.Go(func() error {
				defer sem.Release(1)

				return c.processRecordWorkUnit(ctx, processingCtx, records, process, result)
			})
		}

		return nil
	})

	if err := cfn.Wait(); err != nil {
		return fmt.Errorf("unexpected error during record processing: %w", err)
	}

	c.logRecordsAffectedByExpiredProcessingDeadline(ctx, workUnits, result)

	c.writeMetric(ctx, metricNameRecordsConsumedFailed, float64(result.failedCount()), metric.UnitCount)

	if err := c.commitRecords(ctx, result.records()); err != nil {
		return fmt.Errorf("failed to commit records: %w", err)
	}

	return nil
}

// logRecordsAffectedByExpiredProcessingDeadline reports what an expired processing deadline cost, which the drain
// error itself can not: it runs without access to the batch and thus can not tell how much work was left undone.
//
// It is only ever called once all work units finished, so a done drain context can not be confused with the
// cancellation from the deferred stopDraining, which happens strictly later. Note that the records which were not
// processed were skipped at admission by isStopping rather than cut short by the deadline, as every admitted work unit
// runs to a record boundary.
func (c *consumer) logRecordsAffectedByExpiredProcessingDeadline(ctx context.Context, workUnits [][]*kgo.Record, result *recordBatchResult) {
	drainCtx, ok := exec.DrainContextFrom(ctx)
	if !ok || drainCtx.Err() == nil {
		return
	}

	processedCount, totalCount := len(result.records()), 0
	for _, records := range workUnits {
		totalCount += len(records)
	}

	c.logger.Error(ctx, "processing deadline expired while processing records: %d of %d records were handed to the callback (%d of them failed) and are committed regardless, the remaining %d were skipped and are consumed again",
		processedCount, totalCount, result.failedCount(), totalCount-processedCount)
}

func (c *consumer) processRecordWorkUnit(ctx, processingCtx context.Context, records []*kgo.Record, process func(ctx context.Context, record *kgo.Record) bool, result *recordBatchResult) error {
	for i, record := range records {
		// A cancelled delay cuts the wait short rather than skipping the record: for the first record of the
		// work unit skipping is not an option (see below), and for the following ones the check right after
		// stops the unit anyway.
		c.delayConsume(ctx, record)

		// The first record is processed unconditionally. The work unit was already admitted, so the
		// concurrently running work units may already have recorded higher offsets; skipping it would leave
		// a hole below them. Cancellation is only honoured at the following record boundaries.
		if i > 0 && c.isStopping(ctx) {
			return nil
		}

		if ok := c.processWithRecovery(processingCtx, record, process); !ok {
			result.markFailed()
		}

		result.addProcessed(record)

		c.healthCheckTimer.MarkHealthy()
	}

	return nil
}

// delayConsume waits until the given record is at least ConsumeDelay old, so downstream consumers only see data after a
// deliberate lag. It returns immediately if the delay is disabled or the record is already old enough.
//
// The age is measured against the record's kafka timestamp, which the broker only assigns for topics configured with
// message.timestamp.type=LogAppendTime. Otherwise it comes from the producer, so it is not trustworthy: a record dated
// into the future must not be held back for longer than the configured delay, and a record without a timestamp appears
// arbitrarily old and is passed on right away.
//
// Waiting blocks the work unit and thereby the poll loop, so the health check has to be kept alive explicitly:
// processRecordWorkUnit only marks progress per processed record, and being killed for waiting exactly as configured
// would be absurd. It also blocks rebalances, which is why ConsumeDelay is validated against RebalanceTimeout.
//
// A cancellation ends the wait early and is not reported: the caller decides what to do with the record, and during a
// shutdown a record released ahead of its delay is preferable to one committed without having been processed.
func (c *consumer) delayConsume(ctx context.Context, record *kgo.Record) {
	if c.settings.ConsumeDelay <= 0 {
		return
	}

	recordAge := c.clock.Since(record.Timestamp)
	if recordAge >= c.settings.ConsumeDelay {
		return
	}

	// clamp to the configured delay, as a record timestamped into the future has a negative age
	durationToSleep := min(c.settings.ConsumeDelay-recordAge, c.settings.ConsumeDelay)

	timer := c.clock.NewTimer(durationToSleep)
	defer timer.Stop()

	c.healthCheckTimer.MarkHealthy()
	ticker := c.clock.NewTicker(c.healthCheckKeepAliveInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.Chan():
			c.writeMetric(ctx, metricNameSleepDuration, float64(durationToSleep.Milliseconds()), metric.UnitMillisecondsAverage)

			return
		case <-ticker.Chan():
			c.healthCheckTimer.MarkHealthy()
		}
	}
}

// healthCheckKeepAliveInterval returns how often the health check is marked healthy while waiting out a consume delay.
// Half the health check timeout leaves room for a delayed tick without turning unhealthy in between.
func (c *consumer) healthCheckKeepAliveInterval() time.Duration {
	if interval := c.settings.Healthcheck.Timeout / 2; interval > 0 {
		return interval
	}

	return healthCheckKeepAliveFallback
}

// drainContext returns the context in flight processing runs with.
//
// The deadline for processing is owned by the caller and communicated via exec.WithDrainContext: the consumer hands
// records to a callback it knows nothing about, so it must not decide on its own for how long that callback may still
// run during a shutdown. As long as the caller's drain context is alive, the returned context keeps the values of ctx
// but survives its cancellation, so a record is never committed without having been processed. Once the drain context
// is done, in flight processing is cancelled and the commit window (see commitRecords) begins.
//
// Without a drain context the caller owns cancellation directly and ctx is returned unchanged: inventing a grace period
// here would silently extend a shutdown the caller expected to be immediate.
func (c *consumer) drainContext(ctx context.Context) (context.Context, context.CancelFunc) {
	drainCtx, ok := exec.DrainContextFrom(ctx)
	if !ok {
		return ctx, func() {}
	}

	processingCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	stopDrainPropagation := context.AfterFunc(drainCtx, func() {
		// Logged at error level because the cancelled records are committed nonetheless (see
		// processRecordWorkUnits), so their processing is not retried. This is also the only signal left
		// if a callback never returns at all: processRecordWorkUnits keeps waiting for it and never
		// reaches the summary below.
		c.logger.Error(processingCtx, "processing deadline expired, cancelling in-flight record processing")
		cancel()
	})

	return processingCtx, func() {
		stopDrainPropagation()
		cancel()
	}
}

// isStopping reports whether the consumer is shutting down and should neither poll nor admit more work.
// c.shutdownCtx is checked in addition to ctx because Stop propagates to ctx via context.AfterFunc, which runs
// asynchronously and may therefore still lag behind the actual Stop call.
func (c *consumer) isStopping(ctx context.Context) bool {
	return ctx.Err() != nil || c.shutdownCtx.Err() != nil
}

func (c *consumer) processWithRecovery(ctx context.Context, record *kgo.Record, process func(ctx context.Context, record *kgo.Record) bool) (ok bool) {
	defer func() {
		if err := coffin.ResolveRecovery(recover()); err != nil {
			c.logger.Error(ctx, "panic while processing kafka record: %s", err)
			ok = false
		}
	}()

	return process(ctx, record)
}

// commitRecords commits the given records and grants the commit its own GraceTime window. The window is created here
// instead of upfront so it is measured from the moment processing finished: WithDelayedCancelContext only starts its
// timer once the parent context is done, which for an already cancelled ctx is the moment of this call. The commit
// therefore always gets its full window, no matter how long the caller's processing deadline was.
func (c *consumer) commitRecords(ctx context.Context, records []*kgo.Record) error {
	delayedCtx, stop := exec.WithDelayedCancelContext(ctx, c.settings.GraceTime)
	defer stop()

	start := c.clock.Now()

	_, err := c.executorCommitting.Execute(delayedCtx, func(ctx context.Context) (any, error) {
		return nil, c.reader.CommitRecords(ctx, records...)
	})
	if err != nil {
		// runLoop discards a cancelled commit to keep the shutdown itself clean, so this is the only place
		// where the consequence is still visible: the records were processed but their offsets never made it
		// to the broker, which means they get redelivered. Never let that pass silently. A cancelled commit of
		// an empty batch loses nothing and is not worth a warning.
		if exec.IsRequestCanceled(err) && len(records) > 0 {
			c.logger.Error(ctx, "commit grace time of %v expired, %d records could not be committed and will be consumed again: %s", c.settings.GraceTime, len(records), err.Error())
		}

		return fmt.Errorf("failed to commit records: %w", err)
	}

	commitDuration := float64(c.clock.Since(start).Milliseconds())
	c.writeMetric(ctx, metricNameCommitDuration, commitDuration, metric.UnitMillisecondsAverage)

	return nil
}

func (c *consumer) runnerCount() int {
	if c.settings.RunnerCount <= 0 {
		return 1
	}

	return c.settings.RunnerCount
}

func (c *consumer) writeMetric(ctx context.Context, metricName string, value float64, unit metric.StandardUnit) {
	dims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: c.name, kafka.DimensionTopic: c.fullTopicName}

	c.metricWriter.Write(ctx, metric.Data{
		metric.NewMetricDatum(metricName, dims, value, unit, metric.PriorityHigh),
	})
}

func (c *consumer) writeProcessingMetrics(ctx context.Context, pollDurationMs float64, processDuration float64, recordCount int) {
	dims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: c.name, kafka.DimensionTopic: c.fullTopicName}

	c.metricWriter.Write(ctx, metric.Data{
		metric.NewMetricDatum(metricNamePollCount, dims, 1.0, metric.UnitCount, metric.PriorityHigh),
		metric.NewMetricDatum(metricNamePollDuration, dims, pollDurationMs, metric.UnitMillisecondsAverage, metric.PriorityHigh),
		metric.NewMetricDatum(metricNameProcessDuration, dims, processDuration, metric.UnitMillisecondsAverage, metric.PriorityHigh),
		metric.NewMetricDatum(metricNameRecordsConsumed, dims, float64(recordCount), metric.UnitCount, metric.PriorityHigh),
	})
}

func getConsumerDefaultMetrics(name, topicName string) metric.Data {
	dims := metric.Dimensions{kafka.DimensionClientType: kafka.DimensionConsumer, kafka.DimensionClient: name, kafka.DimensionTopic: topicName}

	return metric.Data{
		{Priority: metric.PriorityHigh, MetricName: metricNameRecordsConsumed, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameRecordsConsumedFailed, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNamePollCount, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNamePollDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameProcessDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameCommitDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameRebalanceCount, Dimensions: dims, Unit: metric.UnitCount, Kind: metric.KindDefault},
		{Priority: metric.PriorityHigh, MetricName: metricNameSleepDuration, Dimensions: dims, Unit: metric.UnitMillisecondsAverage, Kind: metric.KindDefault},
	}
}

type topicPartition struct {
	topic     string
	partition int32
}

// recordBatchResult collects the outcome of processing a batch of record work units. Records are added in
// completion order; CommitRecords only looks at the highest offset per partition, so the order is irrelevant
// as long as the set of added records has no gaps (see processRecordWorkUnits).
type recordBatchResult struct {
	lck       sync.Mutex
	processed []*kgo.Record
	failed    atomic.Int32
}

// addProcessed records that the given record was handed to the processing callback and may be committed.
func (r *recordBatchResult) addProcessed(record *kgo.Record) {
	r.lck.Lock()
	defer r.lck.Unlock()

	r.processed = append(r.processed, record)
}

// markFailed counts a record whose processing callback reported a failure.
func (r *recordBatchResult) markFailed() {
	r.failed.Add(1)
}

// records returns all records which were handed to the processing callback.
func (r *recordBatchResult) records() []*kgo.Record {
	r.lck.Lock()
	defer r.lck.Unlock()

	return r.processed
}

// failedCount returns the number of records whose processing callback reported a failure.
func (r *recordBatchResult) failedCount() int32 {
	return r.failed.Load()
}
