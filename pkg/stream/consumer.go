package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ConsumerCallbackFactory[T any] func(ctx context.Context, config cfg.Config, logger log.Logger) (ConsumerCallback[T], error)

//go:generate mockery --name ConsumerCallback
type ConsumerCallback[T any] interface {
	BaseConsumerCallback[T]
	Consume(ctx context.Context, model T, attributes map[string]interface{}) (bool, error)
}

//go:generate mockery --name RunnableConsumerCallback
type RunnableConsumerCallback[T any] interface {
	ConsumerCallback[T]
	RunnableCallback
}

type Consumer[T comparable] struct {
	*baseConsumer
	callback ConsumerCallback[T]
}

func NewConsumer[T comparable](name string, callbackFactory ConsumerCallbackFactory[T]) func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		loggerCallback := logger.WithChannel("consumerCallback")
		contextEnforcingLogger := log.NewContextEnforcingLogger(loggerCallback)

		var err error
		var callback ConsumerCallback[T]
		var baseConsumer *baseConsumer

		if callback, err = callbackFactory(ctx, config, contextEnforcingLogger); err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		contextEnforcingLogger.Enable()

		if baseConsumer, err = NewBaseConsumer[T](ctx, config, logger, name, callback); err != nil {
			return nil, fmt.Errorf("can not initiate base consumer: %w", err)
		}

		return NewConsumerWithInterfaces(baseConsumer, callback), nil
	}
}

func NewConsumerWithInterfaces[T comparable](base *baseConsumer, callback ConsumerCallback[T]) *Consumer[T] {
	consumer := &Consumer[T]{
		baseConsumer: base,
		callback:     callback,
	}

	return consumer
}

func (c *Consumer[T]) Run(kernelCtx context.Context) error {
	return c.baseConsumer.run(kernelCtx, c.readData)
}

func (c *Consumer[T]) readData(ctx context.Context) error {
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

func (c *Consumer[T]) processAggregateMessage(ctx context.Context, cdata *consumerData) {
	var err error
	start := c.clock.Now()
	batch := make([]*Message, 0)

	if ctx, _, err = c.encoder.Decode(ctx, cdata.msg, &batch); err != nil {
		c.handleError(ctx, err, "an error occurred during disaggregation of the message")
		return
	}

	c.Acknowledge(ctx, cdata)

	for _, m := range batch {
		c.process(ctx, m)
	}

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, int32(len(batch)))

	c.writeMetricDurationAndProcessedCount(duration, len(batch))
}

func (c *Consumer[T]) processSingleMessage(ctx context.Context, cdata *consumerData) {
	start := c.clock.Now()

	if ack := c.process(ctx, cdata.msg); ack {
		c.Acknowledge(ctx, cdata)
	}

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, 1)
	c.writeMetricDurationAndProcessedCount(duration, 1)
}

func (c *Consumer[T]) process(ctx context.Context, msg *Message) bool {
	defer c.recover(ctx, msg)

	var err error
	var ack bool
	var model T
	var attributes map[string]interface{}

	if model = c.callback.GetModel(msg.Attributes); model == *new(T) {
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

	if ack, err = c.callback.Consume(ctx, model, attributes); err != nil {
		c.handleError(ctx, err, "an error occurred during the consume operation")
	}

	if ack {
		return true
	}

	c.retry(ctx, msg)

	return ack
}
