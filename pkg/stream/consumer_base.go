package stream

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	metricNameConsumerDuration       = "Duration"
	metricNameConsumerError          = "Error"
	metricNameConsumerProcessedCount = "ProcessedCount"
	metricNameConsumerRetryGetCount  = "RetryGetCount"
	metricNameConsumerRetryPutCount  = "RetryPutCount"
	dataSourceInput                  = "input"
	dataSourceRetry                  = "retry"
	metadataKeyConsumers             = "stream.consumers"
)

type ConsumerMetadata struct {
	Name         string `json:"name"`
	RetryEnabled bool   `json:"retry_enabled"`
	RetryType    string `json:"retry_type"`
	RunnerCount  int    `json:"runner_count"`
}

//go:generate mockery --name RunnableCallback
type RunnableCallback interface {
	Run(ctx context.Context) error
}

type BaseConsumerCallback interface {
	GetModel(attributes map[string]string) any
}

type consumerData struct {
	msg   *Message
	src   string
	input Input
}

type baseConsumer struct {
	kernel.EssentialModule
	kernel.ApplicationStage
	ConsumerAcknowledge

	clock        clock.Clock
	uuidGen      uuid.Uuid
	logger       log.Logger
	metricWriter metric.Writer
	tracer       tracing.Tracer
	encoder      MessageEncoder
	retryInput   Input
	retryHandler RetryHandler

	wg      sync.WaitGroup
	stopped sync.Once
	cancel  context.CancelFunc
	data    chan *consumerData

	id               string
	name             string
	settings         *ConsumerSettings
	consumerCallback any
	processed        int32
}

