package aws_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/middleware"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestAttemptLoggerRequestBudgetUsesRealClock(t *testing.T) {
	originalClock := clock.Provider
	t.Cleanup(func() { clock.Provider = originalClock })
	execute := func(ctx context.Context, budget time.Duration, request func(context.Context) error) error {
		init := aws.AttemptLoggerInitMiddleware(log.NewLogger(), &exec.BackoffSettings{MaxElapsedTime: budget})
		_, _, err := init.HandleInitialize(ctx, middleware.InitializeInput{}, middleware.InitializeHandlerFunc(func(ctx context.Context, _ middleware.InitializeInput) (middleware.InitializeOutput, middleware.Metadata, error) {
			return middleware.InitializeOutput{}, middleware.Metadata{}, request(ctx)
		}))

		return err
	}
	t.Run("advanced application clock does not cancel a request", func(t *testing.T) {
		clock.Provider = clock.NewFakeClockAt(time.Now().Add(time.Hour))
		err := execute(t.Context(), time.Second, func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
				return nil
			}
		})
		require.NoError(t, err)
	})
	t.Run("network budget expires while the application clock is stopped", func(t *testing.T) {
		clock.Provider = clock.NewFakeClockAt(time.Now())
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		start := time.Now()
		err := execute(ctx, 10*time.Millisecond, func(ctx context.Context) error {
			<-ctx.Done()

			return ctx.Err()
		})
		var exceeded *exec.ErrMaxElapsedTimeExceeded
		require.ErrorAs(t, err, &exceeded)
		require.Less(t, time.Since(start), 500*time.Millisecond)
	})
	t.Run("returned streaming response keeps its context", func(t *testing.T) {
		clock.Provider = clock.NewFakeClockAt(time.Now())
		var responseContext context.Context
		err := execute(t.Context(), 10*time.Millisecond, func(ctx context.Context) error {
			responseContext = ctx

			return nil
		})
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
		require.NoError(t, responseContext.Err())
	})
	t.Run("parent cancellation is preserved", func(t *testing.T) {
		clock.Provider = clock.NewFakeClockAt(time.Now())
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		err := execute(ctx, time.Minute, func(ctx context.Context) error {
			cancel()
			<-ctx.Done()

			return ctx.Err()
		})
		require.ErrorIs(t, err, context.Canceled)
	})
}
