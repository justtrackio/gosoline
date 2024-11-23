package stream

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
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

type BatchConsumerSettings struct {
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
	BatchSize   int           `cfg:"batch_size" default:"1"`
}

type BatchConsumer struct {
	*baseConsumer
	batch    []*consumerData
	callback BatchConsumerCallback
	lck      sync.Mutex
	ticker   clock.Ticker
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

		baseConsumer, err := NewBaseConsumer(ctx, config, logger, name, callback)
		if err != nil {
			return nil, fmt.Errorf("can not initiate base consumer: %w", err)
		}

		ticker := baseConsumer.clock.NewTicker(settings.IdleTimeout)

		batchConsumer := NewBatchConsumerWithInterfaces(baseConsumer, callback, ticker, settings)

		return batchConsumer, nil
	}
}

func NewBatchConsumerWithInterfaces(base *baseConsumer, callback BatchConsumerCallback, ticker clock.Ticker, settings *BatchConsumerSettings) *BatchConsumer {
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
		c.lck.Lock()
		ticker := c.ticker
		c.lck.Unlock()

		select {
		case <-ctx.Done():
			return fmt.Errorf("return from consuming as the coffin is dying")

		case cdata, ok := <-c.data:
			if !ok {
				return nil
			}

			c.lck.Lock()
			if _, ok := cdata.msg.Attributes[AttributeAggregate]; ok {
				c.processAggregateMessage(ctx, cdata)
			} else {
				c.processSingleMessage(ctx, cdata)
			}
			c.lck.Unlock()

		case <-ticker.Chan():
			force = true
		}

		c.lck.Lock()
		batchLen := len(c.batch)
		c.lck.Unlock()

		if batchLen >= c.settings.BatchSize || force {
			c.processBatch(ctx)
		}
	}
}

func (c *BatchConsumer) processAggregateMessage(ctx context.Context, cdata *consumerData) {
	batch := make([]*Message, 0)
	var err error

	ctx, _, err = c.encoder.Decode(ctx, &cdata.msg, &batch)
	if err != nil {
		c.logger.WithContext(ctx).Error("an error occurred during disaggregation of the message: %w", err)

		return
	}

	for _, msg := range batch {
		c.batch = append(c.batch, &consumerData{
			msg: *msg,
			originalMessage: &originalMessage{
				Message: cdata.msg,
				id:      c.uuidGen.NewV4(),
			},
			src:   cdata.src,
			input: cdata.input,
		})
	}
}

func (c *BatchConsumer) processSingleMessage(_ context.Context, cdata *consumerData) {
	c.batch = append(c.batch, cdata)
}

func (c *BatchConsumer) processBatch(ctx context.Context) {
	c.lck.Lock()
	batch := c.batch

	c.batch = make([]*consumerData, 0, c.settings.BatchSize)
	c.ticker.Stop()
	c.ticker = c.clock.NewTicker(c.settings.IdleTimeout)
	c.lck.Unlock()

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

	delayedContext, stop := exec.WithDelayedCancelContext(batchCtx, time.Second*10)
	defer stop()

	ackMessages := make([]*consumerData, 0, len(batch))
	for i, ack := range acks {
		ackMessages = append(ackMessages, batch[i])
		if !ack && (!c.hasNativeRetry() || batch[i].originalMessage != nil) {
			c.retry(delayedContext, &batch[i].msg)
		}

		// acknowledge the message as we will retry the de-aggregated message
		if batch[i].originalMessage != nil {
			acks[i] = true
		}
	}

	c.AcknowledgeBatch(delayedContext, ackMessages, acks)

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

		msgCtx, attribute, err := c.encoder.Decode(batchCtx, &cdata.msg, model)
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

// TODO tests:
// - batch consumer
// - batch consumer with aggregate message
// - batch consumer with multiple runners
// - batch consumer with cancel while processing
// - consumer with cancel while processing
