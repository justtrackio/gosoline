package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/metric"
	"sync"
	"time"
)

const (
	AttributeAggregate      = "goso.aggregate"
	metricNameMessageCount  = "MessageCount"
	metricNameBatchSize     = "BatchSize"
	metricNameAggregateSize = "AggregateSize"
	metricNameIdleDuration  = "IdleDuration"
)

var producerDaemonLock = sync.Mutex{}
var producerDaemons = map[string]*producerDaemon{}

type ProducerDaemonSettings struct {
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
	// and are now writing a message which compresses to 20 kb. Now your buffer reaches 60 kb and 80 kb base64 encoded.
	// Gosoline will not already output a 53 kb message if you requested 64 kb messages (it would accept a 56 kb message),
	// but after writing the next message
	AggregationMaxSize int `cfg:"aggregation_max_size" default:"65536" validate:"min=0"`
	// Additional attributes we append to each message
	MessageAttributes map[string]interface{} `cfg:"message_attributes"`
}

type producerDaemon struct {
	kernel.EssentialBackgroundModule

	name          string
	lck           sync.Mutex
	logger        log.Logger
	metric        metric.Writer
	aggregator    ProducerDaemonAggregator
	batcher       ProducerDaemonBatcher
	outCh         OutputChannel
	output        Output
	tickerFactory clock.TickerFactory
	ticker        clock.Ticker
	settings      ProducerDaemonSettings
}

func ResetProducerDaemons() {
	producerDaemonLock.Lock()
	defer producerDaemonLock.Unlock()

	producerDaemons = map[string]*producerDaemon{}
}

func ProvideProducerDaemon(config cfg.Config, logger log.Logger, name string) (*producerDaemon, error) {
	producerDaemonLock.Lock()
	defer producerDaemonLock.Unlock()

	if _, ok := producerDaemons[name]; ok {
		return producerDaemons[name], nil
	}

	var err error
	producerDaemons[name], err = NewProducerDaemon(config, logger, name)

	if err != nil {
		return nil, err
	}

	return producerDaemons[name], nil
}

func NewProducerDaemon(config cfg.Config, logger log.Logger, name string) (*producerDaemon, error) {
	settings := readProducerSettings(config, name)

	output, err := NewConfigurableOutput(config, logger, settings.Output)
	if err != nil {
		return nil, fmt.Errorf("can not create output for producer daemon %s: %w", name, err)
	}

	defaultMetrics := getProducerDaemonDefaultMetrics(name)
	metric := metric.NewDaemonWriter(defaultMetrics...)

	aggregator, err := NewProducerDaemonAggregator(settings.Daemon, settings.Compression)
	if err != nil {
		return nil, fmt.Errorf("can not create aggregator for producer daemon %s: %w", name, err)
	}

	return NewProducerDaemonWithInterfaces(logger, metric, aggregator, output, clock.NewRealTicker, name, settings.Daemon), nil
}

func NewProducerDaemonWithInterfaces(logger log.Logger, metric metric.Writer, aggregator ProducerDaemonAggregator, output Output, tickerFactory clock.TickerFactory, name string, settings ProducerDaemonSettings) *producerDaemon {
	return &producerDaemon{
		name:          name,
		logger:        logger,
		metric:        metric,
		aggregator:    aggregator,
		batcher:       NewProducerDaemonBatcher(settings),
		outCh:         NewOutputChannel(logger, settings.BufferSize),
		output:        output,
		tickerFactory: tickerFactory,
		settings:      settings,
	}
}

func (d *producerDaemon) GetStage() int {
	return 512
}

func (d *producerDaemon) Run(kernelCtx context.Context) error {
	d.ticker = d.tickerFactory(d.settings.Interval)

	cfn := coffin.New()
	// start the output loops before the ticker look - the output loop can't terminate until
	// we call close, while the ticker can if the context is already canceled
	for i := 0; i < d.settings.RunnerCount; i++ {
		cfn.GoWithContextf(kernelCtx, d.outputLoop, "panic during running the ticker loop")
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

func (d *producerDaemon) Write(_ context.Context, batch []WritableMessage) error {
	d.lck.Lock()
	defer d.lck.Unlock()

	var err error
	var aggregated []*Message
	d.writeMetricMessageCount(len(batch))

	if aggregated, err = d.applyAggregation(batch); err != nil {
		return fmt.Errorf("can not apply aggregation in producer %s: %w", d.name, err)
	}

	for _, msg := range aggregated {
		flushedBatch, err := d.batcher.Append(msg)

		if err != nil {
			return fmt.Errorf("can not append message to batch: %w", err)
		}

		if len(flushedBatch) > 0 {
			d.ticker.Reset()
			d.outCh.Write(flushedBatch)
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

		case <-d.ticker.Tick():
			d.lck.Lock()

			if err = d.flushAll(); err != nil {
				d.logger.Error("can not flush all messages: %w", err)
			}

			d.lck.Unlock()
		}
	}
}

func (d *producerDaemon) applyAggregation(batch []WritableMessage) ([]*Message, error) {
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

		readyMessages, err := d.aggregator.Write(streamMsg)

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

func (d *producerDaemon) flushAggregate() (*Message, error) {
	aggregate, err := d.aggregator.Flush()

	if err != nil {
		return nil, fmt.Errorf("can not marshal aggregate: %w", err)
	}

	if aggregate.MessageCount == 0 {
		return nil, nil
	}

	d.writeMetricAggregateSize(aggregate.MessageCount)
	aggregateMessage := BuildAggregateMessage(aggregate.Body, d.settings.MessageAttributes, aggregate.Attributes)

	return aggregateMessage, nil
}

func (d *producerDaemon) flushBatch() {
	readyBatch := d.batcher.Flush()

	if len(readyBatch) == 0 {
		return
	}

	d.outCh.Write(readyBatch)
}

func (d *producerDaemon) flushAll() error {
	if readyBatch, err := d.flushAggregate(); err != nil {
		return fmt.Errorf("can not flush aggregation: %w", err)
	} else if readyBatch != nil {
		flushedBatch, err := d.batcher.Append(readyBatch)

		if err != nil {
			return fmt.Errorf("can not append message to batch: %w", err)
		}

		if len(flushedBatch) > 0 {
			d.outCh.Write(flushedBatch)
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

func (d *producerDaemon) outputLoop(ctx context.Context) error {
	for {
		start := time.Now()
		batch, ok := d.outCh.Read()
		idleDuration := time.Since(start)

		if !ok {
			return nil
		}

		// no need to have some delayed cancel context or so here - if you need this, your output should've already provided that
		if err := d.output.Write(ctx, batch); err != nil {
			if exec.IsRequestCanceled(err) {
				// we were not fast enough to write all messages and have just lost some messages.
				// however, if this would be a problem, you shouldn't be using the producer daemon at all.
				d.logger.Warn("can not write messages to output in producer %s because of canceled context", d.name)
			} else {
				d.logger.Error("can not write messages to output in producer %s: %w", d.name, err)
			}
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

func BuildAggregateMessage(aggregateBody string, attributes ...map[string]interface{}) *Message {
	attributes = append(attributes, map[string]interface{}{
		AttributeAggregate: true,
	})

	return NewJsonMessage(aggregateBody, attributes...)
}
