package stream

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
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

type (
	ProducerDaemonSettings struct {
		Enabled bool `cfg:"enabled" default:"false"`

		// Amount of time spend waiting for messages before sending out a batch.
		Interval time.Duration `cfg:"interval" default:"1m"`

		// Size of the buffer channel, i.e., how many messages can be in-flight at once? Generally it is a good idea to match
		// this with the number of runners.
		BufferSize int `cfg:"buffer_size" default:"10" validate:"min=1"`

		// Number of daemons running in the background, writing complete batches to the output.
		RunnerCount int `cfg:"runner_count" default:"10" validate:"min=1"`

		// How many SQS messages do we submit in a single batch? SQS can accept up to 10 messages per batch.
		// SNS doesn't support batching, so the value doesn't matter for SNS.
		BatchSize int `cfg:"batch_size" default:"10" validate:"min=1"`

		// How large may the sum of all messages in the aggregation be? For SQS you can't send more than 256 KB in one batch,
		// for SNS a single message can't be larger than 256 KB. We use 252 KB as default to leave some room for request
		// encoding and overhead.
		BatchMaxSize int `cfg:"batch_max_size" default:"258048" validate:"min=0"`

		// How many stream.Messages do we pack together in a single batch (one message in SQS) at once?
		AggregationSize int `cfg:"aggregation_size" default:"1" validate:"min=1"`

		// Maximum size in bytes of a batch. Defaults to 64 KB to leave some room for encoding overhead.
		// Set to 0 to disable limiting the maximum size for a batch (it will still not put more than BatchSize messages
		// in a batch).
		//
		// Note: Gosoline can't ensure your messages stay below this size if your messages are quite large (especially when
		// using compression). Imagine you already aggregated 40kb of compressed messages (around 53kb when base64 encoded)
		// and are now writing a message that compresses to 20 kb. Now your buffer reaches 60 kb and 80 kb base64 encoded.
		// Gosoline will not already output a 53 kb message if you requested 64 kb messages (it would accept a 56 kb message),
		// but after writing the next message
		AggregationMaxSize int `cfg:"aggregation_max_size" default:"65536" validate:"min=0"`

		// If you are writing to an output using a partition key, we ensure messages are still distributed to a partition
		// according to their partition key (although not necessary the same partition as without the producer daemon).
		// For this, we split the messages into buckets while collecting them, thus potentially aggregating more messages in
		// memory (depending on the number of buckets you configure).
		//
		// Note: This still does not guarantee that your messages are perfectly ordered - this is impossible as soon as you
		// have more than once producer. However, messages with the same partition key will end up in the same shard, so if
		// you are reading two different shards and one is much further behind than the other, you will not see messages
		// *massively* out of order - it should be roughly bounded by the time you buffer messages (the Interval setting) and
		// thus be not much more than a minute (using the default setting) instead of hours (if one shard is half a day behind
		// while the other is up-to-date).
		//
		// Second note: If you change the amount of partitions, messages might move between buckets and thus end up in different
		// shards than before. Thus, only do this if you can handle it (e.g., because no shard is currently lagging behind).
		PartitionBucketCount int `cfg:"partition_bucket_count" default:"128" validate:"min=1"`

		// Additional attributes we append to each message
		MessageAttributes map[string]string `cfg:"message_attributes"`
	}

	producerDaemon struct {
		kernel.EssentialBackgroundModule

		name                string
		lck                 sync.Mutex
		logger              log.Logger
		metric              metric.Writer
		aggregator          ProducerDaemonAggregator
		batcher             ProducerDaemonBatcher
		outCh               OutputChannel
		output              Output
		clock               clock.Clock
		ticker              clock.Ticker
		settings            ProducerDaemonSettings
		stopped             int32
		supportsAggregation bool
	}

	producerDaemonKey string
)

var _ SchemaRegistryAwareOutput = &producerDaemon{}

func producerDaemonName(name string) string {
	return fmt.Sprintf("producer-daemon-%s", name)
}

