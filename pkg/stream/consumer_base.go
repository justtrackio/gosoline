package stream

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/reqctx"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type ConsumerMetadata struct {
	Name         string `json:"name"`
	RetryEnabled bool   `json:"retry_enabled"`
	RetryType    string `json:"retry_type"`
}

// consumerBase owns shared dependencies and lifecycle, without interpreting
// messages or deciding when their processing is successful.
type consumerBase struct {
	kernel.EssentialModule
	kernel.ApplicationStage

	clock        clock.Clock
	uuidGen      uuid.Uuid
	logger       log.Logger
	metricWriter metric.Writer
	tracer       tracing.Tracer
	encoder      MessageEncoder
	input        Input
	retryInput   Input
	retryHandler RetryHandler

	wg                  sync.WaitGroup
	stopped             sync.Once
	cancel              context.CancelFunc
	processingStartedAt funk.Maper[uint64, time.Time]
	processingSequence  atomic.Uint64
	drainCtx            context.Context
	drainCancel         context.CancelFunc

	id              string
	name            string
	settings        ConsumerSettings
	processed       atomic.Int32
	samplingDecider smpl.Decider
	inputProcess    InputProcess
	retryProcess    InputProcess

	init           func(context.Context) error
	run            func(context.Context) error
	inputsFinished func()
}

func newConsumerBase(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	name string,
	schemaCallback SchemaSettingsAwareCallback,
	settings ConsumerSettings,
	retryFactory func(context.Context, cfg.Config, log.Logger, Input, *ConsumerRetrySettings, string) (Input, RetryHandler, error),
) (*consumerBase, error) {
	var err error
	var tracer tracing.Tracer
	var input Input
	var encoder MessageEncoder
	var retryInput Input
	var retryHandler RetryHandler
	var samplingDecider smpl.Decider

	consumerLogger := logger.WithChannel(fmt.Sprintf("consumer-%s", name))
	metricWriter := metric.NewWriter(getConsumerDefaultMetrics(name)...)

	if _, err := cfg.GetAppIdentity(config); err != nil {
		return nil, fmt.Errorf("can not get app identity from config: %w", err)
	}

	if tracer, err = tracing.ProvideTracer(ctx, config, consumerLogger); err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	if input, err = NewConfigurableInput(ctx, config, consumerLogger, settings.Input); err != nil {
		return nil, err
	}

	if encoder, err = newConsumerEncoder(ctx, input, schemaCallback, settings.Encoding); err != nil {
		return nil, err
	}

	if retryInput, retryHandler, err = retryFactory(ctx, config, consumerLogger, input, &settings.Retry, name); err != nil {
		return nil, err
	}

	if samplingDecider, err = smpl.ProvideDecider(ctx, config); err != nil {
		return nil, fmt.Errorf("could not initialize sampling decider: %w", err)
	}

	consumerMetadata := ConsumerMetadata{Name: name, RetryEnabled: settings.Retry.Enabled, RetryType: settings.Retry.Type}
	if err := appctx.MetadataAppend(ctx, metadataKeyConsumers, consumerMetadata); err != nil {
		return nil, fmt.Errorf("can not access the appctx metadata: %w", err)
	}

	return NewConsumerBaseWithInterfaces(
		uuid.New(),
		consumerLogger,
		metricWriter,
		tracer,
		input,
		encoder,
		retryInput,
		retryHandler,
		settings,
		name,
		samplingDecider,
		clock.Provider,
	), nil
}

func NewConsumerBaseWithInterfaces(
	uuidGen uuid.Uuid,
	logger log.Logger,
	metricWriter metric.Writer,
	tracer tracing.Tracer,
	input Input,
	encoder MessageEncoder,
	retryInput Input,
	retryHandler RetryHandler,
	settings ConsumerSettings,
	name string,
	samplingDecider smpl.Decider,
	clock clock.Clock,
) *consumerBase {
	drainCtx, drainCancel := context.WithCancel(context.Background())

	return &consumerBase{
		id:                  fmt.Sprintf("consumer-%s", name),
		name:                name,
		clock:               clock,
		uuidGen:             uuidGen,
		logger:              logger,
		metricWriter:        metricWriter,
		tracer:              tracer,
		encoder:             encoder,
		input:               input,
		retryInput:          retryInput,
		retryHandler:        retryHandler,
		settings:            settings,
		samplingDecider:     samplingDecider,
		drainCtx:            drainCtx,
		drainCancel:         drainCancel,
		processingStartedAt: funk.NewMapSynced[uint64, time.Time](),
	}
}

