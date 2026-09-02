package stream_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	redisMocks "github.com/justtrackio/gosoline/pkg/redis/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRedisListInput_RunConcurrently(t *testing.T) {
	input, redisMock := setupRedisListInput(t, 2)

	var readCount atomic.Int32
	redisMock.EXPECT().BLPop(matcher.Context, time.Second, "my-list").RunAndReturn(func(ctx context.Context, _ time.Duration, _ ...string) ([]string, error) {
		switch readCount.Add(1) {
		case 1:
			return []string{"my-list", `{"body":"one"}`}, nil
		case 2:
			return []string{"my-list", `{"body":"two"}`}, nil
		case 3:
			return []string{"my-list", `{"body":"three"}`}, nil
		default:
			time.Sleep(10 * time.Millisecond)

			return nil, nil
		}
	}).Maybe()

	var lock sync.Mutex
	active := 0
	maxActive := 0
	processed := 0
	done := make(chan error, 1)
	go func() {
		done <- input.Run(t.Context(), func(_ context.Context, _ *stream.Message) bool {
			lock.Lock()
			active++
			maxActive = max(maxActive, active)
			processed++
			if processed == 3 {
				input.Stop(t.Context())
			}
			lock.Unlock()

			time.Sleep(10 * time.Millisecond)

			lock.Lock()
			active--
			lock.Unlock()

			return true
		})
	}()

	require.NoError(t, <-done)
	assert.Equal(t, 2, maxActive)
}

func TestRedisListInput_StopDoesNotCancelBlockingRead(t *testing.T) {
	input, redisMock := setupRedisListInput(t, 1)

	started := make(chan struct{})
	release := make(chan struct{})
	redisMock.EXPECT().BLPop(matcher.Context, time.Second, "my-list").RunAndReturn(func(ctx context.Context, _ time.Duration, _ ...string) ([]string, error) {
		close(started)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return nil, nil
		}
	}).Once()

	done := make(chan error, 1)
	go func() {
		done <- input.Run(t.Context(), func(_ context.Context, _ *stream.Message) bool {
			return true
		})
	}()

	<-started
	input.Stop(t.Context())

	select {
	case err := <-done:
		require.Failf(t, "Run returned before BLPop completed", "unexpected error: %v", err)
	case <-time.After(10 * time.Millisecond):
	}

	close(release)
	require.NoError(t, <-done)
}

func TestRedisListInput_RunProcessesMessagePoppedDuringStop(t *testing.T) {
	input, redisMock := setupRedisListInput(t, 1)

	redisMock.EXPECT().BLPop(matcher.Context, time.Second, "my-list").RunAndReturn(func(ctx context.Context, _ time.Duration, _ ...string) ([]string, error) {
		input.Stop(t.Context())

		// Give Stop's cancellation goroutine time to run before returning the popped message.
		select {
		case <-ctx.Done():
		case <-time.After(10 * time.Millisecond):
		}

		return []string{"my-list", `{"body":"popped"}`}, nil
	}).Once()

	processed := make(chan *stream.Message, 1)
	require.NoError(t, input.Run(t.Context(), func(_ context.Context, msg *stream.Message) bool {
		processed <- msg

		return true
	}))

	msg := <-processed
	assert.Equal(t, "popped", msg.Body)
}

func setupRedisListInput(t *testing.T, runnerCount int) (stream.Input, *redisMocks.Client) {
	t.Helper()

	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	metricWriter := metricMocks.NewWriter(t)
	metricWriter.EXPECT().Write(matcher.Context, mock.Anything).Return().Maybe()
	redisMock := redisMocks.NewClient(t)
	redisMock.EXPECT().LLen(matcher.Context, "my-list").Return(int64(0), nil).Maybe()

	input := stream.NewRedisListInputWithInterfaces(
		cfg.New(),
		loggerMock,
		redisMock,
		metricWriter,
		&stream.RedisListInputSettings{
			Key:         "my-list",
			WaitTime:    time.Second,
			RunnerCount: runnerCount,
		},
		clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute),
	)

	return input, redisMock
}
