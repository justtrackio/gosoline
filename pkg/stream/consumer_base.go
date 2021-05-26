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
	metricNameConsumerDuration       = "Duration"
	metricNameConsumerError          = "Error"
	metricNameConsumerProcessedCount = "ProcessedCount"
)

//go:generate mockery -name=RunnableCallback
type RunnableCallback interface {
	Run(ctx context.Context) error
}

type BaseConsumerCallback interface {
	GetModel(attributes map[string]interface{}) interface{}
}

type ConsumerSettings struct {
	Input       string        `cfg:"input" default:"consumer" validate:"required"`
	RunnerCount int           `cfg:"runner_count" default:"1" validate:"min=1"`
	Encoding    EncodingType  `cfg:"encoding" default:"application/json"`
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
}

type baseConsumer struct {
	kernel.EssentialModule
	kernel.ApplicationStage
	ConsumerAcknowledge

	clock        clock.Clock
	logger       mon.Logger
	metricWriter mon.MetricWriter
	tracer       tracing.Tracer
	encoder      MessageEncoder

	wg     sync.WaitGroup
	cancel context.CancelFunc

	id               string
	name             string
	settings         *ConsumerSettings
	consumerCallback interface{}
	processed        int32
}

func NewBaseConsumer(config cfg.Config, logger mon.Logger, name string, consumerCallback BaseConsumerCallback) (*baseConsumer, error) {
	settings := readConsumerSettings(config, name)
	appId := cfg.GetAppIdFromConfig(config)

	tracer, err := tracing.ProvideTracer(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	defaultMetrics := getConsumerDefaultMetrics(name, settings.RunnerCount)
	metricWriter := mon.NewMetricDaemonWriter(defaultMetrics...)

	input, err := NewConfigurableInput(config, logger, settings.Input)
	if err != nil {
		return nil, err
	}

	encoder := NewMessageEncoder(&MessageEncoderSettings{
		Encoding: settings.Encoding,
	})

	return NewBaseConsumerWithInterfaces(logger, metricWriter, tracer, input, encoder, consumerCallback, settings, name, appId), nil
}

func NewBaseConsumerWithInterfaces(
	logger mon.Logger,
	metricWriter mon.MetricWriter,
	tracer tracing.Tracer,
	input Input,
	encoder MessageEncoder,
	consumerCallback interface{},
	settings *ConsumerSettings,
	name string,
	appId cfg.AppId,
) *baseConsumer {
	logger = logger.WithChannel("consumer")

	return &baseConsumer{
		name:                name,
		id:                  fmt.Sprintf("consumer-%s-%s-%s", appId.Family, appId.Application, name),
		logger:              logger,
		metricWriter:        metricWriter,
		tracer:              tracer,
		ConsumerAcknowledge: NewConsumerAcknowledgeWithInterfaces(logger, input),
		encoder:             encoder,
		settings:            settings,
		consumerCallback:    consumerCallback,
		clock:               clock.Provider,
	}
}

func (c *baseConsumer) run(kernelCtx context.Context, inputRunner func(ctx context.Context) error) error {
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
		cfn.GoWithContextf(manualCtx, inputRunner, "panic during consuming")
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

func (c *baseConsumer) handleError(ctx context.Context, err error, msg string) {
	c.logger.WithContext(ctx).Error(err, msg)

	c.metricWriter.Write(mon.MetricData{
		&mon.MetricDatum{
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: 1.0,
		},
	})
}

func (c *baseConsumer) writeMetrics(duration time.Duration, processedCount int) {
	c.metricWriter.Write(mon.MetricData{
		&mon.MetricDatum{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameConsumerDuration,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Unit:  mon.UnitMillisecondsAverage,
			Value: float64(duration.Milliseconds()),
		},
		&mon.MetricDatum{
			MetricName: metricNameConsumerProcessedCount,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: float64(processedCount),
		},
	})
}

func getConsumerDefaultMetrics(name string, runnerCount int) mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameConsumerProcessedCount,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}

func ConfigurableConsumerKey(name string) string {
	return fmt.Sprintf("stream.consumer.%s", name)
}

func readConsumerSettings(config cfg.Config, name string) *ConsumerSettings {
	settings := &ConsumerSettings{}
	key := ConfigurableConsumerKey(name)
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultForKey("encoding", defaultMessageBodyEncoding))

	return settings
}
