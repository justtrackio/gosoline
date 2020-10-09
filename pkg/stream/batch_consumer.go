package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/hashicorp/go-multierror"
	"sync"
	"sync/atomic"
	"time"
)

//go:generate mockery -name=BatchConsumerCallback
type BatchConsumerCallback interface {
	Boot(config cfg.Config, logger mon.Logger) error
	GetModel(attributes map[string]interface{}) interface{}
	Consume(ctx []context.Context, models []interface{}, attributes []map[string]interface{}) ([]bool, []error)
}

type BatchConsumerSettings struct {
	ConsumerSettings
	BatchSize int `cfg:"batch_size" default:"10"`
}

type BaseBatchConsumer struct {
	kernel.EssentialModule
	kernel.ApplicationStage
	ConsumerAcknowledge

	logger  mon.Logger
	encoder MessageEncoder
	mw      mon.MetricWriter
	tracer  tracing.Tracer

	wg     sync.WaitGroup
	cancel context.CancelFunc

	id        string
	name      string
	settings  *BatchConsumerSettings
	callback  BatchConsumerCallback
	processed int32

	m               sync.Mutex
	batch           []*Message
	ticker          *time.Ticker
	tickerHasTicked bool
}

func NewBatchConsumer(name string, callback BatchConsumerCallback) *BaseBatchConsumer {
	return &BaseBatchConsumer{
		name:     name,
		callback: callback,
	}
}

func (c *BaseBatchConsumer) Boot(config cfg.Config, logger mon.Logger) error {
	if err := c.bootCallback(config, logger); err != nil {
		return err
	}

	settings := &BatchConsumerSettings{}
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

	ticker := time.NewTicker(settings.IdleTimeout)

	c.BootWithInterfaces(logger, tracer, mw, input, encoder, settings, ticker)

	return nil
}

func (c *BaseBatchConsumer) BootWithInterfaces(logger mon.Logger, tracer tracing.Tracer, mw mon.MetricWriter, input Input, encoder MessageEncoder, settings *BatchConsumerSettings, ticker *time.Ticker) {
	c.logger = logger.WithChannel("consumer")
	c.tracer = tracer
	c.mw = mw
	c.input = input
	c.ConsumerAcknowledge = NewConsumerAcknowledgeWithInterfaces(logger, c.input)
	c.encoder = encoder
	c.settings = settings
	c.batch = make([]*Message, 0, settings.BatchSize)
	c.ticker = ticker
}

func (c *BaseBatchConsumer) Run(kernelCtx context.Context) error {
	defer c.logger.Infof("leaving batch consumer %s", c.name)
	c.logger.Infof("running batch consumer %s with input %s", c.name, c.settings.Input)

	// create ctx whose done channel is closed on dying coffin
	cfn, dyingCtx := coffin.WithContext(context.Background())

	// create ctx whose done channel is closed on dying coffin and manual cancel
	manualCtx := cfn.Context(context.Background())
	manualCtx, c.cancel = context.WithCancel(manualCtx)

	cfn.GoWithContextf(manualCtx, c.logConsumeCounter, "panic during counter log")
	cfn.GoWithContextf(manualCtx, c.tickerTicking, "panic during ticker ticking")
	cfn.GoWithContextf(manualCtx, c.runCallback, "panic during run of the callback")
	// run the input after the counters are running to make sure our coffin does not immediately
	// die just because Run() immediately returns
	cfn.GoWithContextf(dyingCtx, c.input.Run, "panic during run of the batch consumer input")

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

func (c *BaseBatchConsumer) logConsumeCounter(ctx context.Context) error {
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

func (c *BaseBatchConsumer) tickerTicking(ctx context.Context) error {
	defer c.logger.Debug("tickerTicking is ending")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.ticker.C:
			c.tickerHasTicked = true

			c.logger.WithFields(mon.Fields{
				"name": c.name,
			}).Info("ticker has ticked")
		}
	}
}

func (c *BaseBatchConsumer) runCallback(ctx context.Context) error {
	defer c.logger.Debug("runCallback is ending")

	if runnable, ok := c.callback.(RunnableConsumerCallback); ok {
		return runnable.Run(ctx)
	}

	return nil
}

func (c *BaseBatchConsumer) stopConsuming() error {
	defer c.logger.Debug("stopConsuming is ending")

	c.wg.Wait()
	c.input.Stop()
	c.cancel()

	return nil
}

func (c *BaseBatchConsumer) runConsuming(ctx context.Context) error {
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

		c.m.Lock()
		c.batch = append(c.batch, msg)
		c.m.Unlock()

		if len(c.batch) < c.settings.BatchSize && !c.tickerHasTicked {
			continue
		}

		c.doConsuming()
		c.tickerHasTicked = false

		atomic.AddInt32(&c.processed, 1)
		c.mw.WriteOne(&mon.MetricDatum{
			MetricName: metricNameConsumerProcessedCount,
			Value:      1.0,
		})
	}
}

func (c *BaseBatchConsumer) doConsuming() {
	defer c.recover()

	if len(c.batch) == 0 {
		return
	}

	ctx := context.Background()

	c.m.Lock()
	defer c.m.Unlock()

	var err error
	var span tracing.Span
	ctxs := make([]context.Context, len(c.batch), len(c.batch))
	models := make([]interface{}, len(c.batch), len(c.batch))
	attributes := make([]map[string]interface{}, len(c.batch), len(c.batch))

	for i, msg := range c.batch {
		models[i] = c.callback.GetModel(msg.Attributes)

		// TODO: Check if xray does now disconnect because we don't use the supplied context
		ctx, attributes[i], err = c.encoder.Decode(ctx, msg, models[i])
		if err != nil {
			c.logger.WithContext(ctx).Error(err, "an error occurred during the batch consume operation")
			continue
		}

		ctxs[i], span = c.tracer.StartSpanFromContext(ctx, c.id)
		defer span.Finish()
	}

	acks, errs := c.callback.Consume(ctxs, models, attributes)
	if multierror.Append(nil, errs...).ErrorOrNil() != nil {
		for i, err := range errs {
			if err != nil {
				c.logger.WithContext(ctxs[i]).Error(err, "an error occurred during the batch consume operation")
			}
		}
	}

	for i, ack := range acks {
		if !ack {
			continue
		}

		c.Acknowledge(ctx, c.batch[i])
	}

	atomic.AddInt32(&c.processed, int32(len(c.batch)))
	c.mw.WriteOne(&mon.MetricDatum{
		MetricName: metricNameConsumerProcessedCount,
		Value:      float64(len(c.batch)),
	})
	c.batch = make([]*Message, 0, c.settings.BatchSize)
}

func (c *BaseBatchConsumer) recover() {
	err := coffin.ResolveRecovery(recover())
	if err == nil {
		return
	}

	c.logger.Error(err, err.Error())
}

func (c *BaseBatchConsumer) bootCallback(config cfg.Config, logger mon.Logger) error {
	loggerCallback := logger.WithChannel("callback")
	contextEnforcingLogger := mon.NewContextEnforcingLogger(loggerCallback)

	err := c.callback.Boot(config, contextEnforcingLogger)

	if err != nil {
		return fmt.Errorf("error during booting the consumer callback: %w", err)
	}

	contextEnforcingLogger.Enable()

	return nil
}
