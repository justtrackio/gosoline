package exec

import (
	"context"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/uuid"
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
	logger   log.Logger
	uuidGen  uuid.Uuid
	resource *ExecutableResource
	checks   []ErrorChecker
	settings *BackoffSettings
}

func NewBackoffExecutor(logger log.Logger, res *ExecutableResource, settings *BackoffSettings, checks ...ErrorChecker) *BackoffExecutor {
	return &BackoffExecutor{
		logger:   logger,
		uuidGen:  uuid.New(),
		resource: res,
		checks:   checks,
		settings: settings,
	}
}

func (e *BackoffExecutor) Execute(ctx context.Context, f Executable) (interface{}, error) {
	logger := e.logger.WithContext(ctx).WithFields(log.Fields{
		"exec_id":            e.uuidGen.NewV4(),
		"exec_resource_type": e.resource.Type,
		"exec_resource_name": e.resource.Name,
	})

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
	start := time.Now()

	notify := func(err error, _ time.Duration) {
		logger.Warn("retrying resource %s %s after error: %s", e.resource.Type, e.resource.Name, err.Error())
		retries++
	}

	_ = backoff.RetryNotify(func() error {
		res, err = f(delayedCtx)

		if err == nil {
			return nil
		}

		for _, check := range e.checks {
			errType = check(res, err)

			switch errType {
			case ErrorTypeOk:
				return nil
			case ErrorTypeRetryable:
				return err
			case ErrorTypePermanent:
				return backoff.Permanent(err)
			}
		}

		return backoff.Permanent(err)
	}, backoffCtx, notify)

	duration := time.Since(start)

	// we're having an error after reaching the MaxElapsedTime and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk && e.settings.MaxElapsedTime > 0 && duration > e.settings.MaxElapsedTime {
		logger.Warn("crossed max elapsed time with an error on requesting resource %s %s after %d retries in %s: %s", e.resource.Type, e.resource.Name, retries, duration, err.Error())

		return res, NewMaxElapsedTimeError(e.settings.MaxElapsedTime, duration, err)
	}

	// we're still having an error and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk {
		logger.Warn("error on requesting resource %s %s after %d retries in %s: %s", e.resource.Type, e.resource.Name, retries, duration, err.Error())

		return res, err
	}

	if retries > 0 {
		logger.Info("sent request to resource %s %s successful after %d retries in %s", e.resource.Type, e.resource.Name, retries, duration)
	}

	return res, err
}
