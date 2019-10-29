package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"sync/atomic"
	"time"
)

const metricNameConsumerProcessedCount = "ConsumerProcessedCount"

//go:generate mockery -name Consumer
type Consumer interface {
	Boot(config cfg.Config, logger mon.Logger) error
	Run(ctx context.Context) error
	GetType() string
}

//go:generate mockery -name ConsumerCallback
type ConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	Consume(ctx context.Context, msg *Message) (bool, error)
}

type ConsumerSettings struct {
	Name        string
	Input       string        `cfg:"input" validate:"required"`
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
	RunnerCount int           `cfg:"runner_count" default:"1" validate:"min=1"`
}

type consumer struct {
	kernel.EssentialModule
	ConsumerAcknowledge

	metric mon.MetricWriter
	tracer tracing.Tracer
	cfn    coffin.Coffin
	ticker *time.Ticker

	settings *ConsumerSettings
	callback ConsumerCallback

	processed int32
}

func NewConsumer(callback ConsumerCallback) Consumer {
	return &consumer{
		cfn:      coffin.New(),
		callback: callback,
	}
}

func NewConsumerWithInterfaces(callback ConsumerCallback, logger mon.Logger, tracer tracing.Tracer, input Input, metricWriter mon.MetricWriter, settings *ConsumerSettings) Consumer {
	consumer := NewConsumer(callback).(*consumer)

	consumer.bootWithInterfaces(logger, tracer, input, metricWriter, settings)

	return consumer
}

func (c *consumer) Boot(config cfg.Config, logger mon.Logger) error {
	err := c.callback.Boot(config, logger)

	if err != nil {
		return err
	}

	tracer := tracing.NewAwsTracer(config)

	defaultMetrics := getConsumerDefaultMetrics()
	mw := mon.NewMetricDaemonWriter(defaultMetrics...)

	appId := cfg.GetAppIdFromConfig(config)

	settings := &ConsumerSettings{
		Name: fmt.Sprintf("consumer-%s-%s", appId.Family, appId.Application),
	}

	config.UnmarshalKey("consumer", settings)

	input := NewConfigurableInput(config, logger, settings.Input)

	c.bootWithInterfaces(logger, tracer, input, mw, settings)

	return nil
}

func (c *consumer) bootWithInterfaces(logger mon.Logger, tracer tracing.Tracer, input Input, metricWriter mon.MetricWriter, settings *ConsumerSettings) {
	c.logger = logger
	c.tracer = tracer
	c.input = input
	c.metric = metricWriter
	c.settings = settings
	c.ticker = time.NewTicker(settings.IdleTimeout)
}

func (c *consumer) Run(ctx context.Context) error {
	defer c.logger.Info("leaving consumer ", c.settings.Name)
	defer c.ticker.Stop()

	c.cfn.GoWithContextf(ctx, c.input.Run, "panic during run of the consumer input")

	for i := 0; i < c.settings.RunnerCount; i++ {
		c.cfn.Gof(c.consume, "panic during consuming")
	}

	for {
		select {
		case <-ctx.Done():
			c.input.Stop()
			return c.cfn.Wait()

		case <-c.cfn.Dying():
			c.input.Stop()
			return c.cfn.Wait()

		case <-c.ticker.C:
			processed := atomic.SwapInt32(&c.processed, 0)

			c.logger.WithFields(mon.Fields{
				"count": processed,
			}).Infof("processed %v messages", processed)
		}
	}
}

func (c *consumer) consume() error {
	for {
		msg, ok := <-c.input.Data()

		if !ok {
			return nil
		}

		c.doCallback(msg)

		atomic.AddInt32(&c.processed, 1)
		c.metric.WriteOne(&mon.MetricDatum{
			MetricName: metricNameConsumerProcessedCount,
			Value:      1.0,
		})
	}
}

func (c *consumer) doCallback(msg *Message) {
	ctx, trans := c.tracer.StartSpanFromTraceAble(msg, c.settings.Name)
	defer trans.Finish()

	ack, err := c.callback.Consume(ctx, msg)

	if err != nil {
		c.logger.WithContext(ctx).Error(err, "an error occurred during the consume operation")
	}

	if !ack {
		return
	}

	c.Acknowledge(ctx, msg)
}

func getConsumerDefaultMetrics() mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameConsumerProcessedCount,
			Unit:       mon.UnitCount,
			Value:      0.0,
		},
	}
}
