package cloud

import (
	"context"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/cenkalti/backoff"
	"time"
)

const (
	metricNameErrorCount = "AwsRequestErrorCount"
	metricNameRetryCount = "AwsRequestRetryCount"
)

type delayedCancelContext struct {
	context.Context
	done chan struct{}
	stop chan struct{}
}

func (c *delayedCancelContext) Done() <-chan struct{} {
	return c.done
}

func (c *delayedCancelContext) Stop() {
	close(c.stop)
}

func WithDelayedCancelContext(parentCtx context.Context, delay time.Duration) *delayedCancelContext {
	done := make(chan struct{})
	stop := make(chan struct{})

	go func() {
		select {
		case <-stop:
		case <-parentCtx.Done():
			time.Sleep(delay)
			close(done)
		}
	}()

	return &delayedCancelContext{
		Context: parentCtx,
		done:    done,
		stop:    stop,
	}
}

type BackoffResource struct {
	Type string
	Name string
}

type BackoffSettings struct {
	Enabled             bool          `cfg:"enabled" default:"false"`
	Blocking            bool          `cfg:"blocking" default:"false"`
	CancelDelay         time.Duration `cfg:"cancel_delay" default:"1s"`
	InitialInterval     time.Duration `cfg:"initial_interval" default:"50ms"`
	RandomizationFactor float64       `cfg:"randomization_factor" default:"0.5"`
	Multiplier          float64       `cfg:"multiplier" default:"1.5"`
	MaxInterval         time.Duration `cfg:"max_interval" default:"10s"`
	MaxElapsedTime      time.Duration `cfg:"max_elapsed_time" default:"15m"`
}

type AwsRequestFunction func(ctx context.Context) (interface{}, error)

type RequestExecutor interface {
	Execute(ctx context.Context, f AwsRequestFunction) (interface{}, error)
}

type DefaultExecutor struct {
}

func (e DefaultExecutor) Execute(ctx context.Context, f AwsRequestFunction) (interface{}, error) {
	return f(ctx)
}

type BackoffExecutor struct {
	logger   mon.Logger
	metric   mon.MetricWriter
	res      *BackoffResource
	settings *BackoffSettings
}

func NewBackoffExecutor(logger mon.Logger, res *BackoffResource, settings *BackoffSettings) *BackoffExecutor {
	defaults := getBackoffExecutorDefaultQueueMetrics(res)
	metric := mon.NewMetricDaemonWriter(defaults...)

	return &BackoffExecutor{
		logger:   logger,
		metric:   metric,
		res:      res,
		settings: settings,
	}
}

func (e *BackoffExecutor) Execute(ctx context.Context, f AwsRequestFunction) (interface{}, error) {
	if !e.settings.Enabled {
		return f(ctx)
	}

	logger := e.logger.WithContext(ctx)

	delayedCtx := WithDelayedCancelContext(ctx, e.settings.CancelDelay)
	defer delayedCtx.Stop()

	var out interface{}
	var err error

	backoffConfig := backoff.NewExponentialBackOff()

	if e.settings.InitialInterval > 0 {
		backoffConfig.InitialInterval = e.settings.InitialInterval
	}

	if e.settings.RandomizationFactor > 0 {
		backoffConfig.RandomizationFactor = e.settings.RandomizationFactor
	}

	if e.settings.Multiplier > 0 {
		backoffConfig.Multiplier = e.settings.Multiplier
	}

	if e.settings.MaxInterval > 0 {
		backoffConfig.MaxInterval = e.settings.MaxInterval
	}

	if e.settings.MaxElapsedTime > 0 {
		backoffConfig.MaxElapsedTime = e.settings.MaxElapsedTime
	}

	if e.settings.Blocking {
		backoffConfig.MaxElapsedTime = 0
	}

	cancelCtx, _ := context.WithCancel(ctx)
	backoffCtx := backoff.WithContext(backoffConfig, cancelCtx)

	notify := func(err error, duration time.Duration) {
		logger.Warnf("retrying aws service %s %s after error: %s", e.res.Type, e.res.Name, err.Error())
		e.writeMetric(metricNameRetryCount)
	}

	_ = backoff.RetryNotify(func() error {
		out, err = f(delayedCtx)

		if !IsConnectionError(err) {
			return nil
		}

		return err
	}, backoffCtx, notify)

	if err != nil {
		logger.Warnf("error on requesting aws service %s %s: %s", e.res.Type, e.res.Name, err.Error())
		e.writeMetric(metricNameErrorCount)
	}

	return out, err
}

func (e *BackoffExecutor) writeMetric(metric string) {
	e.metric.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		MetricName: metric,
		Dimensions: map[string]string{
			"Type": e.res.Type,
			"Name": e.res.Name,
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})
}

func getBackoffExecutorDefaultQueueMetrics(res *BackoffResource) mon.MetricData {
	return mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameErrorCount,
			Dimensions: map[string]string{
				"Type": res.Type,
				"Name": res.Name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameRetryCount,
			Dimensions: map[string]string{
				"Type": res.Type,
				"Name": res.Name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
