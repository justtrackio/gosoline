package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
	"time"
)

const (
	AttributeAggregate      = "goso.aggregate"
	metricNameMessageCount  = "MessageCount"
	metricNameBatchSize     = "BatchSize"
	metricNameAggregateSize = "AggregateSize"
)

var producerDaemonLock = sync.Mutex{}
var producerDaemons = map[string]*ProducerDaemon{}

type AggregateMarshaller func(body interface{}, attributes ...map[string]interface{}) (*Message, error)

type ProducerDaemonSettings struct {
	Enabled           bool                   `cfg:"enabled" default:"false"`
	Interval          time.Duration          `cfg:"interval" default:"1m"`
	BufferSize        int                    `cfg:"buffer_size" default:"10" validate:"min=1"`
	RunnerCount       int                    `cfg:"runner_count" default:"10" validate:"min=1"`
	BatchSize         int                    `cfg:"batch_size" default:"10" validate:"min=1"`
	AggregationSize   int                    `cfg:"aggregation_size" default:"1" validate:"min=1"`
	MessageAttributes map[string]interface{} `cfg:"message_attributes"`
}

type ProducerDaemon struct {
	kernel.EssentialModule

	name          string
	lck           sync.Mutex
	logger        mon.Logger
	metric        mon.MetricWriter
	aggregate     []WritableMessage
	batch         []WritableMessage
	outCh         chan []WritableMessage
	output        Output
	tickerFactory clock.TickerFactory
	ticker        clock.Ticker
	marshaller    AggregateMarshaller
	settings      ProducerDaemonSettings
}

func ProvideProducerDaemon(config cfg.Config, logger mon.Logger, name string) *ProducerDaemon {
	producerDaemonLock.Lock()
	defer producerDaemonLock.Unlock()

	if _, ok := producerDaemons[name]; ok {
		return producerDaemons[name]
	}

	producerDaemons[name] = NewProducerDaemon(config, logger, name)

	return producerDaemons[name]
}

func NewProducerDaemon(config cfg.Config, logger mon.Logger, name string) *ProducerDaemon {
	key := ConfigurableProducerKey(name)
	settings := &ProducerSettings{}
	config.UnmarshalKey(key, settings)

	defaultMetrics := getProducerDaemonDefaultMetrics(name)
	metric := mon.NewMetricDaemonWriter(defaultMetrics...)

	if settings.Output == "" {
		settings.Output = name
	}

	output := NewConfigurableOutput(config, logger, settings.Output)

	return &ProducerDaemon{
		name:          name,
		logger:        logger,
		metric:        metric,
		batch:         make([]WritableMessage, 0, settings.Daemon.BatchSize),
		outCh:         make(chan []WritableMessage, settings.Daemon.BufferSize),
		output:        output,
		tickerFactory: clock.NewRealTicker,
		marshaller:    MarshalJsonMessage,
		settings:      settings.Daemon,
	}
}

func NewProducerDaemonWithInterfaces(logger mon.Logger, metric mon.MetricWriter, output Output, tickerFactory clock.TickerFactory, marshaller AggregateMarshaller, name string, settings ProducerDaemonSettings) *ProducerDaemon {
	return &ProducerDaemon{
		name:          name,
		logger:        logger,
		metric:        metric,
		batch:         make([]WritableMessage, 0, settings.BatchSize),
		outCh:         make(chan []WritableMessage, settings.BufferSize),
		output:        output,
		tickerFactory: tickerFactory,
		marshaller:    marshaller,
		settings:      settings,
	}
}

func (d *ProducerDaemon) GetStage() int {
	return 512
}

func (d *ProducerDaemon) Boot(_ cfg.Config, _ mon.Logger) error {
	return nil
}

