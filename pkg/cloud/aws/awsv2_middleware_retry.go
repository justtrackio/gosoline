package aws

import (
	"context"
	"errors"
	"time"

	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/uuid"

	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	awsRetry "github.com/aws/aws-sdk-go-v2/aws/retry"
	smithyMiddleware "github.com/aws/smithy-go/middleware"
	"github.com/justtrackio/gosoline/pkg/log"
)

type attemptInfoKey struct{}

type attemptInfo struct {
	id       string
	resource *exec.ExecutableResource
	start    time.Time
	count    int
	lastErr  error
}

func getAttemptInfo(ctx context.Context) *attemptInfo {
	info := smithyMiddleware.GetStackValue(ctx, attemptInfoKey{})

	if info == nil {
		return nil
	}

	return info.(*attemptInfo)
}

func setAttemptInfo(ctx context.Context, info *attemptInfo) context.Context {
	return smithyMiddleware.WithStackValue(ctx, attemptInfoKey{}, info)
}

func increaseAttemptCount(ctx context.Context) (*attemptInfo, context.Context) {
	stats := getAttemptInfo(ctx)

	if stats != nil {
		stats.count++
		return stats, ctx
	}

	stats = &attemptInfo{
		start: time.Now(),
		count: 1,
	}

	ctx = smithyMiddleware.WithStackValue(ctx, attemptInfoKey{}, stats)

	return stats, ctx
}

func AttemptLoggerInitMiddleware(logger log.Logger, backoff *exec.BackoffSettings) smithyMiddleware.InitializeMiddleware {
	return smithyMiddleware.InitializeMiddlewareFunc("AttemptLoggerInit", func(ctx context.Context, input smithyMiddleware.InitializeInput, handler smithyMiddleware.InitializeHandler) (smithyMiddleware.InitializeOutput, smithyMiddleware.Metadata, error) {
		var err error
		var cancel context.CancelFunc
		var metadata smithyMiddleware.Metadata
		var output smithyMiddleware.InitializeOutput
		var deadline time.Time

		resource := &exec.ExecutableResource{
			Type: awsMiddleware.GetServiceID(ctx),
			Name: awsMiddleware.GetOperationName(ctx),
		}

		attempt := &attemptInfo{
			id:       uuid.New().NewV4(),
			start:    time.Now(),
			resource: resource,
		}
		ctx = setAttemptInfo(ctx, attempt)

		if backoff.CancelDelay > 0 {
			ctx = exec.WithDelayedCancelContext(ctx, backoff.CancelDelay)
			defer (ctx.(*exec.DelayedCancelContext)).Stop()
		}

		if backoff.MaxElapsedTime > 0 {
			deadline = attempt.start.Add(backoff.MaxElapsedTime)
			ctx, cancel = context.WithDeadline(ctx, deadline)

			defer cancel()
		}

		output, metadata, err = handler.HandleInitialize(ctx, input)

		now := time.Now()
		durationTook := now.Sub(attempt.start)

		if backoff.MaxElapsedTime > 0 && deadline.Before(now) && ctx.Err() == context.DeadlineExceeded {
			return output, metadata, exec.NewErrMaxElapsedTimeExceeded(attempt.resource, attempt.count, durationTook, backoff.MaxElapsedTime, err)
		}

		var maxAttemptsError *awsRetry.MaxAttemptsError
		if err != nil && errors.As(err, &maxAttemptsError) {
			return output, metadata, exec.NewErrAttemptsExceeded(attempt.resource, attempt.count, durationTook, err)
		}

		if attempt.count > 1 && err == nil {
			logger.
				WithContext(ctx).
				WithFields(log.Fields{
					"attempt_id": attempt.id,
					"resource":   attempt.resource.String(),
				}).
				Info("sent request to resource %s successful after %d attempts in %s", attempt.resource, attempt.count, durationTook)
		}

		return output, metadata, err
	})
}

func AttemptLoggerRetryMiddleware(logger log.Logger) smithyMiddleware.FinalizeMiddleware {
	return smithyMiddleware.FinalizeMiddlewareFunc("AttemptLoggerRetry", func(ctx context.Context, input smithyMiddleware.FinalizeInput, next smithyMiddleware.FinalizeHandler) (smithyMiddleware.FinalizeOutput, smithyMiddleware.Metadata, error) {
		var attempt *attemptInfo
		var metadata smithyMiddleware.Metadata
		var output smithyMiddleware.FinalizeOutput

		attempt, ctx = increaseAttemptCount(ctx)

		if attempt.count > 1 {
			duration := time.Since(attempt.start)
			logger.
				WithContext(ctx).
				WithFields(log.Fields{
					"attempt_id": attempt.id,
					"resource":   attempt.resource.String(),
				}).Warn("attempt number %d to request resource %s after %s cause of error: %s", attempt.count, attempt.resource, duration, attempt.lastErr)
		}

		output, metadata, attempt.lastErr = next.HandleFinalize(ctx, input)

		return output, metadata, attempt.lastErr
	})
}
