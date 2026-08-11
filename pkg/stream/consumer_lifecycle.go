package stream

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNameConsumerDuration          = "Duration"
	metricNameConsumerError             = "Error"
	metricNameConsumerProcessedCount    = "ProcessedCount"
	metricNameConsumerRetryGetCount     = "RetryGetCount"
	metricNameConsumerRetryPutCount     = "RetryPutCount"
	metricNameConsumerUnknownModelError = "UnknownModelError"
	metadataKeyConsumers                = "stream.consumers"
)

type ConsumerMetadata struct {
	Name         string `json:"name"`
	RetryEnabled bool   `json:"retry_enabled"`
	RetryType    string `json:"retry_type"`
}

type InitializeableCallback interface {
	Init(ctx context.Context) error
}

//go:generate go run github.com/vektra/mockery/v2 --name RunnableCallback
type RunnableCallback interface {
	Run(ctx context.Context) error
}

//go:generate go run github.com/vektra/mockery/v2 --name SchemaSettingsAwareCallback
type SchemaSettingsAwareCallback interface {
	GetSchemaSettings() (*SchemaSettings, error)
}

// IgnorableGetModelError is an interface that can be implemented by errors returned from GetModel
// to indicate whether the error is ignorable (i.e., the message should be acknowledged without processing).
type IgnorableGetModelError interface {
	error
	// IsIgnorableWithSettings returns true if this error should result in the message being ignored
	// based on the given settings.
	IsIgnorableWithSettings(settings IgnoreOnGetModelErrorSettings) bool
}

func (c *Consumer) run(kernelCtx context.Context) error {
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
		cfn.GoWithContextf(dyingCtx, c.trackInputRun(c.input, stopCtx, true), "panic during run of the consumer input")
		cfn.GoWithContextf(dyingCtx, c.trackInputRun(c.retryInput, stopCtx, false), "panic during run of the retry handler")

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

func (c *Consumer) trackInputRun(input Input, stopCtx context.Context, stopOnReturn bool) func(ctx context.Context) error {
	return func(dyingCtx context.Context) error {
		defer c.wg.Done()

		// The consumer owns the processing deadline for every input: it is the only layer which sees the callback and
		// thus the only one which can bound it uniformly. Inputs which hand records to a callback of their own must
		// therefore keep them alive until this context is done instead of running a second timer over the same record.
		dyingCtx = exec.WithDrainContext(dyingCtx, c.drainCtx)

		err := input.Run(dyingCtx, func(ctx context.Context, msg *Message) (ack bool) {
			return c.processData(ctx, msg)
		})

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

func (c *Consumer) logConsumeCounter(ctx context.Context) error {
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

func (c *Consumer) logProcessedMessages(ctx context.Context, lastLog *time.Time) {
	processed := atomic.SwapInt32(&c.processed, 0)
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

func (c *Consumer) initConsumerCallback(ctx context.Context) error {
	if initializeable, ok := c.callback.(InitializeableCallback); ok {
		return initializeable.Init(ctx)
	}

	return nil
}

func (c *Consumer) runConsumerCallback(ctx context.Context) error {
	defer c.logger.Debug(ctx, "runConsumerCallback is ending")

	if runnable, ok := c.callback.(RunnableCallback); ok {
		return runnable.Run(ctx)
	}

	return nil
}

// this one acts as a fallback which should stop all still running routines
func (c *Consumer) stopConsuming(cfn coffin.Coffin) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		defer c.logger.Debug(ctx, "stopConsuming is ending")

		c.wg.Wait()
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

func (c *Consumer) stopIncomingData(ctx context.Context) {
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

func (c *Consumer) recover(ctx context.Context, msg *Message) {
	var err error

	if err = coffin.ResolveRecovery(recover()); err == nil {
		return
	}

	c.handleError(ctx, err, "a panic occurred during the consume operation")

	if msg == nil || c.hasNativeRetry() {
		return
	}

	c.retry(ctx, msg)
}

func (c *Consumer) retry(ctx context.Context, msg *Message) {
	if !c.settings.Retry.Enabled {
		return
	}

	retryMsg, retryId := c.buildRetryMessage(msg)

	ctx = log.AppendGlobalContextFields(ctx, log.Fields{
		"retry_id": retryId,
	})

	c.logger.Warn(ctx, "putting message with id %s into retry", retryId)
	c.writeMetricRetryCount(ctx, metricNameConsumerRetryPutCount)

	ctx, stop := exec.WithDelayedCancelContext(ctx, c.settings.Retry.GraceTime)
	defer stop()

	if err := c.retryHandler.Put(ctx, retryMsg); err != nil {
		c.handleError(ctx, err, "can not put the message into the retry handler")
	}
}

func (c *Consumer) hasNativeRetry() bool {
	_, ok := c.input.(RetryingInput)

	return ok
}

func (c *Consumer) buildRetryMessage(msg *Message) (retryMsg *Message, retryId string) {
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

func (c *Consumer) handleError(ctx context.Context, err error, msg string) {
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

func (c *Consumer) isHealthy() bool {
	return c.input.IsHealthy() && c.retryInput.IsHealthy()
}

func (c *Consumer) writeMetricDurationAndProcessedCount(ctx context.Context, duration time.Duration, processedCount int) {
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

func (c *Consumer) writeMetricRetryCount(ctx context.Context, metricName string) {
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

func getConsumerDefaultMetrics(name string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerProcessedCount,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryPutCount,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryGetCount,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerUnknownModelError,
			Dimensions: map[string]string{
				"Consumer": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
