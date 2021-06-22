package aws

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/log"
	awsMiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	smithyMiddleware "github.com/aws/smithy-go/middleware"
	"time"
)

type attemptInfoKey struct{}

type attemptInfo struct {
	resouceName string
	start       time.Time
	count       int
	lastErr     error
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

type ErrRetryAttempsExceeded struct {
	ResourceName string
	Attempts     int
	DurationTook time.Duration
	Err          error
}

func NewErrRetryAttempsExceeded(resourceName string, attempts int, durationTook time.Duration, err error) *ErrRetryAttempsExceeded {
	return &ErrRetryAttempsExceeded{
		ResourceName: resourceName,
		Attempts:     attempts,
		DurationTook: durationTook,
		Err:          err,
	}
}

func (e *ErrRetryAttempsExceeded) Error() string {
	return fmt.Sprintf("sent request to resource %s failed after %d retries in %s: %s", e.ResourceName, e.Attempts, e.DurationTook, e.Err)
}

func (e *ErrRetryAttempsExceeded) Unwrap() error {
	return e.Err
}

func IsErrRetryAttempsExceeded(err error) bool {
	var errExpected *ErrRetryAttempsExceeded
	return errors.As(err, &errExpected)
}

type ErrRetryMaxElapsedTimeExceeded struct {
	ResourceName string
	Attempts     int
	DurationTook time.Duration
	DurationMax  time.Duration
	Err          error
}

func NewErrRetryMaxElapsedTimeExceeded(resourceName string, attempts int, durationTook time.Duration, durationMax time.Duration, err error) *ErrRetryMaxElapsedTimeExceeded {
	return &ErrRetryMaxElapsedTimeExceeded{
		ResourceName: resourceName,
		Attempts:     attempts,
		DurationTook: durationTook,
		DurationMax:  durationMax,
		Err:          err,
	}
}

func (e *ErrRetryMaxElapsedTimeExceeded) Error() string {
	return fmt.Sprintf("sent request to resource %s failed after %d retries in %s: retry max duration %s exceeded: %s", e.ResourceName, e.Attempts, e.DurationTook, e.DurationMax, e.Err)
}

func (e *ErrRetryMaxElapsedTimeExceeded) Unwrap() error {
	return e.Err
}

func IsErrRetryMaxElapsedTimeExceeded(err error) bool {
	var errExpected *ErrRetryMaxElapsedTimeExceeded
	return errors.As(err, &errExpected)
}

func AttemptLoggerInitMiddleware(logger log.Logger, clock clock.Clock, maxElapsedTime time.Duration) smithyMiddleware.InitializeMiddleware {
	return smithyMiddleware.InitializeMiddlewareFunc("AttemptLoggerInit", func(ctx context.Context, input smithyMiddleware.InitializeInput, handler smithyMiddleware.InitializeHandler) (smithyMiddleware.InitializeOutput, smithyMiddleware.Metadata, error) {
		var err error
		var cancel context.CancelFunc
		var metadata smithyMiddleware.Metadata
		var output smithyMiddleware.InitializeOutput

		serviceId := awsMiddleware.GetServiceID(ctx)
		operation := awsMiddleware.GetOperationName(ctx)
		resourceName := fmt.Sprintf("%s/%s", serviceId, operation)

		info := &attemptInfo{
			start:       clock.Now(),
			resouceName: resourceName,
		}
		ctx = setAttemptInfo(ctx, info)

		if maxElapsedTime > 0 {
			deadline := clock.Now().Add(maxElapsedTime)
			ctx, cancel = context.WithDeadline(ctx, deadline)

			defer cancel()
		}

		output, metadata, err = handler.HandleInitialize(ctx, input)
		durationTook := clock.Now().Sub(info.start)

		if ctx.Err() == context.DeadlineExceeded {
			return output, metadata, NewErrRetryMaxElapsedTimeExceeded(info.resouceName, info.count, durationTook, maxElapsedTime, err)
		}

		var maxAttemptsError *retry.MaxAttemptsError
		if err != nil && errors.As(err, &maxAttemptsError) {
			return output, metadata, NewErrRetryAttempsExceeded(info.resouceName, info.count, durationTook, err)
		}

		if info.count > 1 && err == nil {
			logger.WithContext(ctx).Info("sent request to resource %s successful after %d retries in %s", info.resouceName, info.count, durationTook)
		}

		return output, metadata, err
	})
}

func AttemptLoggerRetryMiddleware(logger log.Logger, clock clock.Clock) smithyMiddleware.FinalizeMiddleware {
	return smithyMiddleware.FinalizeMiddlewareFunc("AttemptLoggerRetry", func(ctx context.Context, input smithyMiddleware.FinalizeInput, next smithyMiddleware.FinalizeHandler) (smithyMiddleware.FinalizeOutput, smithyMiddleware.Metadata, error) {
		var info *attemptInfo
		var metadata smithyMiddleware.Metadata
		var output smithyMiddleware.FinalizeOutput

		info, ctx = increaseAttemptCount(ctx)

		if info.count > 1 {
			duration := clock.Now().Sub(info.start)
			logger.WithContext(ctx).Warn("attempt number %d to request resource %s after %s cause of error %s", info.count, info.resouceName, duration, info.lastErr)
		}

		output, metadata, info.lastErr = next.HandleFinalize(ctx, input)

		return output, metadata, info.lastErr
	})
}
