package exec

import (
	"context"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/cenkalti/backoff"
	"time"
)

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

type BackoffExecutor struct {
	logger   mon.Logger
	res      *ExecutableResource
	checks   []ErrorChecker
	settings *BackoffSettings
}

func NewBackoffExecutor(logger mon.Logger, res *ExecutableResource, settings *BackoffSettings, checks ...ErrorChecker) *BackoffExecutor {
	return &BackoffExecutor{
		logger:   logger,
		res:      res,
		checks:   checks,
		settings: settings,
	}
}

func (e *BackoffExecutor) Execute(ctx context.Context, f Executable) (interface{}, error) {
	logger := e.logger.WithContext(ctx)

	delayedCtx := WithDelayedCancelContext(ctx, e.settings.CancelDelay)
	defer delayedCtx.Stop()

	var res interface{}
	var err error
	var errType ErrorType

	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = e.settings.InitialInterval
	backoffConfig.RandomizationFactor = e.settings.RandomizationFactor
	backoffConfig.Multiplier = e.settings.Multiplier
	backoffConfig.MaxInterval = e.settings.MaxInterval
	backoffConfig.MaxElapsedTime = e.settings.MaxElapsedTime

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
		}).Warnf("retrying resource %s %s after error: %s", e.res.Type, e.res.Name, err.Error())

		retries++
		timespan += duration
	}

	_ = backoff.RetryNotify(func() error {
		res, err = f(delayedCtx)

		if err == nil {
			return nil
		}

		for _, check := range e.checks {
			errType = check(res, err)

			switch errType {
			case ErrorOk:
				return nil
			case ErrorRetryable:
				return err
			case ErrorPermanent:
				return backoff.Permanent(err)
			}
		}

		return backoff.Permanent(err)
	}, backoffCtx, notify)

	if err != nil && errType != ErrorOk {
		logger.Warnf("error on requesting resource %s %s: %s", e.res.Type, e.res.Name, err.Error())

		return res, err
	}

	if retries > 0 {
		logger.Infof("sent request to resource %s %s successful after %d retries in %s", e.res.Type, e.res.Name, retries, timespan)
	}

	return res, err
}
