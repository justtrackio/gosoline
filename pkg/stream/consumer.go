package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/reqctx"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type UntypedConsumerCallbackFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (UntypedConsumerCallback, error)

//go:generate go run github.com/vektra/mockery/v2 --name UntypedConsumerCallback
type UntypedConsumerCallback interface {
	GetModel(attributes map[string]string) (any, error)
	Consume(ctx context.Context, model any, attributes map[string]string) (bool, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name RunnableUntypedConsumerCallback
type RunnableUntypedConsumerCallback interface {
	UntypedConsumerCallback
	RunnableCallback
}

type Consumer struct {
	kernel.EssentialModule
	kernel.ApplicationStage

	clock        clock.Clock
	uuidGen      uuid.Uuid
	logger       log.Logger
	metricWriter metric.Writer
	tracer       tracing.Tracer
	encoder      MessageEncoder
	input        Input
	retryInput   Input
	retryHandler RetryHandler

	wg                  sync.WaitGroup
	stopped             sync.Once
	cancel              context.CancelFunc
	processingStartedAt funk.Maper[uint64, time.Time]
	processingSequence  atomic.Uint64

	drainCtx    context.Context
	drainCancel context.CancelFunc

	id              string
	name            string
	settings        ConsumerSettings
	processed       int32
	callback        UntypedConsumerCallback
	samplingDecider smpl.Decider
}

var _ kernel.FullModule = &Consumer{}

func NewUntypedConsumer(name string, callbackFactory UntypedConsumerCallbackFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		loggerCallback := logger.WithChannel("consumerCallback")

		callback, err := callbackFactory(ctx, config, loggerCallback)
		if err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		consumer, err := newConsumer(ctx, config, logger, name, callback)
		if err != nil {
			return nil, fmt.Errorf("can not initiate consumer: %w", err)
		}

		return consumer, nil
	}
}

func newConsumer(ctx context.Context, config cfg.Config, logger log.Logger, name string, callback UntypedConsumerCallback) (*Consumer, error) {
	var err error
	var settings ConsumerSettings
	var tracer tracing.Tracer
	var input, retryInput Input
	var encoder MessageEncoder
	var retryHandler RetryHandler

	consumerLogger := logger.WithChannel(fmt.Sprintf("consumer-%s", name))
	metricWriter := metric.NewWriter(getConsumerDefaultMetrics(name)...)

	if _, err = cfg.GetAppIdentity(config); err != nil {
		return nil, fmt.Errorf("can not get app identity from config: %w", err)
	}

	if settings, err = ReadConsumerSettings(config, name); err != nil {
		return nil, fmt.Errorf("can not read consumer settings for %s: %w", name, err)
	}

	if tracer, err = tracing.ProvideTracer(ctx, config, consumerLogger); err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	if input, err = NewConfigurableInput(ctx, config, consumerLogger, settings.Input); err != nil {
		return nil, err
	}

	if encoder, err = newConsumerEncoder(ctx, input, callback, settings.Encoding); err != nil {
		return nil, err
	}

	if retryInput, retryHandler, err = newConsumerRetryHandler(ctx, config, consumerLogger, input, &settings.Retry, name); err != nil {
		return nil, err
	}

	samplingDecider, err := smpl.ProvideDecider(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("could not initialize sampling decider: %w", err)
	}

	consumerMetadata := ConsumerMetadata{
		Name:         name,
		RetryEnabled: settings.Retry.Enabled,
		RetryType:    settings.Retry.Type,
	}
	if err = appctx.MetadataAppend(ctx, metadataKeyConsumers, consumerMetadata); err != nil {
		return nil, fmt.Errorf("can not access the appctx metadata: %w", err)
	}

	return NewUntypedConsumerWithInterfaces(
		uuid.New(),
		consumerLogger,
		metricWriter,
		tracer,
		input,
		encoder,
		retryInput,
		retryHandler,
		callback,
		settings,
		name,
		samplingDecider,
	), nil
}

func newConsumerEncoder(ctx context.Context, input Input, callback UntypedConsumerCallback, encoding EncodingType) (MessageEncoder, error) {
	encoderSettings := &MessageEncoderSettings{Encoding: encoding}
	schemaRegistryAwareInput, isSchemaRegistryAwareInput := input.(SchemaRegistryAwareInput)
	schemaSettings, err := getConsumerSchemaSettings(callback)
	if err != nil {
		return nil, err
	}

	if !isSchemaRegistryAwareInput || schemaSettings == nil {
		return NewMessageEncoder(encoderSettings), nil
	}

	externalEncoder, err := schemaRegistryAwareInput.InitSchemaRegistry(ctx, schemaSettings.WithEncoding(encoding))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize schema registry: %w", err)
	}

	encoderSettings.ExternalEncoder = externalEncoder

	return NewMessageEncoder(encoderSettings), nil
}