func NewBaseConsumer(ctx context.Context, config cfg.Config, logger log.Logger, name string, consumerCallback BaseConsumerCallback) (*baseConsumer, error) {
	uuidGen := uuid.New()
	logger = logger.WithChannel(fmt.Sprintf("consumer-%s", name))

	settings := readConsumerSettings(config, name)
	appId := cfg.GetAppIdFromConfig(config)

	tracer, err := tracing.ProvideTracer(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	defaultMetrics := getConsumerDefaultMetrics(name)
	metricWriter := metric.NewWriter(defaultMetrics...)

	var input, retryInput Input
	var retryHandler RetryHandler

	if input, err = NewConfigurableInput(ctx, config, logger, settings.Input); err != nil {
		return nil, err
	}

	encoder := NewMessageEncoder(&MessageEncoderSettings{
		Encoding: settings.Encoding,
	})

	// if our input knows how to retry already,
	if retryingInput, ok := input.(RetryingInput); ok {
		settings.Retry.Enabled = true
		retryInput, retryHandler = retryingInput.GetRetryHandler()
	} else if retryInput, retryHandler, err = NewRetryHandler(ctx, config, logger, &settings.Retry, name); err != nil {
		return nil, fmt.Errorf("can not create retry handler: %w", err)
	}

	consumerMetadata := ConsumerMetadata{
		Name:         name,
		RetryEnabled: settings.Retry.Enabled,
		RetryType:    settings.Retry.Type,
		RunnerCount:  settings.RunnerCount,
	}

	if err = appctx.MetadataAppend(ctx, metadataKeyConsumers, consumerMetadata); err != nil {
		return nil, fmt.Errorf("can not access the appctx metadata: %w", err)
	}

	return NewBaseConsumerWithInterfaces(uuidGen, logger, metricWriter, tracer, input, encoder, retryInput, retryHandler, consumerCallback, settings, name, appId), nil
}

func NewBaseConsumerWithInterfaces(
	uuidGen uuid.Uuid,
	logger log.Logger,
	metricWriter metric.Writer,
	tracer tracing.Tracer,
	input Input,
	encoder MessageEncoder,
	retryInput Input,
	retryHandler RetryHandler,
	consumerCallback any,
	settings *ConsumerSettings,
	name string,
	appId cfg.AppId,
) *baseConsumer {
	return &baseConsumer{
		name:                name,
		id:                  fmt.Sprintf("consumer-%s-%s-%s-%s", appId.Family, appId.Group, appId.Application, name),
		clock:               clock.Provider,
		uuidGen:             uuidGen,
		logger:              logger,
		metricWriter:        metricWriter,
		tracer:              tracer,
		ConsumerAcknowledge: NewConsumerAcknowledgeWithInterfaces(logger, input),
		encoder:             encoder,
		retryInput:          retryInput,
		retryHandler:        retryHandler,
		settings:            settings,
		consumerCallback:    consumerCallback,
		data:                make(chan *consumerData),
	}
}

func (c *baseConsumer) run(kernelCtx context.Context, inputRunner func(ctx context.Context) error) error {
	defer c.logger.Info("leaving consumer %s", c.name)
	c.logger.Info("running consumer %s with input %s", c.name, c.settings.Input)

	// create ctx whose done channel is closed on dying coffin
	cfn, dyingCtx := coffin.WithContext(context.Background())

	// create ctx whose done channel is closed on dying coffin and manual cancel
	manualCtx := cfn.Context(context.Background())
	manualCtx, c.cancel = context.WithCancel(manualCtx)

	cfn.Go(func() error {
		cfn.GoWithContextf(manualCtx, c.logConsumeCounter, "panic during counter log")
		cfn.GoWithContextf(manualCtx, c.runConsumerCallback, "panic during run of the consumerCallback")
		// run the input after the counters are running to make sure our coffin does not immediately
		// die just because Run() immediately returns
		cfn.GoWithContextf(dyingCtx, c.input.Run, "panic during run of the consumer input")
		cfn.GoWithContextf(dyingCtx, c.retryInput.Run, "panic during run of the retry handler")
		cfn.GoWithContextf(dyingCtx, c.ingestData, "panic during shoveling the data")

		c.wg.Add(c.settings.RunnerCount)
		for i := 0; i < c.settings.RunnerCount; i++ {
			cfn.GoWithContextf(manualCtx, inputRunner, "panic during consuming")
		}

		cfn.Gof(c.stopConsuming, "panic during stopping the consuming")

		cfn.GoWithContext(manualCtx, func(manualCtx context.Context) error {
			// wait for kernel or coffin cancel...
			select {
			case <-manualCtx.Done():
			case <-kernelCtx.Done():
			}

			// and stop the input
			c.stopIncomingData()

			return nil
		})

		return nil
	})

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

			logger.WithFields(log.Fields{
				"count": processed,
				"name":  c.name,
			}).Info("processed %v messages", processed)
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

func (c *baseConsumer) ingestData(ctx context.Context) error {
	defer c.logger.Debug("ingestData is ending")
	defer close(c.data)

	cfn := coffin.New()
	cfn.GoWithContextf(ctx, c.ingestDataFromSource(c.input, dataSourceInput), "panic during shoveling data from input")
	cfn.GoWithContextf(ctx, c.ingestDataFromSource(c.retryInput, dataSourceRetry), "panic during shoveling data from retry")

	return cfn.Wait()
}

func (c *baseConsumer) ingestDataFromSource(input Input, src string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		defer c.logger.Debug("ingestDataFromSource %s is ending", src)
		defer c.stopIncomingData()

		for {
			select {
			case <-ctx.Done():
				return nil

			case msg, ok := <-input.Data():
				if !ok {
					return nil
				}

				if retryId, ok := msg.Attributes[AttributeRetryId]; ok {
					c.logger.Warn("retrying message with id %s", retryId)
					c.writeMetricRetryCount(metricNameConsumerRetryGetCount)
				}

				c.data <- &consumerData{
					msg:   msg,
					src:   src,
					input: input,
				}
			}
		}
	}
}

// this one acts as a fallback which should stop all still running routines
func (c *baseConsumer) stopConsuming() error {
	defer c.logger.Debug("stopConsuming is ending")

	c.wg.Wait()
	c.stopIncomingData()
	c.cancel()

	return nil
}

func (c *baseConsumer) stopIncomingData() {
	c.stopped.Do(func() {
		defer c.logger.Debug("stopIncomingData is ending")

		c.retryInput.Stop()
		c.input.Stop()
	})
}

func (c *baseConsumer) recover(ctx context.Context, msg *Message) {
	var err error

	if err = coffin.ResolveRecovery(recover()); err == nil {
		return
	}

	c.handleError(ctx, err, "a panic occurred during the consume operation")

	if msg == nil || c.hasNativeRetry() {
		return
	}

	c.retry(ctx, msg)
}

func (c *baseConsumer) retry(ctx context.Context, msg *Message) {
	if !c.settings.Retry.Enabled {
		return
	}

	retryMsg, retryId := c.buildRetryMessage(msg)

	ctx = log.AppendGlobalContextFields(ctx, log.Fields{
		"retry_id": retryId,
	})

	c.logger.WithContext(ctx).Warn("putting message with id %s into retry", retryId)
	c.writeMetricRetryCount(metricNameConsumerRetryPutCount)

	if err := c.retryHandler.Put(ctx, retryMsg); err != nil {
		c.handleError(ctx, err, "can not put the message into the retry handler")
	}
}

func (c *baseConsumer) hasNativeRetry() bool {
	_, ok := c.input.(RetryingInput)

	return ok
}

func (c *baseConsumer) buildRetryMessage(msg *Message) (retryMsg *Message, retryId string) {
	if retryId, ok := msg.Attributes[AttributeRetryId]; ok {
		return msg, retryId
	}

	retryId = c.uuidGen.NewV4()
	retryMsg = &Message{
		Attributes: funk.MergeMaps(msg.Attributes, map[string]string{
			AttributeRetry:   strconv.FormatBool(true),
			AttributeRetryId: retryId,
		}),
		Body: msg.Body,
	}

	return retryMsg, retryId
}

func (c *baseConsumer) handleError(ctx context.Context, err error, msg string) {
	c.logger.WithContext(ctx).Error("%s: %w", msg, err)

	c.metricWriter.Write(metric.Data{
		&metric.Datum{
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: 1.0,
		},
	})
}

func (c *baseConsumer) writeMetricDurationAndProcessedCount(duration time.Duration, processedCount int) {
	c.metricWriter.Write(metric.Data{
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerDuration,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: float64(duration.Milliseconds()),
		},
		&metric.Datum{
			MetricName: metricNameConsumerProcessedCount,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: float64(processedCount),
		},
	})
}

func (c *baseConsumer) writeMetricRetryCount(metricName string) {
	c.metricWriter.Write(metric.Data{
		&metric.Datum{
			MetricName: metricName,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: float64(1),
		},
	})
}

func getConsumerDefaultMetrics(name string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerProcessedCount,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryPutCount,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryGetCount,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
