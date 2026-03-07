package exec

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type BackoffExecutor struct {
	logger             log.Logger
	uuidGen            uuid.Uuid
	resource           *ExecutableResource
	settings           *BackoffSettings
	checks             []ErrorChecker
	notifier           []Notify
	timeTrackerFactory func() ElapsedTimeTracker
}

// BackoffExecutorOption is a functional option for configuring a BackoffExecutor.
type BackoffExecutorOption func(*BackoffExecutor)

// WithElapsedTimeTrackerFactory sets a factory function for creating elapsed time trackers.
// Each Execute call will use a new tracker instance, making it safe for concurrent use.
// Use this to change how the MaxElapsedTime budget is measured.
func WithElapsedTimeTrackerFactory(factory func() ElapsedTimeTracker) BackoffExecutorOption {
	return func(e *BackoffExecutor) {
		e.timeTrackerFactory = factory
	}
}

// WithNotifiers adds notifiers that will be called on each retry attempt.
func WithNotifiers(notifiers ...Notify) BackoffExecutorOption {
	return func(e *BackoffExecutor) {
		e.notifier = append(e.notifier, notifiers...)
	}
}

// NewBackoffExecutor creates a BackoffExecutor with functional options.
func NewBackoffExecutor(
	logger log.Logger,
	res *ExecutableResource,
	settings *BackoffSettings,
	checks []ErrorChecker,
	opts ...BackoffExecutorOption,
) *BackoffExecutor {
	e := &BackoffExecutor{
		logger:             logger,
		uuidGen:            uuid.New(),
		resource:           res,
		checks:             checks,
		settings:           settings,
		timeTrackerFactory: func() ElapsedTimeTracker { return NewDefaultElapsedTimeTracker() },
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *BackoffExecutor) Execute(ctx context.Context, f Executable, notifier ...Notify) (any, error) {
	logger := e.logger.WithFields(log.Fields{
		"exec_id":            e.uuidGen.NewV4(),
		"exec_resource_type": e.resource.Type,
		"exec_resource_name": e.resource.Name,
	})

	delayedCtx, stop := WithDelayedCancelContext(ctx, e.settings.CancelDelay)
	defer stop()

	var res any
	var err error
	var errType ErrorType

	attempts := 1
	timeTracker := e.timeTrackerFactory()
	timeTracker.Start()

	// Use TrackedBackOff which delegates elapsed time checking to our tracker.
	// This ensures the MaxElapsedTime budget is measured according to the tracker's strategy.
	trackedBackoff := NewTrackedBackOff(e.settings, timeTracker)
	backoffCtx := backoff.WithContext(trackedBackoff, ctx)

	notify := func(err error, dur time.Duration) {
		for _, n := range append(e.notifier, notifier...) {
			n(err, dur)
		}

		logger.WithFields(log.Fields{
			"attempts": attempts,
		}).Warn(ctx, "retrying resource %s after error: %s", e.resource, err.Error())
		attempts++
	}

	//nolint:errcheck // we rely on the err variable in the closure
	_ = backoff.RetryNotify(func() error {
		res, err = f(delayedCtx)

		if err == nil {
			timeTracker.OnSuccess()

			return nil
		}

		if e.settings.MaxAttempts > 0 && attempts >= e.settings.MaxAttempts {
			return backoff.Permanent(err)
		}

		for _, check := range e.checks {
			errType = check(res, err)

			switch errType {
			case ErrorTypeOk:
				timeTracker.OnSuccess()

				return nil
			case ErrorTypeRetryable:
				timeTracker.OnError(err)

				return err
			case ErrorTypePermanent:
				return backoff.Permanent(err)
			case ErrorTypeUnknown:
				// try next check
			}
		}

		return backoff.Permanent(err)
	}, backoffCtx, notify)

	duration := timeTracker.Elapsed()
	logger = logger.WithFields(log.Fields{
		"attempts":        attempts,
		"duration_millis": duration.Milliseconds(),
	})

	// we're having an error after reaching the MaxAttempts and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk && e.settings.MaxAttempts > 0 && attempts > e.settings.MaxAttempts {
		logger.Warn(ctx, "crossed max attempts with an error on requesting resource %s after %d attempts in %s: %s", e.resource, attempts, duration, err.Error())

		return res, NewErrAttemptsExceeded(e.resource, attempts, duration, err)
	}

	// we're having an error after reaching the MaxElapsedTime and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk && e.settings.MaxElapsedTime > 0 && duration > e.settings.MaxElapsedTime {
		logger.Warn(ctx, "crossed max elapsed time with an error on requesting resource %s after %d attempts in %s: %s", e.resource, attempts, duration, err.Error())

		return res, NewErrMaxElapsedTimeExceeded(e.resource, attempts, duration, e.settings.MaxElapsedTime, err)
	}

	// we're still having an error and the error isn't good-natured
	if err != nil && errType != ErrorTypeOk {
		logger.Warn(ctx, "error on requesting resource %s after %d attempts in %s: %s", e.resource, attempts, duration, err.Error())

		return res, err
	}

	if attempts > 1 {
		logger.Info(ctx, "sent request to resource %s successful after %d attempts in %s", e.resource, attempts, duration)
	}

	return res, err
}
