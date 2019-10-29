package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"sync"
	"time"
)

const MetricNameConsumerReceivedCount = "ConsumerReceivedCount"
const MetricNameConsumerProcessedCount = "ConsumerProcessedCount"

//go:generate mockery -name BatchConsumerCallback
type BatchConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	Process(ctx context.Context, messages []*Message) ([]*Message, error)
}

type BatchConsumerSettings struct {
	Name        string        `cfg:"name"`
	Input       string        `cfg:"input" validate:"required"`
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
	BatchSize   int           `cfg:"batch_size" default:"10" validate:"omitempty,min=1"`
}

type batchConsumer struct {
	kernel.EssentialModule
	ConsumerAcknowledge

	logger mon.Logger
	tracer tracing.Tracer
	metric mon.MetricWriter

	cfn    coffin.Coffin
	ticker *time.Ticker

	name     string
	callback BatchConsumerCallback
	settings *BatchConsumerSettings

	lck   sync.Mutex
	batch []*Message
}

func NewBatchConsumer(callback BatchConsumerCallback) Consumer {
	return &batchConsumer{
		cfn:      coffin.New(),
		callback: callback,
	}
}

func NewBatchConsumerWithInterfaces(callback BatchConsumerCallback, logger mon.Logger, tracer tracing.Tracer, input Input, metricWriter mon.MetricWriter, settings *BatchConsumerSettings) Consumer {
	c := NewBatchConsumer(callback).(*batchConsumer)

	c.bootWithInterfaces(logger, tracer, input, metricWriter, settings)

	return c
}

func (c *batchConsumer) Boot(config cfg.Config, logger mon.Logger) error {
	err := c.callback.Boot(config, logger)

	if err != nil {
		return err
	}

	tracer := tracing.NewAwsTracer(config)

	defaultMetrics := getDefaultConsumerMetrics()
	mw := mon.NewMetricDaemonWriter(defaultMetrics...)

	appId := cfg.GetAppIdFromConfig(config)

	settings := &BatchConsumerSettings{
		Name: fmt.Sprintf("consumer-%s-%s", appId.Family, appId.Application),
	}

	config.UnmarshalKey("consumer", settings)

	input := NewConfigurableInput(config, logger, settings.Input)

	c.bootWithInterfaces(logger, tracer, input, mw, settings)

	return nil
}

func (c *batchConsumer) bootWithInterfaces(logger mon.Logger, tracer tracing.Tracer, input Input, metricWriter mon.MetricWriter, settings *BatchConsumerSettings) {
	c.logger = logger
	c.tracer = tracer
	c.input = input
	c.metric = metricWriter
	c.settings = settings
	c.ticker = time.NewTicker(settings.IdleTimeout)
	c.batch = make([]*Message, 0, c.settings.BatchSize)
}

func (c *batchConsumer) Run(ctx context.Context) error {
	defer c.logger.Info("leaving consumer ", c.name)
	defer c.process(context.Background())
	defer c.ticker.Stop()

	c.cfn.GoWithContextf(ctx, c.input.Run, "panic during run of the consumer input")
	c.cfn.Gof(func() error {
		return c.consume(ctx)
	}, "panic during consuming")

	for {
		select {
		case <-ctx.Done():
			c.input.Stop()
			return c.cfn.Wait()

		case <-c.cfn.Dead():
			c.input.Stop()
			return c.cfn.Err()
		}
	}
}

func (c *batchConsumer) consume(ctx context.Context) error {
	for {
		force := false

		select {
		case msg, ok := <-c.input.Data():
			if !ok {
				return nil
			}

			c.batch = append(c.batch, msg)

		case <-c.ticker.C:
			force = true
		}

		if len(c.batch) >= c.settings.BatchSize || force {
			c.process(ctx)
		}
	}
}

func (c *batchConsumer) process(ctx context.Context) {
	c.lck.Lock()
	defer c.lck.Unlock()

	batchSize := len(c.batch)

	if batchSize == 0 {
		c.logger.Info("consumer has nothing to do")

		return
	}

	c.writeMetric(MetricNameConsumerReceivedCount, batchSize)

	defer func() {
		c.ticker = time.NewTicker(c.settings.IdleTimeout)
		c.batch = make([]*Message, 0, c.settings.BatchSize)
	}()

	c.ticker.Stop()

	msgs, err := c.callback.Process(ctx, c.batch)
	if err != nil {
		c.logger.Error(err, "could not consume batch messages")
	}

	c.AcknowledgeBatch(ctx, msgs)

	consumedCount := len(msgs)

	c.logger.Infof("consumer processed %d of %d messages", consumedCount, batchSize)
	c.writeMetric(MetricNameConsumerProcessedCount, consumedCount)
}

func (c *batchConsumer) writeMetric(name string, value int) {
	c.metric.WriteOne(&mon.MetricDatum{
		MetricName: MetricNameConsumerReceivedCount,
		Value:      float64(value),
	})
}

func getDefaultConsumerMetrics() mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: MetricNamePipelineReceivedCount,
			Unit:       mon.UnitCount,
			Value:      0.0,
		}, {
			Priority:   mon.PriorityHigh,
			MetricName: MetricNamePipelineProcessedCount,
			Unit:       mon.UnitCount,
			Value:      0.0,
		},
	}
}
