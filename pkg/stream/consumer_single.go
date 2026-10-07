package stream

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/tracing"
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

// Consumer processes individual records using the shared consumer lifecycle.
type Consumer struct {
	*consumerBase
	callback UntypedConsumerCallback
}

var _ kernel.FullModule = &Consumer{}

func NewUntypedConsumer(name string, callbackFactory UntypedConsumerCallbackFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var callback UntypedConsumerCallback
		var settings ConsumerSettings
		var base *consumerBase

		loggerCallback := logger.WithChannel("consumerCallback")

		if callback, err = callbackFactory(ctx, config, loggerCallback); err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		if settings, err = ReadConsumerSettings(config, name); err != nil {
			return nil, fmt.Errorf("can not read consumer settings for %s: %w", name, err)
		}

		schemaCallback, _ := callback.(SchemaSettingsAwareCallback)

		if base, err = newConsumerBase(ctx, config, logger, name, schemaCallback, settings, newConsumerSingleRetryHandler); err != nil {
			return nil, fmt.Errorf("can not initiate consumer: %w", err)
		}

		return NewUntypedConsumerWithInterfaces(base, callback), nil
	}
}

func NewUntypedConsumerWithInterfaces(base *consumerBase, callback UntypedConsumerCallback) *Consumer {
	consumer := &Consumer{consumerBase: base, callback: callback}
	consumer.setCallbackHooks(callback)
	consumer.inputProcess = consumer.processData
	consumer.retryProcess = consumer.processData

	return consumer
}

func (c *Consumer) processData(ctx context.Context, msg *Message) (ack bool) {
	var err error
	var newCtx context.Context

	processingID := c.processingSequence.Add(1)
	c.processingStartedAt.Put(processingID, c.clock.Now())

	defer c.processingStartedAt.Remove(processingID)

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

		if newCtx, _, err = decoder.Decode(gracedCtx, nil, maps.Clone(msg.Attributes)); err != nil {
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

	// Decoders consume context attributes; preserve the original message for redelivery.
	decodeMsg := *msg
	decodeMsg.Attributes = maps.Clone(msg.Attributes)

	if ctx, _, err = c.encoder.Decode(ctx, &decodeMsg, &batch); err != nil {
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

		duration := c.clock.Since(start)
		c.processed.Add(1)

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

	duration := c.clock.Since(start)
	c.processed.Add(1)
	c.writeMetricDurationAndProcessedCount(gracedCtx, duration, 1)

	return
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

	// Decoders consume context attributes; preserve the original message for retries.
	decodeMsg := *msg
	decodeMsg.Attributes = maps.Clone(msg.Attributes)

	if gracedCtx, attributes, err = c.encoder.Decode(gracedCtx, &decodeMsg, model); err != nil {
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

	if messageId, ok = msg.Attributes[AttributeSqsMessageId]; ok {
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

func (c *Consumer) recover(ctx context.Context, msg *Message) {
	err := coffin.ResolveRecovery(recover())
	if err == nil {
		return
	}

	c.handleError(ctx, err, "a panic occurred during the consume operation")
	if msg == nil || c.hasNativeRetry() {
		return
	}

	c.retry(ctx, msg)
}

func (c *Consumer) retry(ctx context.Context, msg *Message) {
	if !c.settings.Retry.Enabled {
		return
	}

	retryMsg, retryId := c.buildRetryMessage(msg)
	ctx = log.AppendGlobalContextFields(ctx, log.Fields{"retry_id": retryId})

	c.logger.Warn(ctx, "putting message with id %s into retry", retryId)
	c.writeMetricRetryCount(ctx, metricNameConsumerRetryPutCount)

	ctx, stop := exec.WithDelayedCancelContext(ctx, c.settings.Retry.GraceTime)
	defer stop()

	if err := c.retryHandler.Put(ctx, retryMsg); err != nil {
		c.handleError(ctx, err, "can not put the message into the retry handler")
	}
}

func (c *Consumer) hasNativeRetry() bool {
	_, ok := c.input.(RetryingInput)

	return ok
}
