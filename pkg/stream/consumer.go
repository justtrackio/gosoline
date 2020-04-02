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
	GetModel() interface{}
	Consume(ctx context.Context, model interface{}, attributes map[string]interface{}) (bool, error)
}

type ConsumerSettings struct {
	Input       string        `cfg:"input" default:"consumer" validate:"required"`
	RunnerCount int           `cfg:"runner_count" default:"10" validate:"min=1"`
	Encoding    string        `cfg:"encoding"`
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
}

type Consumer struct {
	kernel.EssentialModule
	kernel.ServiceStage
	ConsumerAcknowledge

	logger  mon.Logger
	encoder MessageEncoder
	mw      mon.MetricWriter
	tracer  tracing.Tracer
	cfn     coffin.Coffin
	ticker  *time.Ticker

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
	if err := c.boolCallback(config, logger); err != nil {
		return err
	}

	settings := &ConsumerSettings{}
	c.settings = settings

	key := fmt.Sprintf("stream.consumer.%s", c.name)
	config.UnmarshalKey(key, settings)

	appId := cfg.GetAppIdFromConfig(config)
	c.id = fmt.Sprintf("consumer-%v-%v", appId.Family, appId.Application)

	c.logger = logger.WithChannel("consumer")
	c.tracer = tracing.ProviderTracer(config, logger)
	c.ticker = time.NewTicker(settings.IdleTimeout)

	defaultMetrics := getConsumerDefaultMetrics()
	c.mw = mon.NewMetricDaemonWriter(defaultMetrics...)

	c.input = NewConfigurableInput(config, logger, settings.Input)
	c.ConsumerAcknowledge = NewConsumerAcknowledgeWithInterfaces(logger, c.input)

	c.encoder = NewMessageEncoder(&MessageEncoderSettings{
		Encoding: settings.Encoding,
	})

	return nil
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.logger.Info("leaving consumer ", c.id)

	c.cfn.GoWithContextf(ctx, c.input.Run, "panic during run of the consumer input")

	for i := 0; i < c.settings.RunnerCount; i++ {
		c.cfn.Gof(c.consume, "panic during consuming")
	}

	run := true

	for {
		if !run {
			break
		}

		select {
		case <-ctx.Done():
			run = false
			break

		case <-c.cfn.Dying():
			run = false
			break

		case <-c.cfn.Dead():
			run = false
			break

		case <-c.ticker.C:
			processed := atomic.SwapInt32(&c.processed, 0)

			c.logger.WithFields(mon.Fields{
				"count": processed,
			}).Infof("processed %v messages", processed)
		}
	}

	c.input.Stop()
	return c.cfn.Wait()
}

func (c *Consumer) boolCallback(config cfg.Config, logger mon.Logger) error {
	loggerCallback := logger.WithChannel("callback")
	contextEnforcingLogger := mon.NewContextEnforcingLogger(loggerCallback)

	err := c.callback.Boot(config, contextEnforcingLogger)

	if err != nil {
		return fmt.Errorf("error during booting the consumer callback: %w", err)
	}

	contextEnforcingLogger.Enable()

	return nil
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
	ctx := context.Background()
	model := c.callback.GetModel()

	ctx, attributes, err := c.encoder.Decode(ctx, msg, model)
	logger := c.logger.WithContext(ctx)

	if err != nil {
		logger.Error(err, "an error occurred during the consume operation")
		return
	}

	ctx, trans := c.tracer.StartSpanFromContext(ctx, c.id)
	defer trans.Finish()

	ack, err := c.callback.Consume(ctx, model, attributes)

	if err != nil {
		logger.Error(err, "an error occurred during the consume operation")
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