func ProvideProducerDaemon(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*producerDaemon, error) {
	producer, err := appctx.Provide(ctx, producerDaemonKey(producerDaemonName(name)), func() (*producerDaemon, error) {
		return NewProducerDaemon(ctx, config, logger, name)
	})
	if err != nil {
		return nil, fmt.Errorf("retrieving daemon from appctx: %w", err)
	}

	if producer.isStopped() {
		producer.restart()
	}

	return producer, err
}

func NewProducerDaemon(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*producerDaemon, error) {
	logger = logger.WithChannel(producerDaemonName(name))

	settings, err := readProducerSettings(config, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read producer settings for producer daemon %q: %w", name, err)
	}

	defaultMetrics := getProducerDaemonDefaultMetrics(name)
	metricWriter := metric.NewWriter(defaultMetrics...)

	output, outputCapabilities, err := NewConfigurableOutput(ctx, config, logger, settings.Output)
	if err != nil {
		return nil, fmt.Errorf("can not create output for producer daemon %s: %w", name, err)
	}

	if outputCapabilities.ProvidesCompression {
		settings.Compression = CompressionNone
	}

	if outputCapabilities.MaxBatchSize != nil && (outputCapabilities.IgnoreProducerDaemonBatchSettings || *outputCapabilities.MaxBatchSize < settings.Daemon.BatchSize) {
		settings.Daemon.BatchSize = *outputCapabilities.MaxBatchSize
	}

	if outputCapabilities.MaxMessageSize != nil && (outputCapabilities.IgnoreProducerDaemonBatchSettings || *outputCapabilities.MaxMessageSize < settings.Daemon.BatchMaxSize) {
		settings.Daemon.BatchMaxSize = *outputCapabilities.MaxMessageSize
	}

	aggregator, err := NewProducerDaemonAggregator(settings.Daemon, settings.Compression)
	if err != nil {
		return nil, fmt.Errorf("can not create aggregator for producer daemon %s: %w", name, err)
	}

	if outputCapabilities.IsPartitionedOutput && settings.Daemon.PartitionBucketCount > 1 {
		if aggregator, err = NewProducerDaemonPartitionedAggregator(logger, settings.Daemon, settings.Compression); err != nil {
			return nil, fmt.Errorf("can not create partitioned aggregator for producer daemon %s: %w", name, err)
		}
	}

	batcher := NewProducerDaemonBatcher(settings.Daemon)

	if !outputCapabilities.SupportsAggregation {
		aggregator = NewProducerDaemonNoopAggregator()
		batcher = NewProducerDaemonBatcherWithoutJsonEncoding(settings.Daemon)
	}

	return NewProducerDaemonWithInterfaces(
		logger,
		metricWriter,
		aggregator,
		batcher,
		output,
		clock.Provider,
		name,
		settings.Daemon,
		outputCapabilities.SupportsAggregation,
	), nil
}

func NewProducerDaemonWithInterfaces(
	logger log.Logger,
	metric metric.Writer,
	aggregator ProducerDaemonAggregator,
	batcher ProducerDaemonBatcher,
	output Output,
	clock clock.Clock,
	name string,
	settings ProducerDaemonSettings,
	supportsAggregation bool,
) *producerDaemon {
	return &producerDaemon{
		name:                name,
		logger:              logger,
		metric:              metric,
		aggregator:          aggregator,
		batcher:             batcher,
		outCh:               NewOutputChannel(logger, settings.BufferSize),
		output:              output,
		clock:               clock,
		settings:            settings,
		supportsAggregation: supportsAggregation,
	}
}

func (d *producerDaemon) isStopped() bool {
	return atomic.LoadInt32(&d.stopped) != 0
}

func (d *producerDaemon) restart() {
	d.lck.Lock()
	defer d.lck.Unlock()

	if d.outCh.IsClosed() {
		d.outCh = NewOutputChannel(d.logger, d.settings.BufferSize)
	}

	atomic.StoreInt32(&d.stopped, 0)
}

func (d *producerDaemon) GetStage() int {
	return kernel.StageProducerDaemon
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
		if err := d.close(kernelCtx); err != nil {
			return fmt.Errorf("error on close: %w", err)
		}
	case <-kernelCtx.Done():
		if err := d.close(kernelCtx); err != nil {
			return fmt.Errorf("error on close: %w", err)
		}
	}

	return cfn.Wait()
}