func (d *ProducerDaemon) Run(kernelCtx context.Context) error {
	d.ticker = d.tickerFactory(d.settings.Interval)

	cfn := coffin.New()
	cfn.GoWithContextf(kernelCtx, d.tickerLoop, "panic during running the ticker loop")

	for i := 0; i < d.settings.RunnerCount; i++ {
		cfn.GoWithContextf(kernelCtx, d.outputLoop, "panic during running the ticker loop")
	}

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

func (d *ProducerDaemon) WriteOne(ctx context.Context, msg WritableMessage) error {
	return d.Write(ctx, []WritableMessage{msg})
}

func (d *ProducerDaemon) Write(_ context.Context, batch []WritableMessage) error {
	d.lck.Lock()
	defer d.lck.Unlock()

	var err error
	d.writeMetricMessageCount(len(batch))

	if batch, err = d.applyAggregation(batch); err != nil {
		return fmt.Errorf("can not apply aggregation in producer %s: %w", d.name, err)
	}

	d.batch = append(d.batch, batch...)

	if len(d.batch) < d.settings.BatchSize {
		return nil
	}

	d.ticker.Reset()
	d.flushBatch()

	return nil
}

func (d *ProducerDaemon) tickerLoop(ctx context.Context) error {
	var err error

	for {
		select {
		case <-ctx.Done():
			d.ticker.Stop()
			return nil

		case <-d.ticker.Tick():
			d.lck.Lock()

			if err = d.flushAll(); err != nil {
				d.logger.Error(err, "can not flush all messages")
			}

			d.lck.Unlock()
		}
	}
}

func (d *ProducerDaemon) applyAggregation(batch []WritableMessage) ([]WritableMessage, error) {
	if d.settings.AggregationSize <= 1 {
		return batch, nil
	}

	d.aggregate = append(d.aggregate, batch...)

	if len(d.aggregate) < d.settings.AggregationSize {
		return nil, nil
	}

	return d.flushAggregate()
}

func (d *ProducerDaemon) flushAggregate() ([]WritableMessage, error) {
	if len(d.aggregate) == 0 {
		return nil, nil
	}

	size := d.settings.AggregationSize

	if len(d.aggregate) < size {
		size = len(d.aggregate)
	}

	var readyAggregate []WritableMessage
	readyAggregate, d.aggregate = d.aggregate[:size], d.aggregate[size:]

	d.writeMetricAggregateSize(len(readyAggregate))
	aggregateMessage, err := BuildAggregateMessage(d.marshaller, readyAggregate, d.settings.MessageAttributes)

	if err != nil {
		return nil, fmt.Errorf("can not marshal aggregate: %w", err)
	}

	return []WritableMessage{aggregateMessage}, nil
}

func (d *ProducerDaemon) flushBatch() {
	if len(d.batch) == 0 {
		return
	}

	size := d.settings.BatchSize

	if len(d.batch) < size {
		size = len(d.batch)
	}

	var readyBatch []WritableMessage
	readyBatch, d.batch = d.batch[:size], d.batch[size:]

	d.outCh <- readyBatch
}

func (d *ProducerDaemon) flushAll() error {
	var err error
	var batch []WritableMessage

	if batch, err = d.flushAggregate(); err != nil {
		return fmt.Errorf("can not flush aggregation: %w", err)
	}

	d.batch = append(d.batch, batch...)
	d.flushBatch()

	return nil
}

func (d *ProducerDaemon) close() error {
	d.lck.Lock()
	defer d.lck.Unlock()
	defer close(d.outCh)

	if err := d.flushAll(); err != nil {
		return fmt.Errorf("can not flush all messages: %w", err)
	}

	return nil
}

func (d *ProducerDaemon) outputLoop(ctx context.Context) error {
	var err error

	for batch := range d.outCh {
		if err = d.output.Write(ctx, batch); err != nil {
			d.logger.Errorf(err, "can not write messages to output in producer %s", d.name)
		}

		d.writeMetricBatchSize(len(batch))
	}

	return nil
}

func (d *ProducerDaemon) writeMetricMessageCount(count int) {
	d.metric.WriteOne(&mon.MetricDatum{
		MetricName: metricNameMessageCount,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(count),
	})
}

func (d *ProducerDaemon) writeMetricBatchSize(size int) {
	d.metric.WriteOne(&mon.MetricDatum{
		MetricName: metricNameBatchSize,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(size),
	})
}

func (d *ProducerDaemon) writeMetricAggregateSize(size int) {
	d.metric.WriteOne(&mon.MetricDatum{
		MetricName: metricNameAggregateSize,
		Dimensions: map[string]string{
			"ProducerDaemon": d.name,
		},
		Value: float64(size),
	})
}

func getProducerDaemonDefaultMetrics(name string) mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameMessageCount,
			Dimensions: map[string]string{
				"ProducerDaemon": name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameBatchSize,
			Dimensions: map[string]string{
				"ProducerDaemon": name,
			},
			Unit:  mon.UnitCountAverage,
			Value: 0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameAggregateSize,
			Dimensions: map[string]string{
				"ProducerDaemon": name,
			},
			Unit:  mon.UnitCountAverage,
			Value: 0.0,
		},
	}
}

func BuildAggregateMessage(marshaller AggregateMarshaller, aggregate []WritableMessage, attributes ...map[string]interface{}) (WritableMessage, error) {
	attributes = append(attributes, map[string]interface{}{
		AttributeAggregate: true,
	})

	return marshaller(aggregate, attributes...)
}
