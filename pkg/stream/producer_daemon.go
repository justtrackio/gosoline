package stream

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	AttributeAggregate      = "goso.aggregate"
	AttributeAggregateCount = "goso.aggregate.count"
	metricNameMessageCount  = "MessageCount"
	metricNameBatchSize     = "BatchSize"
	metricNameAggregateSize = "AggregateSize"
	metricNameIdleDuration  = "IdleDuration"
)

var (
	producerDaemonLock = sync.Mutex{}
	producerDaemons    = map[string]*producerDaemon{}
)

type producerDaemon struct {
	kernel.EssentialBackgroundModule

	name       string
	lck        sync.Mutex
	logger     log.Logger
	metric     metric.Writer
	aggregator ProducerDaemonAggregator
	batcher    ProducerDaemonBatcher
	outCh      OutputChannel
	output     Output
	clock      clock.Clock
	ticker     clock.Ticker
	settings   ProducerDaemonSettings
	stopped    int32
}

func ResetProducerDaemons() {
	producerDaemonLock.Lock()
	defer producerDaemonLock.Unlock()

	producerDaemons = map[string]*producerDaemon{}
}

func ProvideProducerDaemon(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*producerDaemon, error) {
	producerDaemonLock.Lock()
	defer producerDaemonLock.Unlock()

	if _, ok := producerDaemons[name]; ok {
		return producerDaemons[name], nil
	}

	var err error
	producerDaemons[name], err = NewProducerDaemon(ctx, config, logger, name)
	if err != nil {
		return nil, err
	}

	return producerDaemons[name], nil
}

func NewProducerDaemon(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*producerDaemon, error) {
	logger = logger.WithChannel(fmt.Sprintf("producer-daemon-%s", name))
	settings := readProducerSettings(config, name)

	defaultMetrics := getProducerDaemonDefaultMetrics(name)
	metricWriter := metric.NewWriter(defaultMetrics...)

	var err error
	var output Output
	var aggregator ProducerDaemonAggregator

	if output, err = NewConfigurableOutput(ctx, config, logger, settings.Output); err != nil {
		return nil, fmt.Errorf("can not create output for producer daemon %s: %w", name, err)
	}

	if sro, ok := output.(SizeRestrictedOutput); ok {
		if maxBatchSize := sro.GetMaxBatchSize(); maxBatchSize != nil && *maxBatchSize < settings.Daemon.BatchSize {
			settings.Daemon.BatchSize = *maxBatchSize
		}

		if maxMessageSize := sro.GetMaxMessageSize(); maxMessageSize != nil && *maxMessageSize < settings.Daemon.BatchMaxSize {
			settings.Daemon.BatchMaxSize = *maxMessageSize
		}
	}

	if po, ok := output.(PartitionedOutput); ok && po.IsPartitionedOutput() && settings.Daemon.PartitionBucketCount > 1 {
		if aggregator, err = NewProducerDaemonPartitionedAggregator(logger, settings.Daemon, settings.Compression); err != nil {
			return nil, fmt.Errorf("can not create partitioned aggregator for producer daemon %s: %w", name, err)
		}
	} else {
		if aggregator, err = NewProducerDaemonAggregator(settings.Daemon, settings.Compression); err != nil {
			return nil, fmt.Errorf("can not create aggregator for producer daemon %s: %w", name, err)
		}
	}

	return NewProducerDaemonWithInterfaces(logger, metricWriter, aggregator, output, clock.Provider, name, settings.Daemon), nil
}

func NewProducerDaemonWithInterfaces(
	logger log.Logger,
	metric metric.Writer,
	aggregator ProducerDaemonAggregator,
	output Output,
	clock clock.Clock,
	name string,
	settings ProducerDaemonSettings,
) *producerDaemon {
	return &producerDaemon{
		name:       name,
		logger:     logger,
		metric:     metric,
		aggregator: aggregator,
		batcher:    NewProducerDaemonBatcher(settings),
		outCh:      NewOutputChannel(logger, settings.BufferSize),
		output:     output,
		clock:      clock,
		settings:   settings,
	}
}

func (d *producerDaemon) GetStage() int {
	return 512
}

