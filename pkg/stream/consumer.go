package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"sync/atomic"
)

type ConsumerCallbackFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (ConsumerCallback, error)

//go:generate mockery -name=ConsumerCallback
type ConsumerCallback interface {
	BaseConsumerCallback
	Consume(ctx context.Context, model interface{}, attributes map[string]interface{}) (bool, error)
}

//go:generate mockery -name=RunnableConsumerCallback
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

		callback, err := callbackFactory(ctx, config, contextEnforcingLogger)
		if err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		contextEnforcingLogger.Enable()

		baseConsumer, err := NewBaseConsumer(config, logger, name, callback)
		if err != nil {
			return nil, fmt.Errorf("can not initiate base consumer: %w", err)
		}

		consumer := NewConsumerWithInterfaces(baseConsumer, callback)

		return consumer, nil
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
	return c.baseConsumer.run(kernelCtx, c.run)
}

func (c *Consumer) run(ctx context.Context) error {
	defer c.logger.Debug("runConsuming is ending")
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("return from consuming as the coffin is dying")

		case msg, ok := <-c.input.Data():
			if !ok {
				return nil
			}

			if _, ok := msg.Attributes[AttributeAggregate]; ok {
				c.processAggregateMessage(ctx, msg)
			} else {
				c.processSingleMessage(ctx, msg)
			}
		}
	}
}

func (c *Consumer) processAggregateMessage(ctx context.Context, msg *Message) {
	var err error
	var start = c.clock.Now()
	var batch = make([]*Message, 0)

	if ctx, _, err = c.encoder.Decode(ctx, msg, &batch); err != nil {
		c.handleError(ctx, err, "an error occurred during disaggregation of the message")
		return
	}

	c.Acknowledge(ctx, msg)

	for _, m := range batch {
		c.process(ctx, m)
	}

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, int32(len(batch)))

	c.writeMetrics(duration, len(batch))
}

func (c *Consumer) processSingleMessage(ctx context.Context, msg *Message) {
	start := c.clock.Now()
	ack := c.process(ctx, msg)

	if ack {
		c.Acknowledge(ctx, msg)
	}

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, 1)

	c.writeMetrics(duration, 1)
}

func (c *Consumer) process(ctx context.Context, msg *Message) bool {
	defer c.recover()

	var err error
	var ack bool
	var model interface{}
	var attributes map[string]interface{}

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

	if ack, err = c.callback.Consume(ctx, model, attributes); err != nil {
		c.handleError(ctx, err, "an error occurred during the consume operation")
	}

	return ack
}
