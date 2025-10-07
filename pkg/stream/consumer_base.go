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
	"github.com/justtrackio/gosoline/pkg/exec"
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

type InitializeableCallback interface {
	Init(ctx context.Context) error
}

//go:generate go run github.com/vektra/mockery/v2 --name RunnableCallback
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
	consumerAcknowledge

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
	settings         ConsumerSettings
	consumerCallback any
	processed        int32
}

func NewBaseConsumer(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	name string,
	consumerCallback BaseConsumerCallback,
) (*baseConsumer, error) {
	uuidGen := uuid.New()
	logger = logger.WithChannel(fmt.Sprintf("consumer-%s", name))
	appId, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can not get app id from config: %w", err)
	}

	settings, err := ReadConsumerSettings(config, name)
	if err != nil {
		return nil, fmt.Errorf("can not read consumer settings for %s: %w", name, err)
	}

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

	return NewBaseConsumerWithInterfaces(
		uuidGen,
		logger,
		metricWriter,
		tracer,
		input,
		encoder,
		retryInput,
		retryHandler,
		consumerCallback,
		settings,
		name,
		appId,
	), nil
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
	settings ConsumerSettings,
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
		consumerAcknowledge: newConsumerAcknowledgeWithInterfaces(settings.AcknowledgeGraceTime, logger, input),
		encoder:             encoder,
		retryInput:          retryInput,
		retryHandler:        retryHandler,
		settings:            settings,
		consumerCallback:    consumerCallback,
		data:                make(chan *consumerData),
	}
}