func getConsumerSchemaSettings(callback UntypedConsumerCallback) (*SchemaSettings, error) {
	schemaSettingsAware, ok := callback.(SchemaSettingsAwareCallback)
	if !ok {
		return nil, nil
	}

	return schemaSettingsAware.GetSchemaSettings()
}

func newConsumerRetryHandler(ctx context.Context, config cfg.Config, logger log.Logger, input Input, settings *ConsumerRetrySettings, name string) (Input, RetryHandler, error) {
	if retryingInput, ok := input.(RetryingInput); ok {
		settings.Enabled = true

		retryInput, retryHandler := retryingInput.GetRetryHandler()

		return retryInput, retryHandler, nil
	}

	retryInput, retryHandler, err := NewRetryHandler(ctx, config, logger, settings, name)
	if err != nil {
		return nil, nil, fmt.Errorf("can not create retry handler: %w", err)
	}

	return retryInput, retryHandler, nil
}

func NewUntypedConsumerWithInterfaces(
	uuidGen uuid.Uuid,
	logger log.Logger,
	metricWriter metric.Writer,
	tracer tracing.Tracer,
	input Input,
	encoder MessageEncoder,
	retryInput Input,
	retryHandler RetryHandler,
	callback UntypedConsumerCallback,
	settings ConsumerSettings,
	name string,
	samplingDecider smpl.Decider,
	clocks ...clock.Clock,
) *Consumer {
	clk := clock.Provider
	if len(clocks) > 0 {
		clk = clocks[0]
	}
	drainCtx, drainCancel := context.WithCancel(context.Background())

	return &Consumer{
		id:                  fmt.Sprintf("consumer-%s", name),
		name:                name,
		clock:               clk,
		uuidGen:             uuidGen,
		logger:              logger,
		metricWriter:        metricWriter,
		tracer:              tracer,
		encoder:             encoder,
		input:               input,
		retryInput:          retryInput,
		retryHandler:        retryHandler,
		settings:            settings,
		callback:            callback,
		samplingDecider:     samplingDecider,
		drainCtx:            drainCtx,
		drainCancel:         drainCancel,
		processingStartedAt: funk.NewMapSynced[uint64, time.Time](),
	}
}

func (c *Consumer) Run(kernelCtx context.Context) error {
	return c.run(kernelCtx)
}

// IsHealthy reports whether the consumer inputs remain healthy and no message processing has exceeded the configured healthcheck timeout.
//
// While idle, health depends solely on the inputs. Each active callback is evaluated independently against the timeout,
// so continuous processing remains healthy while every callback completes in time. When any active callback exceeds the
// timeout, the consumer is unhealthy even if other callbacks complete. Health recovers once all processing has completed,
// provided the inputs are healthy.
func (c *Consumer) IsHealthy(_ context.Context) (bool, error) {
	timedOut := c.processingStartedAt.Any(func(_ uint64, startedAt time.Time) bool {
		return c.clock.Since(startedAt) > c.settings.Healthcheck.Timeout
	})

	return c.isHealthy() && !timedOut, nil
}

func (c *Consumer) processData(ctx context.Context, msg *Message) (ack bool) {
	processingID := c.processingSequence.Add(1)
	c.processingStartedAt.Put(processingID, c.clock.Now())

	defer func() {
		c.processingStartedAt.Remove(processingID)
	}()

	// Keep the input context's values, but let in-flight processing outlive the input's cancellation
	// until the shared shutdown drain deadline expires.
	gracedCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	// Cancel this message when the shared drain window ends instead of starting a separate grace timer per message.
	stopDrainPropagation := context.AfterFunc(c.drainCtx, cancel)
	defer stopDrainPropagation()

	if retryId, ok := msg.Attributes[AttributeRetryId]; ok {
		// get the trace id from the message so our message can be found a lot easier in the logs
		decoder := tracing.NewMessageWithTraceEncoder(tracing.TraceIdErrorReturnStrategy{})
		newCtx, _, err := decoder.Decode(gracedCtx, nil, funk.MergeMaps(msg.Attributes)) // copy the attributes as Decode modifies the map...
		if err != nil {
			newCtx = gracedCtx
		}

		c.logger.Warn(newCtx, "retrying message with id %s", retryId)
		c.writeMetricRetryCount(newCtx, metricNameConsumerRetryGetCount)
	}

	if _, ok := msg.Attributes[AttributeAggregate]; ok {
		return c.processAggregateMessage(gracedCtx, msg, processingID)
	}

	return c.processSingleMessage(gracedCtx, msg)
}

