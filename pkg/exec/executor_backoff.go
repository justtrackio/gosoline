package exec

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type backoffExecuteAttempts string

const BackoffExecuteAttempt = backoffExecuteAttempts("backoff.execute.attempt")

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

	attempts := 1
	attemptsContext := context.WithValue(ctx, BackoffExecuteAttempt, &attempts)
	delayedCtx := WithDelayedCancelContext(attemptsContext, e.settings.CancelDelay)
	defer delayedCtx.Stop()

	var res interface{}
	var err error
	var errType ErrorType

	backoffConfig := NewExponentialBackOff(e.settings)
	backoffCtx := backoff.WithContext(backoffConfig, ctx)

	start := time.Now()

	notify := func(err error, _ time.Duration) {
		logger.Warn("retrying resource %s after error: %s", e.resource, err.Error())
		attempts++
	}

	_ = backoff.RetryNotify(func() error {
		res, err = f(delayedCtx)

		if err == nil {
			return nil
		}

		if e.settings.MaxAttempts > 0 && attempts >= e.settings.MaxAttempts {
			return backoff.Permanent(err)
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

	// we're having an error after reaching the MaxAttempts and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk && e.settings.MaxAttempts > 0 && attempts > e.settings.MaxAttempts {
		logger.Warn("crossed max attempts with an error on requesting resource %s after %d attempts in %s: %s", e.resource, attempts, duration, err.Error())

		return res, NewErrAttemptsExceeded(e.resource, attempts, duration, err)
	}

	// we're having an error after reaching the MaxElapsedTime and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk && e.settings.MaxElapsedTime > 0 && duration > e.settings.MaxElapsedTime {
		logger.Warn("crossed max elapsed time with an error on requesting resource %s after %d attempts in %s: %s", e.resource, attempts, duration, err.Error())

		return res, NewErrMaxElapsedTimeExceeded(e.resource, attempts, duration, e.settings.MaxElapsedTime, err)
	}

	// we're still having an error and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk {
		logger.Warn("error on requesting resource %s after %d attempts in %s: %s", e.resource, attempts, duration, err.Error())

		return res, err
	}

	if attempts > 1 {
		logger.Info("sent request to resource %s successful after %d attempts in %s", e.resource, attempts, duration)
	}

	return res, err
}
