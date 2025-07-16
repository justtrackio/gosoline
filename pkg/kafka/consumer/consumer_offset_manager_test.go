package consumer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

func TestOffsetManager_NotCommitting(t *testing.T) {
	var (
		pool        = coffin.New(t.Context())
		ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = mocks.NewReader(t)
	)
	reader.EXPECT().FetchMessage(matcher.Context).
		RunAndReturn(func(ctx context.Context) (kafka.Message, error) {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches), nil
		})
	reader.EXPECT().Close().Times(1).Return(nil)

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	manager := consumer.NewOffsetManager(
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		reader,
		2,
		time.Second,
		healthCheckTimer,
	)
	pool.GoWithContext("manager", manager.Start, coffin.WithContext(ctx))

	// 1st call to batch() should return a non-empty batch.
	assert.Equal(t, []kafka.Message{
		{
			Partition: 1,
			Offset:    1,
		},
		{
			Partition: 2,
			Offset:    2,
		},
	}, manager.Batch(ctx))

	// 2nd call to batch() should return an empty batch, since previous batch was not committed.
	assert.Equal(t, []kafka.Message{}, manager.Batch(ctx))

	err := pool.Wait()
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestOffsetManager_PartialCommit(t *testing.T) {
	var (
		pool        = coffin.New(t.Context())
		ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = mocks.NewReader(t)
	)
	reader.EXPECT().FetchMessage(matcher.Context).RunAndReturn(
		func(ctx context.Context) (kafka.Message, error) {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches), nil
		})
	reader.EXPECT().CommitMessages(matcher.Context, kafka.Message{Partition: 1, Offset: 1}).Return(nil).Times(1)
	reader.EXPECT().Close().Times(1).Return(nil)

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	manager := consumer.NewOffsetManager(
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		reader,
		2,
		time.Second,
		healthCheckTimer,
	)
	pool.GoWithContext("manager", manager.Start, coffin.WithContext(ctx))

	// 1st call to batch() should return a non-empty batch.
	assert.Equal(t, []kafka.Message{
		{
			Partition: 1,
			Offset:    1,
		},
		{
			Partition: 2,
			Offset:    2,
		},
	}, manager.Batch(ctx))

	assert.Nil(t, manager.Commit(ctx, kafka.Message{Partition: 1, Offset: 1}))

	// 2nd call to batch() should return an empty batch, since previous batch was not committed.
	assert.Equal(t, []kafka.Message{}, manager.Batch(ctx))

	err := pool.Wait()
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestOffsetManager_DoubleCommit(t *testing.T) {
	var (
		pool        = coffin.New(t.Context())
		ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = mocks.NewReader(t)
	)
	reader.EXPECT().FetchMessage(matcher.Context).RunAndReturn(
		func(ctx context.Context) (kafka.Message, error) {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches), nil
		})
	reader.EXPECT().CommitMessages(matcher.Context, kafka.Message{Partition: 1, Offset: 1}).Return(nil).Times(1)
	reader.EXPECT().CommitMessages(matcher.Context,
		kafka.Message{Partition: 1, Offset: 1},
		kafka.Message{Partition: 2, Offset: 2},
	).Return(nil).Times(1)
	reader.EXPECT().Close().Times(1).Return(nil)

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	manager := consumer.NewOffsetManager(
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		reader,
		2,
		time.Second,
		healthCheckTimer,
	)
	pool.GoWithContext("manager", manager.Start, coffin.WithContext(ctx))

	// 1st call to batch() should return a non-empty batch.
	assert.Equal(t, []kafka.Message{
		{
			Partition: 1,
			Offset:    1,
		},
		{
			Partition: 2,
			Offset:    2,
		},
	}, manager.Batch(ctx))

	assert.Nil(t, manager.Commit(ctx, kafka.Message{Partition: 1, Offset: 1}))
	assert.Nil(t, manager.Commit(ctx, kafka.Message{Partition: 1, Offset: 1}, kafka.Message{Partition: 2, Offset: 2}))

	// 2nd call to batch() should return non-empty batch, since previous batch was committed.
	assert.Equal(t, []kafka.Message{
		{
			Partition: 3,
			Offset:    3,
		},
		{
			Partition: 4,
			Offset:    4,
		},
	}, manager.Batch(ctx))

	err := pool.Wait()
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestOffsetManager_FullCommit(t *testing.T) {
	var (
		pool        = coffin.New(t.Context())
		ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = mocks.NewReader(t)
	)
	reader.EXPECT().FetchMessage(matcher.Context).RunAndReturn(
		func(ctx context.Context) (kafka.Message, error) {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches), nil
		})
	reader.EXPECT().CommitMessages(matcher.Context, kafka.Message{Partition: 1, Offset: 1}).Return(nil).Times(1)
	reader.EXPECT().CommitMessages(matcher.Context, kafka.Message{Partition: 2, Offset: 2}).Return(nil).Times(1)
	reader.EXPECT().Close().Times(1).Return(nil)

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	manager := consumer.NewOffsetManager(
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		reader,
		2,
		time.Second,
		healthCheckTimer,
	)
	pool.GoWithContext("manager", manager.Start, coffin.WithContext(ctx))

	// 1st call to batch() should return a non-empty batch.
	assert.Equal(t, []kafka.Message{
		{
			Partition: 1,
			Offset:    1,
		},
		{
			Partition: 2,
			Offset:    2,
		},
	}, manager.Batch(ctx))

	assert.Nil(t, manager.Commit(ctx, kafka.Message{Partition: 1, Offset: 1}))
	assert.Nil(t, manager.Commit(ctx, kafka.Message{Partition: 2, Offset: 2}))

	// 2nd call to batch() should return a non-empty batch, since previous batch was committed.
	assert.Equal(t, []kafka.Message{
		{
			Partition: 3,
			Offset:    3,
		},
		{
			Partition: 4,
			Offset:    4,
		},
	}, manager.Batch(ctx))

	// 3rd call to batch() should return an empty batch, since previous there are no more messages.
	assert.Equal(t, []kafka.Message{}, manager.Batch(ctx))

	err := pool.Wait()
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestOffsetManager_FetchMessageErrors(t *testing.T) {
	ctx := t.Context()

	var (
		reader    = mocks.NewReader(t)
		readerErr = errors.New("reader: failed")
	)

	reader.EXPECT().FetchMessage(ctx).RunAndReturn(func(ctx context.Context) (kafka.Message, error) {
		time.Sleep(time.Millisecond)

		return kafka.Message{}, readerErr
	})
	reader.EXPECT().Close().Times(1).Return(nil)

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	manager := consumer.NewOffsetManager(
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		reader,
		2,
		time.Second,
		healthCheckTimer,
	)
	assert.ErrorIs(t, manager.Start(ctx), readerErr)
}

func TestOffsetManager_FlushErrors(t *testing.T) {
	ctx := t.Context()

	var (
		reader    = mocks.NewReader(t)
		readerErr = errors.New("reader: failed")
	)

	reader.EXPECT().FetchMessage(ctx).RunAndReturn(func(ctx context.Context) (kafka.Message, error) {
		time.Sleep(time.Millisecond)

		return kafka.Message{}, readerErr
	})
	reader.EXPECT().Close().Times(1).Return(readerErr)

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	manager := consumer.NewOffsetManager(
		logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)),
		reader,
		2,
		time.Second,
		healthCheckTimer,
	)
	assert.ErrorIs(t, manager.Start(ctx), readerErr)
}

func OnFetch(_ context.Context, call int) kafka.Message {
	return kafka.Message{
		Partition: call + 1,
		Offset:    int64(call + 1),
	}
}
