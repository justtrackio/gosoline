package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"sync"
	"sync/atomic"
	"time"
)

const (
	metricNameConsumerProcessedCount = "ConsumerProcessedCount"
	metricNameConsumerDuration       = "ConsumerDuration"
)

type InputCallback interface {
	run(ctx context.Context) error
}

//go:generate mockery -name=RunnableCallback
type RunnableCallback interface {
	Run(ctx context.Context) error
}

type BaseConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	GetModel(attributes map[string]interface{}) interface{}
}

type ConsumerSettings struct {
	Input       string        `cfg:"input" default:"consumer" validate:"required"`
	RunnerCount int           `cfg:"runner_count" default:"10" validate:"min=1"`
	Encoding    string        `cfg:"encoding" default:"application/json"`
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
}

type baseConsumer struct {
	kernel.EssentialModule
	kernel.ApplicationStage
	ConsumerAcknowledge

	logger  mon.Logger
	encoder MessageEncoder
	mw      mon.MetricWriter
	tracer  tracing.Tracer
	clock   clock.Clock

	wg     sync.WaitGroup
	cancel context.CancelFunc

	id               string
	name             string
	settings         *ConsumerSettings
	consumerCallback interface{}
	inputCallback    InputCallback
	processed        int32
}

func newBaseConsumer(name string, consumerCallback BaseConsumerCallback, inputCallback InputCallback) *baseConsumer {
	return &baseConsumer{
		name:             name,
		consumerCallback: consumerCallback,
		inputCallback:    inputCallback,
		clock:            clock.Provider,
	}
}

func (c *baseConsumer) Boot(config cfg.Config, logger mon.Logger) error {
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

func (c *baseConsumer) BootWithInterfaces(logger mon.Logger, tracer tracing.Tracer, mw mon.MetricWriter, input Input, encoder MessageEncoder, settings *ConsumerSettings) {
	c.logger = logger.WithChannel("consumer")
	c.tracer = tracer
	c.mw = mw
	c.input = input
	c.ConsumerAcknowledge = NewConsumerAcknowledgeWithInterfaces(logger, c.input)
	c.encoder = encoder
	c.settings = settings
}

func (c *baseConsumer) bootCallback(config cfg.Config, logger mon.Logger) error {
	loggerCallback := logger.WithChannel("consumerCallback")
	contextEnforcingLogger := mon.NewContextEnforcingLogger(loggerCallback)

	err := c.consumerCallback.(BaseConsumerCallback).Boot(config, contextEnforcingLogger)

	if err != nil {
		return fmt.Errorf("error during booting the consumer consumerCallback: %w", err)
	}

	contextEnforcingLogger.Enable()

	return nil
}

func ConfigurableConsumerKey(name string) string {
	return fmt.Sprintf("stream.consumer.%s", name)
}

func getConsumerDefaultMetrics() mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameConsumerProcessedCount,
			Unit:       mon.UnitCount,
			Value:      0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameConsumerDuration,
			Unit:       mon.UnitMillisecondsAverage,
			Value:      0.0,
		},
	}
}

func (c *baseConsumer) Run(kernelCtx context.Context) error {
	defer c.logger.Infof("leaving consumer %s", c.name)
	c.logger.Infof("running consumer %s with input %s", c.name, c.settings.Input)

	// create ctx whose done channel is closed on dying coffin
	cfn, dyingCtx := coffin.WithContext(context.Background())

	// create ctx whose done channel is closed on dying coffin and manual cancel
	manualCtx := cfn.Context(context.Background())
	manualCtx, c.cancel = context.WithCancel(manualCtx)

	cfn.GoWithContextf(manualCtx, c.logConsumeCounter, "panic during counter log")
	cfn.GoWithContextf(manualCtx, c.runConsumerCallback, "panic during run of the consumerCallback")
	// run the input after the counters are running to make sure our coffin does not immediately
	// die just because Run() immediately returns
	cfn.GoWithContextf(dyingCtx, c.input.Run, "panic during run of the consumer input")

	c.wg.Add(c.settings.RunnerCount)
	cfn.Go(c.stopConsuming)

	for i := 0; i < c.settings.RunnerCount; i++ {
		cfn.GoWithContextf(manualCtx, c.inputCallback.run, "panic during consuming")
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

func (c *baseConsumer) logConsumeCounter(ctx context.Context) error {
	logger := c.logger.WithContext(ctx)
	defer logger.Debug("logConsumeCounter is ending")

	ticker := time.NewTicker(c.settings.IdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			processed := atomic.SwapInt32(&c.processed, 0)

			logger.WithFields(mon.Fields{
				"count": processed,
				"name":  c.name,
			}).Infof("processed %v messages", processed)
		}
	}
}

func (c *baseConsumer) runConsumerCallback(ctx context.Context) error {
	logger := c.logger.WithContext(ctx)
	defer logger.Debug("runConsumerCallback is ending")

	if runnable, ok := c.consumerCallback.(RunnableCallback); ok {
		return runnable.Run(ctx)
	}

	return nil
}

func (c *baseConsumer) stopConsuming() error {
	defer c.logger.Debug("stopConsuming is ending")

	c.wg.Wait()
	c.input.Stop()
	c.cancel()

	return nil
}

func (c *baseConsumer) recover() {
	err := coffin.ResolveRecovery(recover())
	if err == nil {
		return
	}

	c.logger.Error(err, err.Error())
}
