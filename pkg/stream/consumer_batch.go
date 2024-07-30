package stream

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type BatchConsumerCallbackFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (BatchConsumerCallback, error)

//go:generate mockery --name=BatchConsumerCallback
type BatchConsumerCallback interface {
	BaseConsumerCallback
	Consume(ctx context.Context, models []interface{}, attributes []map[string]string) ([]bool, error)
}

//go:generate mockery --name=RunnableBatchConsumerCallback
type RunnableBatchConsumerCallback interface {
	BatchConsumerCallback
	RunnableCallback
}

type BatchConsumer struct {
	*baseConsumer
	batch    []*consumerData
	callback BatchConsumerCallback
	ticker   *time.Ticker
	settings *BatchConsumerSettings
}

func NewBatchConsumer(name string, callbackFactory BatchConsumerCallbackFactory) func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		loggerCallback := logger.WithChannel("consumerCallback")
		contextEnforcingLogger := log.NewContextEnforcingLogger(loggerCallback)

		callback, err := callbackFactory(ctx, config, contextEnforcingLogger)
		if err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		contextEnforcingLogger.Enable()

		settings := &BatchConsumerSettings{}
		key := ConfigurableConsumerKey(name)
		config.UnmarshalKey(key, settings)

		ticker := time.NewTicker(settings.IdleTimeout)

		baseConsumer, err := NewBaseConsumer(ctx, config, logger, name, callback)
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
	return c.baseConsumer.run(kernelCtx, c.readFromInput)
}

func (c *BatchConsumer) readFromInput(ctx context.Context) error {
	logger := c.logger.WithContext(ctx)
	defer logger.Debug("run is ending")
	defer c.wg.Done()
	defer c.processBatch(context.Background())

	for {
		force := false

		select {
		case <-ctx.Done():
			return fmt.Errorf("return from consuming as the coffin is dying")

		case cdata, ok := <-c.data:
			if !ok {
				return nil
			}

			if _, ok := cdata.msg.Attributes[AttributeAggregate]; ok {
				c.processAggregateMessage(ctx, cdata)
			} else {
				c.processSingleMessage(ctx, cdata)
			}

		case <-c.ticker.C:
			force = true
		}

		if len(c.batch) >= c.settings.BatchSize || force {
			c.processBatch(ctx)
		}
	}
}

func (c *BatchConsumer) processAggregateMessage(ctx context.Context, cdata *consumerData) {
	batch := make([]*Message, 0)
	var err error

	ctx, _, err = c.encoder.Decode(ctx, cdata.msg, &batch)
	if err != nil {
		c.logger.WithContext(ctx).Error("an error occurred during disaggregation of the message: %w", err)

		return
	}

	for _, msg := range batch {
		c.batch = append(c.batch, &consumerData{
			msg:   msg,
			input: cdata.input,
		})
	}
}

func (c *BatchConsumer) processSingleMessage(_ context.Context, cdata *consumerData) {
	c.batch = append(c.batch, cdata)
}

func (c *BatchConsumer) processBatch(ctx context.Context) {
	batch := c.batch

	c.batch = make([]*consumerData, 0, c.settings.BatchSize)
	c.ticker.Stop()
	c.ticker = time.NewTicker(c.settings.IdleTimeout)

	c.consumeBatch(ctx, batch)
}

func (c *BatchConsumer) consumeBatch(ctx context.Context, batch []*consumerData) {
	defer c.recover(ctx, nil)

	start := c.clock.Now()

	// make sure to create new context as we can't rely on the tracer to create a new one
	batchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var span tracing.Span
	batchCtx, span = c.tracer.StartSpanFromContext(batchCtx, "stream.consumeBatch")
	defer span.Finish()

	if len(batch) == 0 {
		return
	}

	batch, models, attributes, subSpans := c.decodeMessages(batchCtx, batch)
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

	if len(batch) != len(acks) {
		logger.Error("number of acks does not match number of messages in batch: %d != %d", len(acks), len(batch))
	}

	ackMessages := make([]*consumerData, 0, len(batch))
	for i, ack := range acks {
		ackMessages = append(ackMessages, batch[i])
		if !ack && !c.hasNativeRetry() {
			c.retry(batchCtx, batch[i].msg)
		}
	}

	c.AcknowledgeBatch(batchCtx, ackMessages, acks)

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, int32(len(ackMessages)))

	c.writeMetricDurationAndProcessedCount(duration, len(batch))
}

func (c *BatchConsumer) decodeMessages(batchCtx context.Context, batch []*consumerData) ([]*consumerData, []interface{}, []map[string]string, []tracing.Span) {
	models := make([]interface{}, 0, len(batch))
	attributes := make([]map[string]string, 0, len(batch))
	spans := make([]tracing.Span, 0, len(batch))
	newBatch := make([]*consumerData, 0, len(batch))

	for _, cdata := range batch {
		model := c.callback.GetModel(cdata.msg.Attributes)

		msgCtx, attribute, err := c.encoder.Decode(batchCtx, cdata.msg, model)
		if err != nil {
			c.logger.WithContext(msgCtx).Error("an error occurred during the batch decode message operation: %w", err)

			continue
		}

		models = append(models, model)
		attributes = append(attributes, attribute)
		newBatch = append(newBatch, cdata)

		_, span := c.tracer.StartSubSpan(msgCtx, c.id)
		spans = append(spans, span)
	}

	return newBatch, models, attributes, spans
}
