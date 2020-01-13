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

//go:generate mockery -name=ConsumerCallback
type ConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	Consume(ctx context.Context, msg *Message) (bool, error)
}

type ConsumerSettings struct {
	Input       string        `cfg:"input" default:"consumer"`
	RunnerCount int           `cfg:"runner_count" default:"10"`
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
}

type Consumer struct {
	kernel.EssentialModule
	ConsumerAcknowledge

	logger mon.Logger
	mw     mon.MetricWriter
	tracer tracing.Tracer
	cfn    coffin.Coffin
	ticker *time.Ticker

	id        string
	name      string
	settings  *ConsumerSettings
	callback  ConsumerCallback
	processed int32
}

func NewConsumer(name string, callback ConsumerCallback) *Consumer {
	return &Consumer{
		cfn:      coffin.New(),
		name:     name,
		callback: callback,
	}
}

func (c *Consumer) Boot(config cfg.Config, logger mon.Logger) error {
	err := c.callback.Boot(config, logger)

	if err != nil {
		return err
	}

	settings := &ConsumerSettings{}
	c.settings = settings

	key := fmt.Sprintf("stream.consumer.%s", c.name)
	config.UnmarshalKey(key, settings)

	appId := cfg.GetAppIdFromConfig(config)
	c.id = fmt.Sprintf("consumer-%v-%v", appId.Family, appId.Application)

	c.logger = logger.WithChannel("consumer")
	c.tracer = tracing.NewAwsTracer(config)

	defaultMetrics := getConsumerDefaultMetrics()
	c.mw = mon.NewMetricDaemonWriter(defaultMetrics...)

	c.ticker = time.NewTicker(settings.IdleTimeout)

	input := NewConfigurableInput(config, logger, settings.Input)

	c.input = input
	c.ConsumerAcknowledge = NewConsumerAcknowledgeWithInterfaces(logger, input)

	return nil
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.logger.Info("leaving consumer ", c.id)

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

func (c *Consumer) consume() error {
	for {
		msg, ok := <-c.input.Data()

		if !ok {
			return nil
		}

		c.doCallback(msg)

		atomic.AddInt32(&c.processed, 1)
		c.mw.WriteOne(&mon.MetricDatum{
			MetricName: metricNameConsumerProcessedCount,
			Value:      1.0,
		})
	}
}

func (c *Consumer) doCallback(msg *Message) {
	ctx, trans := c.tracer.StartSpanFromTraceAble(msg, c.id)
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
