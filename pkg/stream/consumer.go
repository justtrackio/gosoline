package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"sync/atomic"
)

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

func NewConsumer(name string, callback ConsumerCallback) *Consumer {
	consumer := &Consumer{
		callback: callback,
	}

	baseConsumer := newBaseConsumer(name, callback, consumer)

	consumer.baseConsumer = baseConsumer

	return consumer
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
	batch := make([]*Message, 0)
	var err error

	start := c.clock.Now()

	ctx, _, err = c.encoder.Decode(ctx, msg, &batch)

	if err != nil {
		c.logger.WithContext(ctx).Error(err, "an error occurred during disaggregation of the message")
		return
	}

	for _, m := range batch {
		c.process(ctx, m)
	}

	c.Acknowledge(ctx, msg)

	duration := c.clock.Now().Sub(start)

	atomic.AddInt32(&c.processed, int32(len(batch)))
	c.mw.Write(mon.MetricData{
		&mon.MetricDatum{
			MetricName: metricNameConsumerProcessedCount,
			Value:      float64(len(batch)),
		},
		&mon.MetricDatum{
			MetricName: metricNameConsumerDuration,
			Value:      float64(duration.Milliseconds()),
		},
	})
}

func (c *Consumer) processSingleMessage(ctx context.Context, msg *Message) {
	start := c.clock.Now()

	ack := c.process(ctx, msg)

	if !ack {
		return
	}

	c.Acknowledge(ctx, msg)

	duration := c.clock.Now().Sub(start)

	atomic.AddInt32(&c.processed, 1)
	c.mw.Write(mon.MetricData{
		&mon.MetricDatum{
			MetricName: metricNameConsumerProcessedCount,
			Value:      1.0,
		},
		&mon.MetricDatum{
			MetricName: metricNameConsumerDuration,
			Value:      float64(duration.Milliseconds()),
		},
	})
}

func (c *Consumer) process(ctx context.Context, msg *Message) bool {
	defer c.recover()

	model := c.callback.GetModel(msg.Attributes)

	ctx, attributes, err := c.encoder.Decode(ctx, msg, model)

	if err != nil {
		c.logger.WithContext(ctx).Error(err, "an error occurred during the consume operation")
		return false
	}

	ctx, span := c.tracer.StartSpanFromContext(ctx, c.id)
	defer span.Finish()

	ack, err := c.callback.Consume(ctx, model, attributes)

	if err != nil {
		// one could think that we should just initialize this logger once, but the ctx used
		// in the other error case might be in fact different and if we use the wrong context,
		// we miss a trace id in the logs later on
		c.logger.WithContext(ctx).Error(err, "an error occurred during the consume operation")
	}

	return ack
}