// setCallbackHooks wires optional callback hooks before the consumer runs.
func (c *consumerBase) setCallbackHooks(callback any) {
	c.init, c.run, c.inputsFinished = nil, nil, nil

	if initializeable, ok := callback.(InitializeableCallback); ok {
		c.init = initializeable.Init
	}

	if runnable, ok := callback.(RunnableCallback); ok {
		c.run = runnable.Run
	}
}

func (c *consumerBase) Run(ctx context.Context) error {
	return c.runConsumer(ctx)
}

// IsHealthy checks input health and the processing deadline of each active callback.
func (c *consumerBase) IsHealthy(_ context.Context) (bool, error) {
	timedOut := c.processingStartedAt.Any(func(_ uint64, startedAt time.Time) bool {
		return c.clock.Since(startedAt) > c.settings.Healthcheck.Timeout
	})

	return c.isHealthy() && !timedOut, nil
}

func (c *consumerBase) startTracingContext(ctx context.Context) (context.Context, tracing.Span) {
	ctx, span := c.tracer.StartSpanFromContext(ctx, c.id)
	ctx = log.InitContext(ctx)
	ctx = log.WithFingersCrossedScope(ctx)
	ctx = reqctx.New(ctx)

	return ctx, span
}

func (c *consumerBase) runConsumer(kernelCtx context.Context) error {
	defer c.logger.Info(kernelCtx, "leaving consumer %s", c.name)

	if err := c.initConsumerCallback(kernelCtx); err != nil {
		return fmt.Errorf("can not init consumer callback: %w", err)
	}

	c.logger.Info(kernelCtx, "running consumer %s with input %s", c.name, c.settings.Input)
	c.wg.Add(2)

	// create ctx whose done channel is closed on dying coffin
	cfn, dyingCtx := coffin.WithContext(context.Background())

	// Stop may need to perform I/O while shutting down, so it must not inherit kernel cancellation.
	stopCtx := context.WithoutCancel(kernelCtx)

	// Keep callback contexts alive while inputs finish processing and acknowledging in-flight messages.
	manualCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	defer c.drainCancel()

	cfn.Go(func() error {
		cfn.GoWithContextf(manualCtx, c.logConsumeCounter, "panic during counter log")
		cfn.GoWithContextf(manualCtx, c.runConsumerCallback, "panic during run of the consumerCallback")
		cfn.GoWithContextf(dyingCtx, c.trackInputRun(c.input, c.inputProcess, stopCtx, true), "panic during run of the consumer input")
		cfn.GoWithContextf(dyingCtx, c.trackInputRun(c.retryInput, c.retryProcess, stopCtx, false), "panic during run of the retry handler")

		cfn.GoWithContextf(manualCtx, c.stopConsuming(cfn), "panic during stopping the consuming")

		cfn.Go(func() error {
			// wait for kernel or coffin cancel...
			select {
			case <-dyingCtx.Done():
			case <-kernelCtx.Done():
			}

			// and stop the input
			c.stopIncomingData(stopCtx)

			return nil
		})

		return nil
	})

	if err := cfn.Wait(); err != nil {
		return fmt.Errorf("error while waiting for all routines to stop: %w", err)
	}

	return nil
}

func (c *consumerBase) trackInputRun(input Input, process InputProcess, stopCtx context.Context, stopOnReturn bool) func(ctx context.Context) error {
	return func(dyingCtx context.Context) error {
		defer c.wg.Done()

		// The consumer owns the processing deadline for every input: it is the only layer which sees the callback and
		// thus the only one which can bound it uniformly. Inputs which hand records to a callback of their own must
		// therefore keep them alive until this context is done instead of running a second timer over the same record.
		dyingCtx = exec.WithDrainContext(dyingCtx, c.drainCtx)

		err := input.Run(dyingCtx, process)

		if stopOnReturn {
			// The consumer input returned on its own, so no further messages will arrive. Stop the remaining inputs
			// gracefully rather than killing the coffin: killing it would cancel the context the retry input runs
			// with and make it abandon messages which are still queued. Stopping instead lets the retry input drain
			// its backlog until the shared grace deadline expires, after which stopConsuming tears the rest down.
			c.stopIncomingData(stopCtx)
		}

		return err
	}
}

