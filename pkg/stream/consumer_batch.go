/*
Batch consumer flow

The shared consumerBase initializes the callback and runs the primary input,
retry input, and batch loop. Both inputs feed a bounded admission channel;
one loop owns the batch slice and processes callbacks serially.

Primary input -> enqueue -> bounded channel -> collect -> Consume(batch)

	|                                         |
	+-> acknowledge admission                 +-> failures -> retry queue

Retry input   -> enqueueRetry -> same channel -> process retry envelope

	|                                      |
	+---------- wait for result <----------+
	            success: acknowledge
	            failure: transport redelivery / DLQ

Primary records flush on batch_size or idle_timeout. Aggregate envelopes flatten
into child records before decoding. Retry deliveries first flush pending primary
work, then process immediately so a single retry runner can make progress.
Failed primary records go to the configured retry handler when enabled (SQS by
default); failed retry deliveries remain unacknowledged rather than being requeued.

On shutdown, inputs stop fetching. Once both inputs return, the admission channel
closes and buffered work is drained within the shared consumer grace deadline.
Failed primary work can still be persisted to retry; unresolved final failures
return an error. Admission acknowledgements precede processing, so the in-memory
buffer is not durable.
*/
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
)

// UntypedBatchConsumerCallback decodes models and processes a batch, returning
// one success decision per model. False decisions request a configured retry.
//
//go:generate go run github.com/vektra/mockery/v2 --name UntypedBatchConsumerCallback
type UntypedBatchConsumerCallback interface {
	GetModel(attributes map[string]string) (any, error)
	Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error)
}

// RunnableUntypedBatchConsumerCallback additionally runs alongside the batch loop.
//
//go:generate go run github.com/vektra/mockery/v2 --name RunnableUntypedBatchConsumerCallback
type RunnableUntypedBatchConsumerCallback interface {
	UntypedBatchConsumerCallback
	RunnableCallback
}

// UntypedBatchConsumerCallbackFactory constructs an untyped batch callback.
type UntypedBatchConsumerCallbackFactory func(context.Context, cfg.Config, log.Logger) (UntypedBatchConsumerCallback, error)

// BatchConsumerSettings controls collection. Zero BufferSize uses BatchSize.
// Idle timeout, processing grace and retry settings remain under the consumer.
type BatchConsumerSettings struct {
	BatchSize  int `cfg:"batch_size" default:"1" validate:"min=1"`
	BufferSize int `cfg:"buffer_size" validate:"min=0"`
}

type batchMessage struct {
	ctx     context.Context
	message *Message
	result  chan bool
}

// BatchConsumer acknowledges transport messages when admitted to memory, then
// processes size/time-triggered batches serially. It does not provide durable
// buffering: crashes, exhausted retries or incomplete drains require replay.
type BatchConsumer struct {
	*consumerBase
	batchCallback UntypedBatchConsumerCallback
	batchSettings BatchConsumerSettings
	data          chan batchMessage
	batch         []batchMessage
}

var _ kernel.FullModule = &BatchConsumer{}

// NewUntypedBatchConsumer creates a channel-driven batch consumer using the
// existing retry handlers (SQS by default).
func NewUntypedBatchConsumer(name string, callbackFactory UntypedBatchConsumerCallbackFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var callback UntypedBatchConsumerCallback
		var settings ConsumerSettings
		var base *consumerBase

		if callback, err = callbackFactory(ctx, config, logger.WithChannel("consumerCallback")); err != nil {
			return nil, fmt.Errorf("can not initiate batch callback for %s: %w", name, err)
		}

		if settings, err = ReadConsumerSettings(config, name); err != nil {
			return nil, err
		}

		var batchSettings BatchConsumerSettings
		if err := config.UnmarshalKey(ConfigurableConsumerKey(name), &batchSettings); err != nil {
			return nil, fmt.Errorf("can not read batch settings for %s: %w", name, err)
		}

		if settings.IdleTimeout <= 0 || settings.GraceTime <= 0 {
			return nil, fmt.Errorf("invalid batch consumer timeouts or retry settings for %s", name)
		}

		schemaCallback, _ := callback.(SchemaSettingsAwareCallback)

		if base, err = newConsumerBase(ctx, config, logger, name, schemaCallback, settings, newConsumerBatchRetryHandler); err != nil {
			return nil, fmt.Errorf("can not initiate batch consumer: %w", err)
		}

		return NewUntypedBatchConsumerWithInterfaces(base, callback, batchSettings)
	}
}

