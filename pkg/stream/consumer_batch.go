package stream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type UntypedBatchConsumerCallbackFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (UntypedBatchConsumerCallback, error)

//go:generate go run github.com/vektra/mockery/v2 --name=UntypedBatchConsumerCallback
type UntypedBatchConsumerCallback interface {
	BaseConsumerCallback
	Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name=RunnableUntypedBatchConsumerCallback
type RunnableUntypedBatchConsumerCallback interface {
	UntypedBatchConsumerCallback
	RunnableCallback
}

type BatchConsumerSettings struct {
	IdleTimeout      time.Duration `cfg:"idle_timeout" default:"10s"`
	BatchSize        int           `cfg:"batch_size" default:"1"`
	ConsumeGraceTime time.Duration `cfg:"consume_grace_time" default:"10s"`
}

type BatchConsumer struct {
	*baseConsumer
	batch []*consumerData
	// aggregates holds the messages the current batch got expanded from. They have to be acknowledged as well once
	// the batch was consumed, otherwise inputs which track the completion of their messages would stall.
	aggregates []*consumerData
	callback   UntypedBatchConsumerCallback
	ticker     *time.Ticker
	settings   *BatchConsumerSettings
}

func NewUntypedBatchConsumer(name string, callbackFactory UntypedBatchConsumerCallbackFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		loggerCallback := logger.WithChannel("consumerCallback")

		callback, err := callbackFactory(ctx, config, loggerCallback)
		if err != nil {
			return nil, fmt.Errorf("can not initiate callback for consumer %s: %w", name, err)
		}

		settings := &BatchConsumerSettings{}
		key := ConfigurableConsumerKey(name)
		if err := config.UnmarshalKey(key, settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch consumer settings for key %q in NewUntypedBatchConsumer: %w", key, err)
		}

		ticker := time.NewTicker(settings.IdleTimeout)

		baseConsumer, err := NewBaseConsumer(ctx, config, logger, name, callback)
		if err != nil {
			return nil, fmt.Errorf("can not initiate base consumer: %w", err)
		}

		batchConsumer := NewUntypedBatchConsumerWithInterfaces(baseConsumer, callback, ticker, settings)

		return batchConsumer, nil
	}
}

func NewUntypedBatchConsumerWithInterfaces(base *baseConsumer, callback UntypedBatchConsumerCallback, ticker *time.Ticker, settings *BatchConsumerSettings) *BatchConsumer {
	consumer := &BatchConsumer{
		baseConsumer: base,
		callback:     callback,
		ticker:       ticker,
		settings:     settings,
	}

	return consumer
}

func (c *BatchConsumer) Run(kernelCtx context.Context) error {
	return c.run(kernelCtx, c.readFromInput)
}

func (c *BatchConsumer) readFromInput(ctx context.Context) error {
	defer c.logger.Debug(ctx, "run is ending")
	defer c.wg.Done()

	consumeCtx, stop := exec.WithDelayedCancelContext(ctx, c.settings.ConsumeGraceTime)
	defer stop()
	defer c.processBatch(consumeCtx)

	for {
		force := false

		select {
		case cdata, ok := <-c.data:
			if !ok {
				return nil
			}

			if _, ok := cdata.msg.Attributes[AttributeAggregate]; ok {
				c.processAggregateMessage(consumeCtx, cdata)
			} else {
				c.processSingleMessage(consumeCtx, cdata)
			}

		case <-c.ticker.C:
			force = true
		}

		if len(c.batch) >= c.settings.BatchSize || force {
			c.processBatch(consumeCtx)
		}
	}
}

func (c *BatchConsumer) processAggregateMessage(ctx context.Context, cdata *consumerData) {
	batch := make([]*Message, 0)
	var err error

	ctx, _, err = c.encoder.Decode(ctx, cdata.msg, &batch)
	if err != nil {
		c.logger.Error(ctx, "an error occurred during disaggregation of the message: %w", err)

		// we can not do anything with this message, but we still have to acknowledge it negatively so an input
		// which tracks the completion of its messages does not stall
		c.Acknowledge(ctx, cdata, false)

		return
	}

	for _, msg := range batch {
		// the disaggregated messages have no identity on the transport itself (an SQS message from an aggregate has
		// no receipt handle of its own, a Kafka record is a single record regardless of how many messages it holds),
		// so they must never be acknowledged - only the aggregate they came from is
		c.batch = append(c.batch, &consumerData{
			msg: msg,
			src: cdata.src,
		})
	}

	// the aggregate itself is acknowledged once the batch containing its messages was consumed. All messages of an
	// aggregate are appended in one go and can therefore never be split across two batches.
	c.aggregates = append(c.aggregates, cdata)
}

