package cloud

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/cenkalti/backoff"
	"time"
)

const (
	metricNameErrorCount = "AwsRequestErrorCount"
	metricNameRetryCount = "AwsRequestRetryCount"
)

type CustomExecResultHandler func(err error) (error, bool)

type BackoffResource struct {
	Type    string
	Name    string
	Handler []CustomExecResultHandler
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
	MetricEnabled       bool          `cfg:"metric_enabled" default:"true"`
}

type BackoffExecutor struct {
	logger   mon.Logger
	metric   mon.MetricWriter
	res      *BackoffResource
	settings *BackoffSettings
}

func NewBackoffExecutor(logger mon.Logger, res *BackoffResource, settings *BackoffSettings) *BackoffExecutor {
	defaults := getBackoffExecutorDefaultQueueMetrics(res)
	if !settings.MetricEnabled {
		defaults = make(mon.MetricData, 0)
	}
	metric := mon.NewMetricDaemonWriter(defaults...)

	return &BackoffExecutor{
		logger:   logger,
		metric:   metric,
		res:      res,
		settings: settings,
	}
}

func (e *BackoffExecutor) Execute(ctx context.Context, f RequestFunction) (interface{}, error) {
	logger := e.logger.WithContext(ctx)

	delayedCtx := WithDelayedCancelContext(ctx, e.settings.CancelDelay)
	defer delayedCtx.Stop()

	var req *request.Request
	var out interface{}
	var err error
	var errOk = false

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

	backoffCtx := backoff.WithContext(backoffConfig, ctx)

	retries := 0
	timespan := time.Duration(0)

	notify := func(err error, duration time.Duration) {
		logger.WithFields(mon.Fields{
			"resource_type": e.res.Type,
			"resource_name": e.res.Name,
		}).Warnf("retrying aws service %s %s after error: %s", e.res.Type, e.res.Name, err.Error())

		e.writeMetric(metricNameRetryCount)

		retries++
		timespan += duration
	}

	_ = backoff.RetryNotify(func() error {
		req, out = f()

		req.SetContext(delayedCtx)
		err = req.Send()

		if req.HTTPResponse.StatusCode >= 500 && req.HTTPResponse.StatusCode != 501 {
			return fmt.Errorf("http status code: %d", req.HTTPResponse.StatusCode)
		}

		if err == nil {
			return nil
		}

		if IsRequestCanceled(err) {
			return backoff.Permanent(err)
		}

		if IsUsedClosedConnectionError(err) {
			return err
		}

		if request.IsErrorRetryable(err) {
			return err
		}

		if request.IsErrorThrottle(err) {
			return err
		}

		return nil
	}, backoffCtx, notify)

	for _, h := range e.res.Handler {
		if err, errOk = h(err); errOk {
			break
		}
	}

	if err != nil && !errOk {
		logger.Warnf("error on requesting aws service %s %s: %s", e.res.Type, e.res.Name, err.Error())
		e.writeMetric(metricNameErrorCount)

		return out, err
	}

	if retries > 0 {
		logger.Infof("sent request to aws service %s %s successful after %d retries in %s", e.res.Type, e.res.Name, retries, timespan)
	}

	return out, err
}

func (e *BackoffExecutor) writeMetric(metric string) {
	if !e.settings.MetricEnabled {
		return
	}

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