func (c *consumerBase) logConsumeCounter(ctx context.Context) error {
	defer c.logger.Debug(ctx, "logConsumeCounter is ending")

	lastLog := c.clock.Now()
	ticker := c.clock.NewTicker(c.settings.IdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logProcessedMessages(ctx, &lastLog)

			return nil
		case <-ticker.Chan():
			c.logProcessedMessages(ctx, &lastLog)
		}
	}
}

func (c *consumerBase) logProcessedMessages(ctx context.Context, lastLog *time.Time) {
	processed := c.processed.Swap(0)

	now := c.clock.Now()
	took := now.Sub(*lastLog)
	*lastLog = now

	c.logger.WithFields(log.Fields{
		"count": processed,
		"took":  took,
		"name":  c.name,
	}).Info(
		ctx,
		"consumer %s processed %d messages in %vs (%.1f messages/s)",
		c.name,
		processed,
		took.Seconds(),
		float64(processed)/took.Seconds(),
	)
}

func (c *consumerBase) initConsumerCallback(ctx context.Context) error {
	if c.init != nil {
		return c.init(ctx)
	}

	return nil
}

func (c *consumerBase) runConsumerCallback(ctx context.Context) error {
	defer c.logger.Debug(ctx, "runConsumerCallback is ending")

	if c.run != nil {
		return c.run(ctx)
	}

	return nil
}

// this one acts as a fallback which should stop all still running routines
func (c *consumerBase) stopConsuming(cfn coffin.Coffin) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		defer c.logger.Debug(ctx, "stopConsuming is ending")

		c.wg.Wait()
		if c.inputsFinished != nil {
			c.inputsFinished()
		}

		c.stopIncomingData(ctx)
		c.cancel()

		// Both inputs returned, so there is nothing left to consume. Kill the coffin to release the routine waiting
		// for the kernel or the coffin to shut us down: when the input finished on its own instead of the kernel
		// cancelling us, neither of those ever triggers and the consumer would stay alive forever.
		if cfn.Alive() {
			cfn.Kill(nil)
		}

		return nil
	}
}

func (c *consumerBase) stopIncomingData(ctx context.Context) {
	c.stopped.Do(func() {
		defer c.logger.Debug(ctx, "stopIncomingData is ending")

		c.retryInput.Stop(ctx)
		c.input.Stop(ctx)

		go func() {
			timer := c.clock.NewTimer(c.settings.GraceTime)
			defer timer.Stop()

			select {
			case <-timer.Chan():
				c.logger.Warn(ctx, "drain grace time of %v expired, cancelling in-flight messages", c.settings.GraceTime)
				c.drainCancel()
			case <-c.drainCtx.Done():
			}
		}()
	})
}

func (c *consumerBase) buildRetryMessage(msg *Message) (retryMsg *Message, retryId string) {
	if retryId, ok := msg.Attributes[AttributeRetryId]; ok {
		return msg, retryId
	}

	retryId = c.uuidGen.NewV4()
	retryMsg = &Message{
		Attributes: funk.MergeMaps(msg.Attributes, map[string]string{
			AttributeRetry:   strconv.FormatBool(true),
			AttributeRetryId: retryId,
		}),
		Body: msg.Body,
	}

	return retryMsg, retryId
}

func (c *consumerBase) handleError(ctx context.Context, err error, msg string) {
	if exec.IsRequestCanceled(err) || ctx.Err() != nil {
		c.logger.Warn(ctx, "%s during shutdown: %s", msg, err)

		return
	}

	c.logger.Error(ctx, "%s: %w", msg, err)

	c.metricWriter.Write(ctx, metric.Data{
		&metric.Datum{
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: 1.0,
		},
	})
}

func (c *consumerBase) isHealthy() bool {
	return c.input.IsHealthy() && c.retryInput.IsHealthy()
}

func (c *consumerBase) writeMetricDurationAndProcessedCount(ctx context.Context, duration time.Duration, processedCount int) {
	c.metricWriter.Write(ctx, metric.Data{
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerDuration,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: float64(duration.Milliseconds()),
		},
		&metric.Datum{
			MetricName: metricNameConsumerProcessedCount,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: float64(processedCount),
		},
	})
}

func (c *consumerBase) writeMetricRetryCount(ctx context.Context, metricName string) {
	c.metricWriter.Write(ctx, metric.Data{
		&metric.Datum{
			MetricName: metricName,
			Dimensions: map[string]string{
				"Consumer": c.name,
			},
			Value: float64(1),
		},
	})
}
