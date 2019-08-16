package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"gopkg.in/tomb.v2"
	"time"
)

const metricNameConsumerProcessedCount = "ConsumerProcessedCount"

//go:generate mockery -name=ConsumerCallback
type ConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	Consume(ctx context.Context, msg *Message) (bool, error)
}

type Consumer struct {
	kernel.ForegroundModule
	ConsumerAcknowledge

	logger mon.Logger
	mw     mon.MetricWriter
	tracer tracing.Tracer
	tmb    tomb.Tomb
	ticker *time.Ticker

	name      string
	callback  ConsumerCallback
	processed int
}

func NewConsumer(callback ConsumerCallback) *Consumer {
	return &Consumer{
		callback: callback,
	}
}

func (c *Consumer) Boot(config cfg.Config, logger mon.Logger) error {
	err := c.callback.Boot(config, logger)

	if err != nil {
		return err
	}

	appId := cfg.GetAppIdFromConfig(config)
	c.name = fmt.Sprintf("consumer-%v-%v", appId.Family, appId.Application)

	c.logger = logger
	c.tracer = tracing.NewAwsTracer(config)

	defaultMetrics := getConsumerDefaultMetrics()
	c.mw = mon.NewMetricDaemonWriter(defaultMetrics...)

	idleTimeout := config.GetDuration("consumer_idle_timeout")
	c.ticker = time.NewTicker(idleTimeout * time.Second)

	inputName := config.GetString("consumer_input")
	input := NewConfigurableInput(config, logger, inputName)

	c.input = input
	c.ConsumerAcknowledge = NewConsumerAcknowledgeWithInterfaces(logger, input)

	return nil
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.logger.Info("leaving consumer ", c.name)

	c.tmb.Go(c.input.Run)
	c.tmb.Go(c.consume)

	for {
		select {
		case <-ctx.Done():
			c.input.Stop()
			return c.tmb.Wait()

		case <-c.tmb.Dead():
			c.input.Stop()
			return c.tmb.Err()

		case <-c.ticker.C:
			c.logger.Infof("processed %v messages", c.processed)

			c.mw.WriteOne(&mon.MetricDatum{
				MetricName: metricNameConsumerProcessedCount,
				Value:      float64(c.processed),
			})

			c.processed = 0
		}
	}
}

func (c *Consumer) consume() error {
	for {
		msg, ok := <-c.input.Data()

		if !ok {
			return nil
		}

		c.processed++
		c.doCallback(msg)
	}
}

func (c *Consumer) doCallback(msg *Message) {
	ctx, trans := c.tracer.StartSpanFromTraceAble(msg, c.name)
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