func (c *Consumer) processAggregateMessage(ctx context.Context, msg *Message, processingID uint64) (ack bool) {
	ctx, span := c.startTracingContext(ctx)
	defer span.Finish()

	var err error
	batch := make([]*Message, 0)

	if ctx, _, err = c.encoder.Decode(ctx, msg, &batch); err != nil {
		c.handleError(ctx, err, "an error occurred during disaggregation of the message")

		return
	}

	anySucceeded := false
	allSucceeded := true

	for _, m := range batch {
		c.processingStartedAt.Put(processingID, c.clock.Now())
		start := c.clock.Now()

		succeeded := c.process(
			ctx,
			m,
			// we can only retry aggregate messages if we haven't acknowledged them yet and support native retry
			c.settings.AggregateMessageMode == AggregateMessageModeAtLeastOnce && c.hasNativeRetry(),
		)
		anySucceeded = anySucceeded || succeeded
		allSucceeded = allSucceeded && succeeded

		duration := c.clock.Now().Sub(start)
		atomic.AddInt32(&c.processed, 1)

		c.writeMetricDurationAndProcessedCount(ctx, duration, 1)
	}

	if c.settings.AggregateMessageMode == AggregateMessageModeAtMostOnce {
		return anySucceeded
	}

	return allSucceeded
}

func (c *Consumer) processSingleMessage(gracedCtx context.Context, msg *Message) (ack bool) {
	gracedCtx, span := c.startTracingContext(gracedCtx)
	defer span.Finish()

	start := c.clock.Now()

	ack = c.process(gracedCtx, msg, c.hasNativeRetry())

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, 1)
	c.writeMetricDurationAndProcessedCount(gracedCtx, duration, 1)

	return
}

func (c *Consumer) startTracingContext(ctx context.Context) (context.Context, tracing.Span) {
	ctx, span := c.tracer.StartSpanFromContext(ctx, c.id)

	ctx = log.InitContext(ctx)
	ctx = log.WithFingersCrossedScope(ctx)
	ctx = reqctx.New(ctx)

	return ctx, span
}

func (c *Consumer) process(gracedCtx context.Context, msg *Message, hasNativeRetry bool) bool {
	defer c.recover(gracedCtx, msg)

	// if we are shutting down, don't acknowledge any messages and try to retry them if needed
	select {
	case <-gracedCtx.Done():
		if !hasNativeRetry {
			c.retry(gracedCtx, msg)
		}

		return false
	default:
	}

	var err error
	var ack bool
	var model any
	var attributes map[string]string

	if model, err = c.callback.GetModel(msg.Attributes); err != nil {
		c.metricWriter.Write(gracedCtx, metric.Data{
			&metric.Datum{
				MetricName: metricNameConsumerUnknownModelError,
				Dimensions: map[string]string{
					"Consumer": c.name,
				},
				Value: 1.0,
			},
		})

		// Check if this error is ignorable based on consumer settings
		var ignorableErr IgnorableGetModelError
		if errors.As(err, &ignorableErr) && ignorableErr.IsIgnorableWithSettings(c.settings.IgnoreOnGetModelError) {
			c.logger.Info(gracedCtx, "ignoring message due to ignorable GetModel error: %s", err.Error())

			return true
		}

		c.handleError(gracedCtx, err, "an error occurred during the consume operation")

		return false
	}

	if model == nil {
		err := fmt.Errorf("can not get model for message attributes %v", msg.Attributes)
		c.handleError(gracedCtx, err, "an error occurred during the consume operation")

		return false
	}

	if gracedCtx, attributes, err = c.encoder.Decode(gracedCtx, msg, model); err != nil {
		c.handleError(gracedCtx, err, "an error occurred during the consume operation")

		return false
	}

	if smplCtx, _, err := c.samplingDecider.Decide(gracedCtx); err != nil {
		c.logger.Warn(gracedCtx, "could not decide on sampling: %s", err)
	} else {
		gracedCtx = smplCtx
	}

	var messageId string
	var ok bool
	messageId, ok = msg.Attributes[AttributeSqsMessageId]
	if ok {
		c.logger.WithFields(log.Fields{
			"sqs_message_id": messageId,
		}).Debug(gracedCtx, "processing sqs message")
	}

	if ack, err = c.callback.Consume(gracedCtx, model, attributes); err != nil {
		c.handleError(gracedCtx, err, "an error occurred during the consume operation")
	}

	if !ack && !hasNativeRetry {
		c.retry(gracedCtx, msg)
	}

	return ack
}