func (d *producerDaemon) Run(kernelCtx context.Context) error {
	defer func() { atomic.StoreInt32(&d.stopped, 1) }()

	// ensure we don't have a race with the code in Write checking if the ticker is nil
	d.lck.Lock()
	d.ticker = d.clock.NewTicker(d.settings.Interval)
	d.lck.Unlock()

	cfn := coffin.New()
	// start the output loops before the ticker look - the output loop can't terminate until
	// we call close, while the ticker can if the context is already canceled
	for i := 0; i < d.settings.RunnerCount; i++ {
		cfn.Gof(d.outputLoop, "panic during running the ticker loop")
	}

	cfn.GoWithContextf(kernelCtx, d.tickerLoop, "panic during running the ticker loop")

	select {
	case <-cfn.Dying():
		if err := d.close(); err != nil {
			return fmt.Errorf("error on close: %w", err)
		}
	case <-kernelCtx.Done():
		if err := d.close(); err != nil {
			return fmt.Errorf("error on close: %w", err)
		}
	}

	return cfn.Wait()
}

func (d *producerDaemon) WriteOne(ctx context.Context, msg WritableMessage) error {
	return d.Write(ctx, []WritableMessage{msg})
}

func (d *producerDaemon) Write(ctx context.Context, batch []WritableMessage) error {
	d.lck.Lock()
	defer d.lck.Unlock()

	if atomic.LoadInt32(&d.stopped) != 0 {
		return fmt.Errorf("can't write messages as the producer daemon %s is not running", d.name)
	}

	var err error
	var aggregated []*Message
	d.writeMetricMessageCount(len(batch))

	if aggregated, err = d.applyAggregation(ctx, batch); err != nil {
		return fmt.Errorf("can not apply aggregation in producer %s: %w", d.name, err)
	}

	for _, msg := range aggregated {
		flushedBatch, err := d.batcher.Append(msg)
		if err != nil {
			return fmt.Errorf("can not append message to batch: %w", err)
		}

		if len(flushedBatch) > 0 {
			// if Run has not yet started, we might have no ticker to reset.
			// this normally only happens during an integration test, on production
			// systems it normally takes a moment before we have data to write
			// to the producer daemon.
			if d.ticker != nil {
				d.ticker.Reset(d.settings.Interval)
			}
			d.outCh.Write(flushedBatch)
		}
	}

	return nil
}

func (d *producerDaemon) IsPartitionedOutput() bool {
	po, ok := d.output.(PartitionedOutput)

	return ok && po.IsPartitionedOutput()
}

func (d *producerDaemon) GetMaxMessageSize() *int {
	if sro, ok := d.output.(SizeRestrictedOutput); ok {
		return sro.GetMaxMessageSize()
	}

	return nil
}

func (d *producerDaemon) GetMaxBatchSize() *int {
	if sro, ok := d.output.(SizeRestrictedOutput); ok {
		return sro.GetMaxBatchSize()
	}

	return nil
}

func (d *producerDaemon) tickerLoop(ctx context.Context) error {
	var err error

	for {
		select {
		case <-ctx.Done():
			d.ticker.Stop()
			return nil

		case <-d.ticker.Chan():
			d.lck.Lock()

			if err = d.flushAll(); err != nil {
				d.logger.Error("can not flush all messages: %w", err)
			}

			d.lck.Unlock()
		}
	}
}

func (d *producerDaemon) applyAggregation(ctx context.Context, batch []WritableMessage) ([]*Message, error) {
	if d.settings.AggregationSize <= 1 {
		result := make([]*Message, len(batch))

		for i, msg := range batch {
			streamMsg, ok := msg.(*Message)

			if !ok {
				return nil, fmt.Errorf("are you writing to the daemon directly? expected a stream.Message to be written to the producer daemon, got %T instead", msg)
			}

			result[i] = streamMsg
		}

		return result, nil
	}

	var result []*Message

	for _, msg := range batch {
		streamMsg, ok := msg.(*Message)

		if !ok {
			return nil, fmt.Errorf("are you writing to the daemon directly? expected a stream.Message to be written to the producer daemon, got %T instead", msg)
		}

		readyMessages, err := d.aggregator.Write(ctx, streamMsg)
		if err != nil {
			return nil, err
		}

		for _, readyMessage := range readyMessages {
			d.writeMetricAggregateSize(readyMessage.MessageCount)
			aggregateMessage := BuildAggregateMessage(readyMessage.Body, d.settings.MessageAttributes, readyMessage.Attributes)

			result = append(result, aggregateMessage)
		}
	}

	return result, nil
}

