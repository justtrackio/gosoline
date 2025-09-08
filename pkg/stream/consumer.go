package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reqctx"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type UntypedConsumerCallbackFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (UntypedConsumerCallback, error)

//go:generate go run github.com/vektra/mockery/v2 --name UntypedConsumerCallback
type UntypedConsumerCallback interface {
	BaseConsumerCallback
	Consume(ctx context.Context, model any, attributes map[string]string) (bool, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name RunnableUntypedConsumerCallback
type RunnableUntypedConsumerCallback interface {
	UntypedConsumerCallback
	RunnableCallback
}

type Consumer struct {
	*baseConsumer
	callback         UntypedConsumerCallback
	healthCheckTimer clock.HealthCheckTimer
}

var _ kernel.FullModule = &Consumer{}

func NewUntypedConsumer(name string, callbackFactory UntypedConsumerCallbackFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		loggerCallback := logger.WithChannel("consumerCallback")

		var err error
		var callback UntypedConsumerCallback
		var baseConsumer *baseConsumer

		if callback, err = callbackFactory(ctx, config, loggerCallback); err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		if baseConsumer, err = NewBaseConsumer(ctx, config, logger, name, callback); err != nil {
			return nil, fmt.Errorf("can not initiate base consumer: %w", err)
		}

		healthCheckTimer, err := clock.NewHealthCheckTimer(baseConsumer.settings.Healthcheck.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
		}

		return NewUntypedConsumerWithInterfaces(baseConsumer, callback, healthCheckTimer), nil
	}
}

func NewUntypedConsumerWithInterfaces(base *baseConsumer, callback UntypedConsumerCallback, healthCheckTimer clock.HealthCheckTimer) *Consumer {
	consumer := &Consumer{
		baseConsumer:     base,
		callback:         callback,
		healthCheckTimer: healthCheckTimer,
	}

	return consumer
}

func (c *Consumer) Run(kernelCtx context.Context) error {
	return c.run(kernelCtx, c.readData)
}

func (c *Consumer) IsHealthy(_ context.Context) (bool, error) {
	return c.isHealthy() && c.healthCheckTimer.IsHealthy(), nil
}

func (c *Consumer) readData(ctx context.Context) error {
	defer c.logger.Debug(ctx, "read from input is ending")
	defer c.wg.Done()

	// ticker to mark us as healthy should we not get any messages to process
	// (thus, the only way to get unhealthy would be if the consumer callback
	// takes too long to process a single message)
	ticker := c.clock.NewTicker(c.settings.Healthcheck.Timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case cdata, ok := <-c.data:
			if !ok {
				return nil
			}

			// we got a message and are thus healthy
			c.healthCheckTimer.MarkHealthy()

			if _, ok := cdata.msg.Attributes[AttributeAggregate]; ok {
				c.processAggregateMessage(ctx, cdata)
			} else {
				c.processSingleMessage(ctx, cdata)
			}

		case <-ticker.Chan():
			// we didn't get a message for quite some time, but we stay healthy
			c.healthCheckTimer.MarkHealthy()
		}
	}
}

func (c *Consumer) processAggregateMessage(ctx context.Context, cdata *consumerData) {
	ctx, span := c.startTracingContext(ctx)
	defer span.Finish()

	var err error
	start := c.clock.Now()
	batch := make([]*Message, 0)

	if ctx, _, err = c.encoder.Decode(ctx, cdata.msg, &batch); err != nil {
		c.handleError(ctx, err, "an error occurred during disaggregation of the message")

		return
	}

	c.Acknowledge(ctx, cdata, true)
	for _, m := range batch {
		_ = c.process(ctx, m, false) // we can't natively retry aggregate messages
	}

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, int32(len(batch)))

	c.writeMetricDurationAndProcessedCount(ctx, duration, len(batch))
}

func (c *Consumer) processSingleMessage(ctx context.Context, cdata *consumerData) {
	ctx, span := c.startTracingContext(ctx)
	defer span.Finish()

	start := c.clock.Now()

	ack := c.process(ctx, cdata.msg, c.hasNativeRetry())
	c.Acknowledge(ctx, cdata, ack)

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, 1)
	c.writeMetricDurationAndProcessedCount(ctx, duration, 1)
}

func (c *Consumer) startTracingContext(ctx context.Context) (context.Context, tracing.Span) {
	ctx, span := c.tracer.StartSpanFromContext(ctx, c.id)

	ctx = log.InitContext(ctx)
	ctx = reqctx.New(ctx)

	return ctx, span
}

func (c *Consumer) process(ctx context.Context, msg *Message, hasNativeRetry bool) bool {
	// once we processed a message, we made progress and are thus healthy
	defer c.healthCheckTimer.MarkHealthy()
	defer c.recover(ctx, msg)

	var err error
	var ack bool
	var model any
	var attributes map[string]string

	if model = c.callback.GetModel(msg.Attributes); model == nil {
		err := fmt.Errorf("can not get model for message attributes %v", msg.Attributes)
		c.handleError(ctx, err, "an error occurred during the consume operation")

		return false
	}

	if ctx, attributes, err = c.encoder.Decode(ctx, msg, model); err != nil {
		c.handleError(ctx, err, "an error occurred during the consume operation")

		return false
	}

	var messageId string
	var ok bool
	messageId, ok = msg.Attributes[AttributeSqsMessageId]
	if ok {
		c.logger.WithFields(log.Fields{
			"sqs_message_id": messageId,
		}).Debug(ctx, "processing sqs message")
	}

	if ack, err = c.callback.Consume(ctx, model, attributes); err != nil {
		c.handleError(ctx, err, "an error occurred during the consume operation")
	}

	if !ack && !hasNativeRetry {
		c.retry(ctx, msg)
	}

	return ack
}