// NewUntypedBatchConsumerWithInterfaces constructs a batch consumer using the
// shared dependencies of an unstarted consumer. The supplied consumer must not
// subsequently be run: both instances share the same base. Its retry
// handler must support retrying admitted primary messages independently of their
// original transport acknowledgements. Invalid batch settings return an error.
func NewUntypedBatchConsumerWithInterfaces(base *consumerBase, callback UntypedBatchConsumerCallback, settings BatchConsumerSettings) (*BatchConsumer, error) {
	if settings.BatchSize < 1 || settings.BufferSize < 0 {
		return nil, fmt.Errorf("invalid batch consumer settings: batch size must be at least 1 and buffer size must be non-negative (batch size: %d, buffer size: %d)", settings.BatchSize, settings.BufferSize)
	}

	if settings.BufferSize == 0 {
		settings.BufferSize = settings.BatchSize
	}

	batch := &BatchConsumer{
		consumerBase:  base,
		batchCallback: callback,
		batchSettings: settings,
		data:          make(chan batchMessage, settings.BufferSize),
	}

	base.setCallbackHooks(callback)
	base.run = batch.runCallback
	base.inputProcess = batch.enqueue
	base.retryProcess = batch.enqueueRetry
	base.inputsFinished = func() { close(batch.data) }

	return batch, nil
}

// enqueue acknowledges a primary message once it enters the buffer, before processing.
func (c *BatchConsumer) enqueue(ctx context.Context, msg *Message) bool {
	return c.admit(ctx, msg, nil)
}

// enqueueRetry buffers a retry message and waits for its processing result or the drain deadline.
func (c *BatchConsumer) enqueueRetry(ctx context.Context, msg *Message) bool {
	result := make(chan bool, 1)
	if !c.admit(ctx, msg, result) {
		return false
	}

	c.writeMetricRetryCount(ctx, metricNameConsumerRetryGetCount)
	select {
	case ack := <-result:
		return ack
	case <-c.drainCtx.Done():
		return false
	}
}

// admit copies a message into the bounded channel, blocking for space until the drain deadline.
func (c *BatchConsumer) admit(ctx context.Context, msg *Message, result chan bool) bool {
	if c.drainCtx.Err() != nil {
		return false
	}

	copyMsg := *msg
	copyMsg.Attributes = maps.Clone(msg.Attributes)

	select {
	case c.data <- batchMessage{
		ctx:     context.WithoutCancel(ctx),
		message: &copyMsg,
		result:  result,
	}:
		return true
	case <-c.drainCtx.Done():
		c.handleError(ctx, c.drainCtx.Err(), "could not admit message to batch buffer")

		return false
	}
}

// runBatches serially collects messages, flushes on size or time, and drains on shutdown.
func (c *BatchConsumer) runBatches(ctx context.Context) error {
	// Processing shares the consumer's shutdown deadline, not the lifetime of an
	// admission callback or a fresh grace timer for every batch.
	processCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	stop := context.AfterFunc(c.drainCtx, cancel)
	defer stop()

	ticker := c.clock.NewTicker(c.settings.IdleTimeout)
	defer func() { ticker.Stop() }()

	flush := func() {
		ticker.Stop()
		ticker = c.clock.NewTicker(c.settings.IdleTimeout)
		c.consumeBatch(processCtx)
	}

	for {
		select {
		case message, ok := <-c.data:
			if !ok {
				return c.drainBatch(processCtx)
			}

			c.collectMessage(processCtx, message)

			if len(c.batch) >= c.batchSettings.BatchSize {
				flush()
			}
		case <-ticker.Chan():
			flush()
		case <-ctx.Done():
			// The lifecycle cancels this context after both inputs return. Their
			// final admitted records are already in the channel and must be drained.
			return c.drainBatch(processCtx)
		}
	}
}

