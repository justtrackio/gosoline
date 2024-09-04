package consumer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOffsetManager_NotCommitting(t *testing.T) {
	var (
		pool        = coffin.New()
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = &mocks.Reader{}
	)
	reader.On("FetchMessage", mock.AnythingOfType("*context.timerCtx")).Return(
		func(ctx context.Context) kafka.Message {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches)
		},
		func(ctx context.Context) error {
			return nil
		},
	)
	reader.On("Close").Times(1).Return(nil)
	defer reader.AssertExpectations(t)

	manager := consumer.NewOffsetManager(logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)), reader, 2, time.Second)
	pool.GoWithContext(ctx, manager.Start)

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

	_ = pool.Wait()
}

func TestOffsetManager_PartialCommit(t *testing.T) {
	var (
		pool        = coffin.New()
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = &mocks.Reader{}
	)
	reader.On("FetchMessage", mock.AnythingOfType("*context.timerCtx")).Return(
		func(ctx context.Context) kafka.Message {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches)
		},
		func(ctx context.Context) error {
			return nil
		},
	)
	reader.On("CommitMessages", mock.AnythingOfType("*context.timerCtx"), kafka.Message{Partition: 1, Offset: 1}).Return(nil).Times(1)
	reader.On("Close").Times(1).Return(nil)
	defer reader.AssertExpectations(t)

	manager := consumer.NewOffsetManager(logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)), reader, 2, time.Second)
	pool.GoWithContext(ctx, manager.Start)

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

	_ = pool.Wait()
}

func TestOffsetManager_DoubleCommit(t *testing.T) {
	var (
		pool        = coffin.New()
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = &mocks.Reader{}
	)
	reader.On("FetchMessage", mock.AnythingOfType("*context.timerCtx")).Return(
		func(ctx context.Context) kafka.Message {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches)
		},
		func(ctx context.Context) error {
			return nil
		},
	)
	reader.On("CommitMessages", mock.AnythingOfType("*context.timerCtx"), kafka.Message{Partition: 1, Offset: 1}).Return(nil).Times(1)
	reader.On("CommitMessages", mock.AnythingOfType("*context.timerCtx"),
		kafka.Message{Partition: 1, Offset: 1},
		kafka.Message{Partition: 2, Offset: 2},
	).Return(nil).Times(1)
	reader.On("Close").Times(1).Return(nil)
	defer reader.AssertExpectations(t)

	manager := consumer.NewOffsetManager(logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)), reader, 2, time.Second)
	pool.GoWithContext(ctx, manager.Start)

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

	_ = pool.Wait()
}

func TestOffsetManager_FullCommit(t *testing.T) {
	var (
		pool        = coffin.New()
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	)
	defer cancel()

	var (
		readerFetches = 0
		reader        = &mocks.Reader{}
	)
	reader.On("FetchMessage", mock.AnythingOfType("*context.timerCtx")).Return(
		func(ctx context.Context) kafka.Message {
			defer func() {
				time.Sleep(time.Millisecond)
				readerFetches += 1
			}()

			return OnFetch(ctx, readerFetches)
		},
		func(ctx context.Context) error {
			return nil
		},
	)
	reader.On("CommitMessages", mock.AnythingOfType("*context.timerCtx"), kafka.Message{Partition: 1, Offset: 1}).Return(nil).Times(1)
	reader.On("CommitMessages", mock.AnythingOfType("*context.timerCtx"), kafka.Message{Partition: 2, Offset: 2}).Return(nil).Times(1)
	reader.On("Close").Times(1).Return(nil)
	defer reader.AssertExpectations(t)

	manager := consumer.NewOffsetManager(logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)), reader, 2, time.Second)
	pool.GoWithContext(ctx, manager.Start)

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

	_ = pool.Wait()
}

func TestOffsetManager_FetchMessageErrors(t *testing.T) {
	ctx := context.Background()

	var (
		reader    = &mocks.Reader{}
		readerErr = errors.New("reader: failed")
	)

	reader.On("FetchMessage", ctx).Return(
		func(ctx context.Context) kafka.Message {
			time.Sleep(time.Millisecond)

			return kafka.Message{}
		},
		func(ctx context.Context) error {
			return readerErr
		},
	)
	reader.On("Close").Times(1).Return(nil)
	defer reader.AssertExpectations(t)

	manager := consumer.NewOffsetManager(logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)), reader, 2, time.Second)
	assert.ErrorIs(t, manager.Start(ctx), readerErr)
}

func TestOffsetManager_FlushErrors(t *testing.T) {
	ctx := context.Background()

	var (
		reader    = &mocks.Reader{}
		readerErr = errors.New("reader: failed")
	)

	reader.On("FetchMessage", ctx).Return(
		func(ctx context.Context) kafka.Message {
			time.Sleep(time.Millisecond)

			return kafka.Message{}
		},
		func(ctx context.Context) error {
			return readerErr
		},
	)
	reader.On("Close").Times(1).Return(readerErr)
	defer reader.AssertExpectations(t)

	manager := consumer.NewOffsetManager(logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t)), reader, 2, time.Second)
	assert.ErrorIs(t, manager.Start(ctx), readerErr)
}

func OnFetch(ctx context.Context, call int) kafka.Message {
	return kafka.Message{
		Partition: call + 1,
		Offset:    int64(call + 1),
	}
}
