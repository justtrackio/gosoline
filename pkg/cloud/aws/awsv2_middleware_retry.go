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
			var stop exec.StopFunc
			ctx, stop = exec.WithDelayedCancelContext(ctx, backoff.CancelDelay)
			defer stop()
		}

		if backoff.MaxElapsedTime > 0 {
			var stop exec.StopFunc
			deadline = attempt.start.Add(backoff.MaxElapsedTime)
			ctx, stop = exec.WithStoppableDeadlineContext(ctx, deadline)

			defer stop()
		}

		output, metadata, err = handler.HandleInitialize(ctx, input)

		now := time.Now()
		durationTook := now.Sub(attempt.start)

		if backoff.MaxElapsedTime > 0 && deadline.Before(now) && ctx.Err() == context.DeadlineExceeded {
			return output, metadata, exec.NewErrMaxElapsedTimeExceeded(attempt.resource, attempt.count, durationTook, backoff.MaxElapsedTime, ctx.Err())
		}

		var maxAttemptsError *awsRetry.MaxAttemptsError
		if err != nil && errors.As(err, &maxAttemptsError) {
			return output, metadata, exec.NewErrAttemptsExceeded(attempt.resource, attempt.count, durationTook, err)
		}

		// if it was 1 attempt only, don't log anything
		if attempt.count == 1 {
			return output, metadata, err
		}

		logger = logger.WithContext(ctx).WithFields(log.Fields{
			"attempt_id": attempt.id,
			"resource":   attempt.resource.String(),
		})

		if err != nil {
			logger.Warn("sent request to resource %s finally failed after %d attempts in %s: %s", attempt.resource, attempt.count, durationTook, err)

			return output, metadata, err
		}

		logger.Warn("sent request to resource %s successful after %d attempts in %s", attempt.resource, attempt.count, durationTook)

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
				}).Warn("attempt number %d to request resource %s failed after %s cause of error: %s", attempt.count-1, attempt.resource, duration, attempt.lastErr)
		}

		output, metadata, attempt.lastErr = next.HandleFinalize(ctx, input)

		return output, metadata, attempt.lastErr
	})
}