// collectMessage collects primary work or flushes and processes a retry envelope immediately.
// It reports retry success to the waiting input and returns unresolved primary failures.
func (c *BatchConsumer) collectMessage(ctx context.Context, message batchMessage) int {
	if message.result == nil {
		return c.collect(ctx, message)
	}

	failed := c.consumeBatch(ctx)
	retryFailed := c.collect(ctx, message) + c.consumeBatch(ctx)
	message.result <- retryFailed == 0

	return failed
}

// collect appends a record or an aggregate's children, retrying malformed envelopes.
func (c *BatchConsumer) collect(ctx context.Context, message batchMessage) int {
	var err error
	var childCtx context.Context

	if _, aggregate := message.message.Attributes[AttributeAggregate]; !aggregate {
		c.batch = append(c.batch, message)

		return 0
	}

	var children []*Message
	copyMsg := *message.message
	copyMsg.Attributes = maps.Clone(copyMsg.Attributes)

	if childCtx, _, err = c.encoder.Decode(message.ctx, &copyMsg, &children); err != nil {
		c.handleError(ctx, err, "could not disaggregate batch message")

		if !c.retryBatchMessage(ctx, message) {
			return 1
		}

		return 0
	}

	for _, child := range children {
		if child == nil {
			c.handleError(ctx, fmt.Errorf("nil aggregate child"), "could not disaggregate batch message")

			if !c.retryBatchMessage(ctx, message) {
				return 1
			}

			return 0
		}
	}

	for _, child := range children {
		c.batch = append(c.batch, batchMessage{ctx: childCtx, message: child, result: message.result})
	}

	return 0
}

// drainBatch processes remaining buffered work and reports primary failures requiring replay.
func (c *BatchConsumer) drainBatch(ctx context.Context) error {
	failed := 0

	for message := range c.data {
		failed += c.collectMessage(ctx, message)

		if len(c.batch) >= c.batchSettings.BatchSize {
			failed += c.consumeBatch(ctx)
		}
	}

	failed += c.consumeBatch(ctx)
	if failed > 0 {
		return fmt.Errorf("batch consumer stopped with %d unsuccessful acknowledged messages requiring replay", failed)
	}

	return nil
}

// consumeBatch decodes and processes the collected batch, routes failures, and records telemetry.
// It returns failures not handed to retry, including unsuccessful retry-input records.
func (c *BatchConsumer) consumeBatch(ctx context.Context) (failed int) {
	var err error
	var acks []bool

	batch := c.batch
	c.batch = nil

	if len(batch) == 0 {
		return 0
	}

	processingID := c.processingSequence.Add(1)
	start := c.clock.Now()

	c.processingStartedAt.Put(processingID, start)
	defer c.processingStartedAt.Remove(processingID)

	batchCtx, cancel := context.WithCancel(batch[0].ctx)
	defer cancel()

	stop := context.AfterFunc(ctx, cancel)
	defer stop()

	if ctx.Err() != nil {
		cancel()
	}

	ctx = batchCtx
	ctx, span := c.startTracingContext(ctx)
	defer span.Finish()
	defer func() {
		c.processed.Add(int32(len(batch)))
		c.writeMetricDurationAndProcessedCount(ctx, c.clock.Since(start), len(batch))
	}()

	decoded := c.decodeBatch(ctx, batch)
	failed = decoded.failed

	if len(decoded.messages) == 0 {
		return failed
	}

	if acks, err = c.callBatch(decoded.ctx, decoded.models, decoded.attributes); err != nil {
		c.handleError(ctx, err, "batch processing failed")
	}

	if len(acks) != len(decoded.messages) {
		c.handleError(ctx, fmt.Errorf("%d acknowledgements for %d models", len(acks), len(decoded.messages)), "batch result length mismatch")
	}

	for i, message := range decoded.messages {
		if i < len(acks) && acks[i] {
			continue
		}

		if !c.retryBatchMessage(ctx, message) {
			failed++
		}
	}

	return failed
}

type decodedBatch struct {
	ctx        context.Context
	messages   []batchMessage
	models     []any
	attributes []map[string]string
	failed     int
}

