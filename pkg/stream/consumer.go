package stream

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ConsumerCallbackFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (ConsumerCallback, error)

//go:generate mockery --name ConsumerCallback
type ConsumerCallback interface {
	BaseConsumerCallback
	Consume(ctx context.Context, model interface{}, attributes map[string]string) (bool, error)
}

//go:generate mockery --name RunnableConsumerCallback
type RunnableConsumerCallback interface {
	ConsumerCallback
	RunnableCallback
}

type Consumer struct {
	*baseConsumer
	callback ConsumerCallback
}

func NewConsumer(name string, callbackFactory ConsumerCallbackFactory) func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		loggerCallback := logger.WithChannel("consumerCallback")
		contextEnforcingLogger := log.NewContextEnforcingLogger(loggerCallback)

		var err error
		var callback ConsumerCallback
		var baseConsumer *baseConsumer

		if callback, err = callbackFactory(ctx, config, contextEnforcingLogger); err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		contextEnforcingLogger.Enable()

		if baseConsumer, err = NewBaseConsumer(ctx, config, logger, name, callback); err != nil {
			return nil, fmt.Errorf("can not initiate base consumer: %w", err)
		}

		return NewConsumerWithInterfaces(baseConsumer, callback), nil
	}
}

func NewConsumerWithInterfaces(base *baseConsumer, callback ConsumerCallback) *Consumer {
	consumer := &Consumer{
		baseConsumer: base,
		callback:     callback,
	}

	return consumer
}

func (c *Consumer) Run(kernelCtx context.Context) error {
	return c.baseConsumer.run(kernelCtx, c.readData)
}

func (c *Consumer) readData(ctx context.Context) error {
	defer c.logger.Debug("read from input is ending")
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return nil

		case cdata, ok := <-c.data:
			if !ok {
				return nil
			}

			if _, ok := cdata.msg.Attributes[AttributeAggregate]; ok {
				c.processAggregateMessage(ctx, cdata)
			} else {
				c.processSingleMessage(ctx, cdata)
			}
		}
	}
}

func (c *Consumer) processAggregateMessage(ctx context.Context, cdata *consumerData) {
	var err error
	start := c.clock.Now()
	batch := make([]*Message, 0)

	if ctx, _, err = c.encoder.Decode(ctx, &cdata.msg, &batch); err != nil {
		c.handleError(ctx, err, "an error occurred during disaggregation of the message")

		return
	}

	c.Acknowledge(ctx, cdata, true)
	for _, m := range batch {
		_ = c.process(ctx, m, false) // we can't natively retry aggregate messages
	}

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, int32(len(batch)))

	c.writeMetricDurationAndProcessedCount(duration, len(batch))
}

func (c *Consumer) processSingleMessage(ctx context.Context, cdata *consumerData) {
	start := c.clock.Now()

	ack := c.process(ctx, &cdata.msg, c.hasNativeRetry())
	c.Acknowledge(ctx, cdata, ack)

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, 1)
	c.writeMetricDurationAndProcessedCount(duration, 1)
}

func (c *Consumer) process(ctx context.Context, msg *Message, hasNativeRetry bool) bool {
	defer c.recover(ctx, msg)

	var err error
	var ack bool
	var model interface{}
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

	ctx, span := c.tracer.StartSpanFromContext(ctx, c.id)
	defer span.Finish()

	ctx = log.InitContext(ctx)

	if ack, err = c.callback.Consume(ctx, model, attributes); err != nil {
		c.handleError(ctx, err, "an error occurred during the consume operation")
	}

	if !ack && !hasNativeRetry {
		// if we got cancelled, we have to ensure we put the message back into the retry queue it belongs to:
		delayedContext, stop := exec.WithDelayedCancelContext(ctx, time.Second*10)
		defer stop()

		c.retry(delayedContext, msg)
	}

	return ack
}