func (d *producerDaemon) flushAggregate() ([]*Message, error) {
	aggregates, err := d.aggregator.Flush()
	if err != nil {
		return nil, fmt.Errorf("can not marshal aggregates: %w", err)
	}

	messages := make([]*Message, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate.MessageCount == 0 {
			continue
		}

		d.writeMetricAggregateSize(aggregate.MessageCount)
		aggregateMessage := BuildAggregateMessage(aggregate.Body, d.settings.MessageAttributes, aggregate.Attributes)

		messages = append(messages, aggregateMessage)
	}

	return messages, nil
}

func (d *producerDaemon) flushBatch() {
	readyBatch := d.batcher.Flush()

	if len(readyBatch) == 0 {
		return
	}

	d.outCh.Write(readyBatch)
}

func (d *producerDaemon) flushAll() error {
	if readyBatches, err := d.flushAggregate(); err != nil {
		return fmt.Errorf("can not flush aggregation: %w", err)
	} else {
		for _, readyBatch := range readyBatches {
			flushedBatch, err := d.batcher.Append(readyBatch)
			if err != nil {
				return fmt.Errorf("can not append message to batch: %w", err)
			}

			if len(flushedBatch) > 0 {
				d.outCh.Write(flushedBatch)
			}
		}
	}

	d.flushBatch()

	return nil
}

func (d *producerDaemon) close() error {
	d.lck.Lock()
	defer d.lck.Unlock()
	defer d.outCh.Close()

	if err := d.flushAll(); err != nil {
		return fmt.Errorf("can not flush all messages: %w", err)
	}

	return nil
}

func (d *producerDaemon) outputLoop() error {
	// we want to use an empty context here instead of the kernel one to not cancel any remaining write requests
	ctx := context.Background()

	for {
		start := time.Now()
		batch, ok := d.outCh.Read()
		idleDuration := time.Since(start)

		if !ok {
			return nil
		}

		if err := d.output.Write(ctx, batch); err != nil {
			d.logger.Error("can not write messages to output in producer %s: %w", d.name, err)
		}

		d.writeMetricBatchSize(len(batch))
		d.writeMetricIdleDuration(idleDuration)
	}
}

func (d *producerDaemon) writeMetricMessageCount(count int) {
	d.metric.WriteOne(&metric.Datum{
		MetricName: metricNameMessageCount,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(count),
	})
}

func (d *producerDaemon) writeMetricBatchSize(size int) {
	d.metric.WriteOne(&metric.Datum{
		MetricName: metricNameBatchSize,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(size),
	})
}

func (d *producerDaemon) writeMetricAggregateSize(size int) {
	d.metric.WriteOne(&metric.Datum{
		MetricName: metricNameAggregateSize,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(size),
	})
}

func (d *producerDaemon) writeMetricIdleDuration(idleDuration time.Duration) {
	if idleDuration > d.settings.Interval {
		idleDuration = d.settings.Interval
	}

	d.metric.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameIdleDuration,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Unit:  metric.UnitMillisecondsAverage,
		Value: float64(idleDuration.Milliseconds()),
	})
}

func getProducerDaemonDefaultMetrics(name string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameMessageCount,
			Dimensions: map[string]string{
				"ProducerDaemon": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameBatchSize,
			Dimensions: map[string]string{
				"ProducerDaemon": name,
			},
			Unit:  metric.UnitCountAverage,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameAggregateSize,
			Dimensions: map[string]string{
				"ProducerDaemon": name,
			},
			Unit:  metric.UnitCountAverage,
			Value: 0.0,
		},
	}
}

func BuildAggregateMessage(aggregateBody string, attributes ...map[string]string) *Message {
	attributes = append(attributes, map[string]string{
		AttributeAggregate: strconv.FormatBool(true),
	})

	return NewMessage(aggregateBody, attributes...)
}