// decodeBatch prepares callback models and attributes while preserving originals for retry.
// It skips configured ignorable errors and routes other model or decoding failures to retry.
func (c *BatchConsumer) decodeBatch(ctx context.Context, batch []batchMessage) decodedBatch {
	var err error
	var model any
	var decodeCtx context.Context
	var attr map[string]string

	decoded := decodedBatch{
		ctx:        ctx,
		models:     make([]any, 0, len(batch)),
		attributes: make([]map[string]string, 0, len(batch)),
		messages:   make([]batchMessage, 0, len(batch)),
	}

	for _, message := range batch {
		if model, err = c.getBatchModel(message.message.Attributes); err != nil {
			c.metricWriter.Write(ctx, metric.Data{&metric.Datum{MetricName: metricNameConsumerUnknownModelError, Dimensions: map[string]string{"Consumer": c.name}, Value: 1}})

			var ignorable IgnorableGetModelError
			if errors.As(err, &ignorable) && ignorable.IsIgnorableWithSettings(c.settings.IgnoreOnGetModelError) {
				continue
			}
		} else if model == nil {
			err = fmt.Errorf("nil model for message attributes %v", message.message.Attributes)
		}

		if err != nil {
			c.handleError(ctx, err, "batch GetModel failed")

			if !c.retryBatchMessage(ctx, message) {
				decoded.failed++
			}

			continue
		}

		decodeMsg := *message.message
		decodeMsg.Attributes = maps.Clone(decodeMsg.Attributes)

		if decodeCtx, attr, err = c.encoder.Decode(ctx, &decodeMsg, model); err != nil {
			c.handleError(decodeCtx, err, "batch decoding failed")

			if !c.retryBatchMessage(ctx, message) {
				decoded.failed++
			}

			continue
		}

		decoded.models = append(decoded.models, model)
		decoded.attributes = append(decoded.attributes, attr)
		decoded.messages = append(decoded.messages, message)

		if len(decoded.messages) == 1 {
			decoded.ctx = decodeCtx
		}
	}

	return decoded
}

// getBatchModel obtains a model using copied attributes and converts callback panics to errors.
func (c *BatchConsumer) getBatchModel(attributes map[string]string) (model any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			model = nil
			err = fmt.Errorf("batch GetModel panicked: %v", recovered)
		}
	}()

	return c.batchCallback.GetModel(maps.Clone(attributes))
}

// callBatch invokes the callback unless canceled and converts callback panics to errors.
func (c *BatchConsumer) callBatch(ctx context.Context, models []any, attributes []map[string]string) (acks []bool, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			acks = nil
			err = fmt.Errorf("batch callback panicked: %v", recovered)
		}
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return c.batchCallback.Consume(ctx, models, attributes)
}

// retryBatchMessage persists a failed primary record or leaves retry-input redelivery to its transport.
// It returns true only when the primary record was handed to the retry handler.
func (c *BatchConsumer) retryBatchMessage(ctx context.Context, message batchMessage) bool {
	if message.result != nil {
		// Returning false to the retry input preserves its native redelivery/DLQ.
		return false
	}

	if !c.settings.Retry.Enabled {
		c.handleError(ctx, fmt.Errorf("attributes %v", message.message.Attributes), "unsuccessful acknowledged batch message requires replay")

		return false
	}

	retryMessage, _ := c.buildRetryMessage(message.message)
	c.writeMetricRetryCount(ctx, metricNameConsumerRetryPutCount)

	ctx, cancel := exec.WithDelayedCancelContext(ctx, c.settings.Retry.GraceTime)
	defer cancel()

	if err := c.retryHandler.Put(ctx, retryMessage); err != nil {
		c.handleError(ctx, err, "can not put batch message into retry handler; replay required")

		return false
	}

	return true
}

// runCallback runs collection alongside optional callback background work and coordinates stopping inputs.
func (c *BatchConsumer) runCallback(ctx context.Context) error {
	cfn := coffin.New()
	runCtx := cfn.Context(ctx)

	cfn.GoWithContextf(runCtx, c.runBatches, "panic during batch collection")

	if callback, ok := c.batchCallback.(RunnableCallback); ok {
		cfn.GoWithContextf(runCtx, callback.Run, "panic during batch callback Run")
	}

	cfn.Go(func() error {
		<-runCtx.Done()
		c.stopIncomingData(context.WithoutCancel(ctx))

		return nil
	})

	return cfn.Wait()
}