func (c *baseConsumer) run(kernelCtx context.Context, inputRunner func(ctx context.Context) error) error {
	defer c.logger.Info(kernelCtx, "leaving consumer %s", c.name)

	if err := c.initConsumerCallback(kernelCtx); err != nil {
		return fmt.Errorf("can not init consumer callback: %w", err)
	}

	c.logger.Info(kernelCtx, "running consumer %s with input %s", c.name, c.settings.Input)

	// create ctx whose done channel is closed on dying coffin
	cfn, dyingCtx := coffin.WithContext(context.Background())

	// create ctx whose done channel is closed on dying coffin and manual cancel
	manualCtx := cfn.Context(context.Background())
	manualCtx, c.cancel = context.WithCancel(manualCtx)

	cfn.Go(func() error {
		cfn.GoWithContextf(manualCtx, c.logConsumeCounter, "panic during counter log")
		cfn.GoWithContextf(manualCtx, c.runConsumerCallback, "panic during run of the consumerCallback")
		cfn.GoWithContextf(dyingCtx, c.input.Run, "panic during run of the consumer input")
		cfn.GoWithContextf(dyingCtx, c.retryInput.Run, "panic during run of the retry handler")
		cfn.GoWithContextf(dyingCtx, c.ingestData, "panic during shoveling the data")

		c.wg.Add(c.settings.RunnerCount)
		for i := 0; i < c.settings.RunnerCount; i++ {
			cfn.GoWithContextf(kernelCtx, inputRunner, "panic during consuming")
		}

		cfn.GoWithContextf(manualCtx, c.stopConsuming, "panic during stopping the consuming")

		cfn.Go(func() error {
			// wait for kernel or coffin cancel...
			select {
			case <-manualCtx.Done():
			case <-kernelCtx.Done():
			}

			// and stop the input
			c.stopIncomingData(kernelCtx)

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
	defer c.logger.Debug(ctx, "logConsumeCounter is ending")

	lastLog := c.clock.Now()
	ticker := c.clock.NewTicker(c.settings.IdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logProcessedMessages(ctx, &lastLog)

			return nil
		case <-ticker.Chan():
			c.logProcessedMessages(ctx, &lastLog)
		}
	}
}

func (c *baseConsumer) logProcessedMessages(ctx context.Context, lastLog *time.Time) {
	processed := atomic.SwapInt32(&c.processed, 0)
	now := c.clock.Now()
	took := now.Sub(*lastLog)
	*lastLog = now

	c.logger.WithFields(log.Fields{
		"count": processed,
		"took":  took,
		"name":  c.name,
	}).Info(
		ctx,
		"consumer %s processed %d messages in %vs (%.1f messages/s)",
		c.name,
		processed,
		took.Seconds(),
		float64(processed)/took.Seconds(),
	)
}

func (c *baseConsumer) initConsumerCallback(ctx context.Context) error {
	if initializeable, ok := c.consumerCallback.(InitializeableCallback); ok {
		return initializeable.Init(ctx)
	}

	return nil
}

func (c *baseConsumer) runConsumerCallback(ctx context.Context) error {
	defer c.logger.Debug(ctx, "runConsumerCallback is ending")

	if runnable, ok := c.consumerCallback.(RunnableCallback); ok {
		return runnable.Run(ctx)
	}

	return nil
}

func (c *baseConsumer) ingestData(ctx context.Context) error {
	defer c.logger.Debug(ctx, "ingestData is ending")
	defer close(c.data)

	cfn := coffin.New()
	cfn.Go(func() error {
		cfn.GoWithContextf(ctx, c.ingestDataFromSource(c.input, dataSourceInput), "panic during shoveling data from input")
		cfn.GoWithContextf(ctx, c.ingestDataFromSource(c.retryInput, dataSourceRetry), "panic during shoveling data from retry")

		return nil
	})

	return cfn.Wait()
}

func (c *baseConsumer) ingestDataFromSource(input Input, src string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		defer c.logger.Debug(ctx, "ingestDataFromSource %s is ending", src)
		defer c.stopIncomingData(ctx)

		for msg := range input.Data() {
			if retryId, ok := msg.Attributes[AttributeRetryId]; ok {
				// get the trace id from the message so our message can be found a lot easier in the logs
				decoder := tracing.NewMessageWithTraceEncoder(tracing.TraceIdErrorReturnStrategy{})
				newCtx, _, err := decoder.Decode(ctx, nil, funk.MergeMaps(msg.Attributes)) // copy the attributes as Decode modifies the map...
				if err != nil {
					newCtx = ctx
				}

				c.logger.Warn(newCtx, "retrying message with id %s", retryId)
				c.writeMetricRetryCount(newCtx, metricNameConsumerRetryGetCount)
			}

			c.data <- &consumerData{
				msg:   msg,
				src:   src,
				input: input,
			}
		}

		return nil
	}
}

// this one acts as a fallback which should stop all still running routines
func (c *baseConsumer) stopConsuming(ctx context.Context) error {
	defer c.logger.Debug(ctx, "stopConsuming is ending")

	c.wg.Wait()
	c.stopIncomingData(ctx)
	c.cancel()

	return nil
}

func (c *baseConsumer) stopIncomingData(ctx context.Context) {
	c.stopped.Do(func() {
		defer c.logger.Debug(ctx, "stopIncomingData is ending")

		c.retryInput.Stop(ctx)
		c.input.Stop(ctx)
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

	c.logger.Warn(ctx, "putting message with id %s into retry", retryId)
	c.writeMetricRetryCount(ctx, metricNameConsumerRetryPutCount)

	ctx, stop := exec.WithDelayedCancelContext(ctx, c.settings.Retry.GraceTime)
	defer stop()

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
	c.logger.Error(ctx, "%s: %w", msg, err)

	c.metricWriter.Write(ctx, metric.Data{
		&metric.Datum{
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: 1.0,
		},
	})
}

func (c *baseConsumer) isHealthy() bool {
	retryInputHealthy := true
	if c.retryInput != nil {
		retryInputHealthy = c.retryInput.IsHealthy()
	}

	return c.input.IsHealthy() && retryInputHealthy
}

func (c *baseConsumer) writeMetricDurationAndProcessedCount(ctx context.Context, duration time.Duration, processedCount int) {
	c.metricWriter.Write(ctx, metric.Data{
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

func (c *baseConsumer) writeMetricRetryCount(ctx context.Context, metricName string) {
	c.metricWriter.Write(ctx, metric.Data{
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