func (d *producerDaemon) InitSchemaRegistry(ctx context.Context, settings SchemaSettingsWithEncoding) (MessageBodyEncoder, error) {
	if schemaRegistryAwareOutput, ok := d.output.(SchemaRegistryAwareOutput); ok {
		return schemaRegistryAwareOutput.InitSchemaRegistry(ctx, settings)
	}

	return nil, fmt.Errorf("output does not support a schema registry")
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
	d.writeMetricMessageCount(ctx, len(batch))

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
			d.outCh.Write(ctx, flushedBatch)
		}
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

			if err = d.flushAll(ctx); err != nil {
				d.logger.Error(ctx, "can not flush all messages: %w", err)
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
			d.writeMetricAggregateSize(ctx, readyMessage.MessageCount)

			message := BuildAggregateMessage(readyMessage.Body, d.settings.MessageAttributes, readyMessage.Attributes)
			if !d.supportsAggregation {
				message = NewMessage(readyMessage.Body, d.settings.MessageAttributes, readyMessage.Attributes)
			}

			result = append(result, message)
		}
	}

	return result, nil
}

func (d *producerDaemon) flushAggregate(ctx context.Context) ([]*Message, error) {
	aggregates, err := d.aggregator.Flush()
	if err != nil {
		return nil, fmt.Errorf("can not marshal aggregates: %w", err)
	}

	messages := make([]*Message, 0, len(aggregates))
	for _, aggregate := range aggregates {
		if aggregate.MessageCount == 0 {
			continue
		}

		d.writeMetricAggregateSize(ctx, aggregate.MessageCount)

		message := BuildAggregateMessage(aggregate.Body, d.settings.MessageAttributes, aggregate.Attributes)
		if !d.supportsAggregation {
			message = NewMessage(aggregate.Body, d.settings.MessageAttributes, aggregate.Attributes)
		}

		messages = append(messages, message)
	}

	return messages, nil
}

func (d *producerDaemon) flushBatch(ctx context.Context) {
	readyBatch := d.batcher.Flush()

	if len(readyBatch) == 0 {
		return
	}

	d.outCh.Write(ctx, readyBatch)
}

func (d *producerDaemon) flushAll(ctx context.Context) error {
	readyBatches, err := d.flushAggregate(ctx)
	if err != nil {
		return fmt.Errorf("can not flush aggregation: %w", err)
	}

	for _, readyBatch := range readyBatches {
		flushedBatch, err := d.batcher.Append(readyBatch)
		if err != nil {
			return fmt.Errorf("can not append message to batch: %w", err)
		}

		if len(flushedBatch) > 0 {
			d.outCh.Write(ctx, flushedBatch)
		}
	}

	d.flushBatch(ctx)

	return nil
}

func (d *producerDaemon) close(ctx context.Context) error {
	d.lck.Lock()
	defer d.lck.Unlock()
	defer d.outCh.Close(ctx)

	if err := d.flushAll(ctx); err != nil {
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
			d.logger.Error(ctx, "can not write messages to output in producer %s: %w", d.name, err)
		}

		d.writeMetricBatchSize(ctx, len(batch))
		d.writeMetricIdleDuration(ctx, idleDuration)
	}
}

func (d *producerDaemon) writeMetricMessageCount(ctx context.Context, count int) {
	d.metric.WriteOne(ctx, &metric.Datum{
		MetricName: metricNameMessageCount,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(count),
	})
}

func (d *producerDaemon) writeMetricBatchSize(ctx context.Context, size int) {
	d.metric.WriteOne(ctx, &metric.Datum{
		MetricName: metricNameBatchSize,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(size),
	})
}

func (d *producerDaemon) writeMetricAggregateSize(ctx context.Context, size int) {
	d.metric.WriteOne(ctx, &metric.Datum{
		MetricName: metricNameAggregateSize,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(size),
	})
}

func (d *producerDaemon) writeMetricIdleDuration(ctx context.Context, idleDuration time.Duration) {
	if idleDuration > d.settings.Interval {
		idleDuration = d.settings.Interval
	}

	d.metric.WriteOne(ctx, &metric.Datum{
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
