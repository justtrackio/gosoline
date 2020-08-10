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
	"sync/atomic"
	"time"
)

const metricNameConsumerProcessedCount = "ConsumerProcessedCount"

//go:generate mockery -name=ConsumerCallback
type ConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	GetModel(attributes map[string]interface{}) interface{}
	Consume(ctx context.Context, model interface{}, attributes map[string]interface{}) (bool, error)
}

//go:generate mockery -name=RunnableConsumerCallback
type RunnableConsumerCallback interface {
	Run(ctx context.Context) error
}

//go:generate mockery -name=FullConsumerCallback
type FullConsumerCallback interface {
	ConsumerCallback
	RunnableConsumerCallback
}

type ConsumerSettings struct {
	Input       string        `cfg:"input" default:"consumer" validate:"required"`
	RunnerCount int           `cfg:"runner_count" default:"10" validate:"min=1"`
	Encoding    string        `cfg:"encoding" default:"application/json"`
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

	wg     sync.WaitGroup
	cancel context.CancelFunc

	id        string
	name      string
	settings  *ConsumerSettings
	callback  ConsumerCallback
	processed int32
}

func NewConsumer(name string, callback ConsumerCallback) *Consumer {
	return &Consumer{
		name:     name,
		callback: callback,
	}
}

func (c *Consumer) Boot(config cfg.Config, logger mon.Logger) error {
	if err := c.bootCallback(config, logger); err != nil {
		return err
	}

	settings := &ConsumerSettings{}
	key := ConfigurableConsumerKey(c.name)
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultForKey("encoding", defaultMessageBodyEncoding))

	appId := cfg.GetAppIdFromConfig(config)
	c.id = fmt.Sprintf("consumer-%s-%s-%s", appId.Family, appId.Application, c.name)

	tracer := tracing.ProviderTracer(config, logger)

	defaultMetrics := getConsumerDefaultMetrics()
	mw := mon.NewMetricDaemonWriter(defaultMetrics...)

	input := NewConfigurableInput(config, logger, settings.Input)
	encoder := NewMessageEncoder(&MessageEncoderSettings{
		Encoding: settings.Encoding,
	})

	c.BootWithInterfaces(logger, tracer, mw, input, encoder, settings)

	return nil
}

func (c *Consumer) BootWithInterfaces(logger mon.Logger, tracer tracing.Tracer, mw mon.MetricWriter, input Input, encoder MessageEncoder, settings *ConsumerSettings) {
	c.logger = logger.WithChannel("consumer")
	c.tracer = tracer
	c.mw = mw
	c.input = input
	c.ConsumerAcknowledge = NewConsumerAcknowledgeWithInterfaces(logger, c.input)
	c.encoder = encoder
	c.settings = settings
}

func (c *Consumer) Run(kernelCtx context.Context) error {
	defer c.logger.Infof("leaving consumer %s", c.name)
	c.logger.Infof("running consumer %s with input %s", c.name, c.settings.Input)

	// create ctx whose done channel is closed on dying coffin
	cfn, dyingCtx := coffin.WithContext(context.Background())

	// create ctx whose done channel is closed on dying coffin and manual cancel
	manualCtx := cfn.Context(context.Background())
	manualCtx, c.cancel = context.WithCancel(manualCtx)

	cfn.GoWithContextf(dyingCtx, c.input.Run, "panic during run of the consumer input")
	cfn.GoWithContextf(manualCtx, c.logConsumeCounter, "panic during counter log")
	cfn.GoWithContextf(manualCtx, c.runCallback, "panic during run of the callback")

	c.wg.Add(c.settings.RunnerCount)
	cfn.Go(c.stopConsuming)

	for i := 0; i < c.settings.RunnerCount; i++ {
		cfn.GoWithContextf(manualCtx, c.runConsuming, "panic during consuming")
	}

	// stop input on kernel cancel
	go func() {
		<-kernelCtx.Done()
		c.input.Stop()
	}()

	if err := cfn.Wait(); err != nil {
		return fmt.Errorf("error while waiting for all routines to stop: %w", err)
	}

	return nil
}

func (c *Consumer) bootCallback(config cfg.Config, logger mon.Logger) error {
	loggerCallback := logger.WithChannel("callback")
	contextEnforcingLogger := mon.NewContextEnforcingLogger(loggerCallback)

	err := c.callback.Boot(config, contextEnforcingLogger)

	if err != nil {
		return fmt.Errorf("error during booting the consumer callback: %w", err)
	}

	contextEnforcingLogger.Enable()

	return nil
}

func (c *Consumer) runCallback(ctx context.Context) error {
	defer c.logger.Debug("runCallback is ending")

	if runnable, ok := c.callback.(RunnableConsumerCallback); ok {
		return runnable.Run(ctx)
	}

	return nil
}

func (c *Consumer) runConsuming(ctx context.Context) error {
	defer c.logger.Debug("runConsuming is ending")
	defer c.wg.Done()

	var ok bool
	var msg *Message

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("return from consuming as the coffin is dying")

		case msg, ok = <-c.input.Data():
		}

		if !ok {
			return nil
		}

		c.doConsuming(msg)

		atomic.AddInt32(&c.processed, 1)
		c.mw.WriteOne(&mon.MetricDatum{
			MetricName: metricNameConsumerProcessedCount,
			Value:      1.0,
		})
	}
}

func (c *Consumer) doConsuming(msg *Message) {
	defer c.recover()

	ctx := context.Background()
	model := c.callback.GetModel(msg.Attributes)

	ctx, attributes, err := c.encoder.Decode(ctx, msg, model)
	logger := c.logger.WithContext(ctx)

	if err != nil {
		logger.Error(err, "an error occurred during the consume operation")
		return
	}

	ctx, span := c.tracer.StartSpanFromContext(ctx, c.id)
	defer span.Finish()

	ack, err := c.callback.Consume(ctx, model, attributes)

	if err != nil {
		logger.Error(err, "an error occurred during the consume operation")
	}

	if !ack {
		return
	}

	c.Acknowledge(ctx, msg)
}

func (c *Consumer) recover() {
	err := coffin.ResolveRecovery(recover())
	if err == nil {
		return
	}

	c.logger.Error(err, err.Error())
}

func (c *Consumer) stopConsuming() error {
	defer c.logger.Debug("stopConsuming is ending")

	c.wg.Wait()
	c.input.Stop()
	c.cancel()

	return nil
}

func (c *Consumer) logConsumeCounter(ctx context.Context) error {
	defer c.logger.Debug("logConsumeCounter is ending")

	ticker := time.NewTicker(c.settings.IdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			processed := atomic.SwapInt32(&c.processed, 0)

			c.logger.WithFields(mon.Fields{
				"count": processed,
				"name":  c.name,
			}).Infof("processed %v messages", processed)
		}
	}
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

func ConfigurableConsumerKey(name string) string {
	return fmt.Sprintf("stream.consumer.%s", name)
}