func (c *BatchConsumer) processSingleMessage(_ context.Context, cdata *consumerData) {
	c.batch = append(c.batch, cdata)
}

func (c *BatchConsumer) processBatch(ctx context.Context) {
	batch := c.batch
	aggregates := c.aggregates

	c.batch = make([]*consumerData, 0, c.settings.BatchSize)
	c.aggregates = nil
	c.ticker.Stop()
	c.ticker = time.NewTicker(c.settings.IdleTimeout)

	c.consumeBatch(ctx, batch, aggregates)
}

// consumeBatch processes the given batch and acknowledges every message of it exactly once - including the aggregate
// messages it was expanded from, the messages we could not decode, and the cases where we return early or panic.
// Inputs which track the completion of their messages (like Kafka, which may only commit an offset once the
// corresponding record was processed) would stall otherwise.
func (c *BatchConsumer) consumeBatch(ctx context.Context, batch []*consumerData, aggregates []*consumerData) {
	acknowledger := newBatchAcknowledger(c, batch, aggregates)

	// registered before the recover below so that it runs afterwards and acknowledges everything we did not get to
	defer acknowledger.acknowledgeRemaining(ctx)
	defer c.recover(ctx, nil)

	start := c.clock.Now()

	// make sure to create new context as we can't rely on the tracer to create a new one
	batchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var span tracing.Span
	batchCtx, span = c.tracer.StartSpanFromContext(batchCtx, "stream.consumeBatch")
	defer span.Finish()

	if len(batch) == 0 {
		acknowledger.acknowledge(batchCtx, aggregates, funk.Repeat(true, len(aggregates)))

		return
	}

	decoded := c.decodeMessages(batchCtx, batch)
	defer func() {
		for i := range decoded.spans {
			decoded.spans[i].Finish()
		}
	}()

	acknowledger.acknowledge(batchCtx, decoded.skipped, decoded.skippedAcks)

	acks, err := c.callback.Consume(batchCtx, decoded.models, decoded.attributes)
	if err != nil {
		c.logger.Error(batchCtx, "an error occurred during the consume batch operation: %w", err)
	}
	if batchCtx.Err() != nil {
		c.logger.Error(ctx, "consumer %s did not finish processing a batch within the consume grace time of %s", c.name, c.settings.ConsumeGraceTime)
	}

	acks = c.alignAcks(batchCtx, acks, len(decoded.batch))

	ackMessages := make([]*consumerData, 0, len(decoded.batch)+len(aggregates))
	ackValues := make([]bool, 0, len(decoded.batch)+len(aggregates))
	for i, ack := range acks {
		ackMessages = append(ackMessages, decoded.batch[i])
		ackValues = append(ackValues, ack)

		if !ack && !c.hasNativeRetry() {
			c.retry(batchCtx, decoded.batch[i].msg)
		}
	}

	// acknowledge the aggregates together with the messages they were expanded from, so we only need a single call
	ackMessages = append(ackMessages, aggregates...)
	ackValues = append(ackValues, funk.Repeat(true, len(aggregates))...)

	acknowledger.acknowledge(batchCtx, ackMessages, ackValues)

	duration := c.clock.Now().Sub(start)
	atomic.AddInt32(&c.processed, int32(len(decoded.batch)))

	c.writeMetricDurationAndProcessedCount(batchCtx, duration, len(decoded.batch))
}

// alignAcks makes sure we have exactly as many acks as we have messages in the batch, defaulting to a negative
// acknowledgement for the ones a misbehaving callback did not report.
func (c *BatchConsumer) alignAcks(ctx context.Context, acks []bool, size int) []bool {
	if len(acks) == size {
		return acks
	}

	c.logger.Error(ctx, "number of acks does not match number of messages in batch: %d != %d", len(acks), size)

	for len(acks) < size {
		acks = append(acks, false)
	}

	return acks[:size]
}

