package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/tracing"
	"sync/atomic"
	"time"
)

type BatchConsumerCallbackFactory func(config cfg.Config, logger log.Logger) (BatchConsumerCallback, error)

//go:generate mockery --name=BatchConsumerCallback
type BatchConsumerCallback interface {
	BaseConsumerCallback
	Consume(ctx context.Context, models []interface{}, attributes []map[string]interface{}) ([]bool, error)
}

//go:generate mockery --name=RunnableBatchConsumerCallback
type RunnableBatchConsumerCallback interface {
	BatchConsumerCallback
	RunnableCallback
}

type BatchConsumerSettings struct {
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
	BatchSize   int           `cfg:"batch_size" default:"1"`
}

type BatchConsumer struct {
	*baseConsumer
	batch    []*Message
	callback BatchConsumerCallback
	ticker   *time.Ticker
	settings *BatchConsumerSettings
}

func NewBatchConsumer(name string, callbackFactory BatchConsumerCallbackFactory) func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		loggerCallback := logger.WithChannel("consumerCallback")
		contextEnforcingLogger := log.NewContextEnforcingLogger(loggerCallback)

		callback, err := callbackFactory(config, contextEnforcingLogger)

		if err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		contextEnforcingLogger.Enable()

		settings := &BatchConsumerSettings{}
		key := ConfigurableConsumerKey(name)
		config.UnmarshalKey(key, settings)

		ticker := time.NewTicker(settings.IdleTimeout)

		baseConsumer, err := NewBaseConsumer(config, logger, name, callback)
		if err != nil {
			return nil, fmt.Errorf("can not initiate base consumer: %w", err)
		}

		batchConsumer := NewBatchConsumerWithInterfaces(baseConsumer, callback, ticker, settings)

		return batchConsumer, nil
	}
}

func NewBatchConsumerWithInterfaces(base *baseConsumer, callback BatchConsumerCallback, ticker *time.Ticker, settings *BatchConsumerSettings) *BatchConsumer {
	consumer := &BatchConsumer{
		baseConsumer: base,
		callback:     callback,
		ticker:       ticker,
		settings:     settings,
	}

	return consumer
}

func (c *BatchConsumer) Run(kernelCtx context.Context) error {
	return c.baseConsumer.run(kernelCtx, c.run)
}

func (c *BatchConsumer) run(ctx context.Context) error {
	logger := c.logger.WithContext(ctx)
	defer logger.Debug("run is ending")
	defer c.wg.Done()
	defer c.processBatch(context.Background())

	for {
		force := false

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

		case <-c.ticker.C:
			force = true
		}

		if len(c.batch) >= c.settings.BatchSize || force {
			c.processBatch(ctx)
		}
	}
}

func (c *BatchConsumer) processAggregateMessage(ctx context.Context, msg *Message) {
	batch := make([]*Message, 0)
	var err error

	ctx, _, err = c.encoder.Decode(ctx, msg, &batch)

	if err != nil {
		c.logger.WithContext(ctx).Error("an error occurred during disaggregation of the message: %w", err)
		return
	}

	c.batch = append(c.batch, batch...)
}

func (c *BatchConsumer) processSingleMessage(_ context.Context, msg *Message) {
	c.batch = append(c.batch, msg)
}

func (c *BatchConsumer) processBatch(ctx context.Context) {
	batch := c.batch

	c.batch = make([]*Message, 0, c.settings.BatchSize)
	c.ticker.Stop()
	c.ticker = time.NewTicker(c.settings.IdleTimeout)

	c.consumeBatch(ctx, batch)
}

func (c *BatchConsumer) consumeBatch(kernelCtx context.Context, batch []*Message) {
	defer c.recover()

	start := c.clock.Now()

	// make sure to create new context as we can't rely on the tracer to create a new one
	batchCtx, cancel := context.WithCancel(kernelCtx)
	defer cancel()

	var span tracing.Span
	batchCtx, span = c.tracer.StartSpanFromContext(batchCtx, "stream.consumeBatch")
	defer span.Finish()

	if len(batch) == 0 {
		return
	}

	messages, models, attributes, subSpans := c.decodeMessages(batchCtx, batch)
	defer func() {
		for i := range subSpans {
			subSpans[i].Finish()
		}
	}()

	logger := c.logger.WithContext(batchCtx)

	acks, err := c.callback.Consume(batchCtx, models, attributes)
	if err != nil {
		logger.Error("an error occurred during the consume batch operation: %w", err)
	}

	if len(messages) != len(acks) {
		logger.Error("number of acks does not match number of messages in batch: %d != %d", len(acks), len(messages))
	}

	ackMessages := make([]*Message, 0, len(messages))
	for i, ack := range acks {
		if ack {
			ackMessages = append(ackMessages, messages[i])
		}
	}

	c.AcknowledgeBatch(batchCtx, ackMessages)

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, int32(len(ackMessages)))

	c.writeMetrics(duration, len(batch))
}

func (c *BatchConsumer) decodeMessages(batchCtx context.Context, batch []*Message) ([]*Message, []interface{}, []map[string]interface{}, []tracing.Span) {
	models := make([]interface{}, 0, len(batch))
	attributes := make([]map[string]interface{}, 0, len(batch))
	spans := make([]tracing.Span, 0, len(batch))
	newBatch := make([]*Message, 0, len(batch))

	for _, msg := range batch {
		model := c.callback.GetModel(msg.Attributes)

		msgCtx, attribute, err := c.encoder.Decode(batchCtx, msg, model)
		if err != nil {
			c.logger.WithContext(msgCtx).Error("an error occurred during the batch decode message operation: %w", err)
			continue
		}

		models = append(models, model)
		attributes = append(attributes, attribute)
		newBatch = append(newBatch, msg)

		_, span := c.tracer.StartSubSpan(msgCtx, c.id)
		spans = append(spans, span)
	}

	return newBatch, models, attributes, spans
}