// batchAcknowledger makes sure every message of a batch gets acknowledged exactly once, no matter which of the many
// exits of consumeBatch we take.
type batchAcknowledger struct {
	consumer *BatchConsumer
	pending  map[*consumerData]struct{}
}

func newBatchAcknowledger(consumer *BatchConsumer, groups ...[]*consumerData) *batchAcknowledger {
	pending := make(map[*consumerData]struct{})
	for _, group := range groups {
		for _, cdata := range group {
			pending[cdata] = struct{}{}
		}
	}

	return &batchAcknowledger{
		consumer: consumer,
		pending:  pending,
	}
}

// acknowledge acknowledges all of the given messages which have not been acknowledged yet.
func (a *batchAcknowledger) acknowledge(ctx context.Context, cdata []*consumerData, acks []bool) {
	msgs := make([]*consumerData, 0, len(cdata))
	msgAcks := make([]bool, 0, len(cdata))

	for i := range cdata {
		if _, ok := a.pending[cdata[i]]; !ok {
			continue
		}

		delete(a.pending, cdata[i])
		msgs = append(msgs, cdata[i])
		msgAcks = append(msgAcks, acks[i])
	}

	if len(msgs) > 0 {
		a.consumer.AcknowledgeBatch(ctx, msgs, msgAcks)
	}
}

// acknowledgeRemaining negatively acknowledges every message we did not get to, for example because we returned early
// or panicked while consuming the batch.
func (a *batchAcknowledger) acknowledgeRemaining(ctx context.Context) {
	if len(a.pending) == 0 {
		return
	}

	msgs := make([]*consumerData, 0, len(a.pending))
	for cdata := range a.pending {
		msgs = append(msgs, cdata)
	}

	a.acknowledge(ctx, msgs, make([]bool, len(msgs)))
}

// decodedBatch holds the messages of a batch which could be decoded, alongside the ones we had to skip and the
// acknowledgement each of those skipped messages should get.
type decodedBatch struct {
	batch       []*consumerData
	models      []any
	attributes  []map[string]string
	spans       []tracing.Span
	skipped     []*consumerData
	skippedAcks []bool
}

func (d *decodedBatch) skip(cdata *consumerData, ack bool) {
	d.skipped = append(d.skipped, cdata)
	d.skippedAcks = append(d.skippedAcks, ack)
}

func (c *BatchConsumer) decodeMessages(batchCtx context.Context, batch []*consumerData) decodedBatch {
	decoded := decodedBatch{
		batch:      make([]*consumerData, 0, len(batch)),
		models:     make([]any, 0, len(batch)),
		attributes: make([]map[string]string, 0, len(batch)),
		spans:      make([]tracing.Span, 0, len(batch)),
	}

	for _, cdata := range batch {
		model, err := c.callback.GetModel(cdata.msg.Attributes)
		if err != nil {
			c.metricWriter.Write(batchCtx, metric.Data{
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
			if errors.As(err, &ignorableErr) && ignorableErr.IsIgnorableWithSettings(c.baseConsumer.settings.IgnoreOnGetModelError) {
				c.logger.Info(batchCtx, "ignoring message due to ignorable GetModel error: %s", err.Error())

				// the message is skipped on purpose, so acknowledge it positively
				decoded.skip(cdata, true)

				continue
			}

			c.logger.Error(batchCtx, "an error occurred during the batch GetModel operation: %w", err)
			decoded.skip(cdata, false)

			continue
		}

		msgCtx, attribute, err := c.encoder.Decode(batchCtx, cdata.msg, model)
		if err != nil {
			c.logger.Error(msgCtx, "an error occurred during the batch decode message operation: %w", err)
			decoded.skip(cdata, false)

			continue
		}

		decoded.models = append(decoded.models, model)
		decoded.attributes = append(decoded.attributes, attribute)
		decoded.batch = append(decoded.batch, cdata)

		_, span := c.tracer.StartSubSpan(msgCtx, c.id)
		decoded.spans = append(decoded.spans, span)
	}

	return decoded
}
