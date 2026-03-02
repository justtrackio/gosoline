package stream_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/stream"
	streamMocks "github.com/justtrackio/gosoline/pkg/stream/mocks"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestDaemon(input stream.Input, handler stream.RetryHandler, output stream.Output) stream.ProducerRetryDaemon {
	return newTestDaemonWithWriters(input, handler, output, 1)
}

func newTestDaemonWithWriters(input stream.Input, handler stream.RetryHandler, output stream.Output, daemonWriterCount int) stream.ProducerRetryDaemon {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)
	writer := metric.NewWriter()
	name := "test"
	uuidMock := &uuidMocks.Uuid{}
	uuidMock.EXPECT().NewV4().Return("uuid").Maybe()

	return stream.NewProducerRetryDaemonWithInterfaces(
		name,
		logger,
		writer,
		uuidMock,
		input,
		handler,
		output,
		daemonWriterCount,
	)
}

func TestRetryOne_WithValidMessage(t *testing.T) {
	handler := new(streamMocks.RetryHandler)
	daemon := newTestDaemon(nil, handler, nil)

	msg := &stream.Message{
		Body: "body",
		Attributes: map[string]string{
			"goso.retry":    "true",
			"goso.retry.id": "uuid",
		},
	}

	handler.On("Put", mock.Anything, msg).Return(nil)

	err := daemon.RetryOne(context.Background(), msg)
	assert.NoError(t, err)
	handler.AssertCalled(t, "Put", mock.Anything, msg)
}

func TestRetryOne_WithInvalidMessage(t *testing.T) {
	handler := new(streamMocks.RetryHandler)
	daemon := newTestDaemon(nil, handler, nil)

	err := daemon.RetryOne(context.Background(), &stream.RawMessage{})
	assert.ErrorContains(t, err, "can not cast messages to message struct")
}

func TestRun_CancelsGracefully(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	input := new(streamMocks.Input)
	output := new(streamMocks.Output)

	// Create a channel that will be closed when Stop is called
	ch := make(chan *stream.Message)

	// Run should block until the context is done, then return
	input.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		// Wait for the context to be cancelled or the test to call Stop
		<-ch // This will unblock when ch is closed by Stop
	}).Return(nil)

	input.On("Data").Return((<-chan *stream.Message)(ch))

	input.On("Stop", mock.Anything).Run(func(args mock.Arguments) {
		close(ch)
	}).Return()

	daemon := newTestDaemon(input, nil, output)

	err := daemon.Run(ctx)
	assert.NoError(t, err)
	input.AssertExpectations(t)
}

func TestIngestDataFromSource_AcknowledgeableInput(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msg := &stream.Message{Body: "body"}

	input := new(streamMocks.AcknowledgeableInput)
	output := new(streamMocks.Output)

	// Simulate message data stream.
	ch := make(chan *stream.Message, 1)
	ch <- msg
	close(ch)

	input.On("Data").Return((<-chan *stream.Message)(ch))
	output.On("WriteOne", mock.Anything, mock.Anything).Return(nil)
	input.On("Ack", mock.Anything, msg, true).Return(nil)
	input.On("Run", mock.Anything).Return(nil)
	input.On("Stop", mock.Anything).Return()

	daemon := newTestDaemon(input, nil, output)
	err := daemon.Run(ctx) // IngestDataFromSource will be called during Run.
	assert.NoError(t, err)

	// Assert expectations for mocks
	input.AssertExpectations(t)
	output.AssertExpectations(t)
}

func TestParallelProcessing_MultipleDaemonWriters(t *testing.T) {
	const (
		numWriters    = 5
		numMessages   = 50
		writeDelay    = 10 * time.Millisecond
		maxExpectedMs = 300 // With 5 workers processing 50 messages at 10ms each, should take ~100ms, allow 300ms
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	input := new(streamMocks.AcknowledgeableInput)
	output := new(streamMocks.Output)

	// Create messages
	messages := make([]*stream.Message, numMessages)
	for i := 0; i < numMessages; i++ {
		messages[i] = &stream.Message{
			Body: fmt.Sprintf("message-%d", i),
			Attributes: map[string]string{
				"id": fmt.Sprintf("%d", i),
			},
		}
	}

	// Create channel with messages
	ch := make(chan *stream.Message, numMessages)
	for _, msg := range messages {
		ch <- msg
	}

	// Track concurrent writes to verify parallelism
	var (
		concurrentWrites int32
		maxConcurrent    int32
		writesCompleted  int32
	)

	// Mock setup
	input.On("Data").Return((<-chan *stream.Message)(ch))

	// Run should wait until Stop is called (which closes a done channel)
	runDone := make(chan struct{})
	input.On("Run", mock.Anything).Run(func(args mock.Arguments) {
		// Block until Stop is called
		<-runDone
	}).Return(nil)

	input.On("Stop", mock.Anything).Run(func(args mock.Arguments) {
		close(ch)
		close(runDone)
	}).Return()

	// Mock WriteOne to simulate slow writes and track concurrency
	output.On("WriteOne", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		// Increment concurrent counter
		current := atomic.AddInt32(&concurrentWrites, 1)

		// Track max concurrent
		for {
			maxVal := atomic.LoadInt32(&maxConcurrent)
			if current <= maxVal || atomic.CompareAndSwapInt32(&maxConcurrent, maxVal, current) {
				break
			}
		}

		// Simulate slow write operation
		time.Sleep(writeDelay)

		// Decrement concurrent counter
		atomic.AddInt32(&concurrentWrites, -1)
		atomic.AddInt32(&writesCompleted, 1)
	}).Return(nil).Times(numMessages)

	// Mock Ack for each message
	input.On("Ack", mock.Anything, mock.Anything, true).Return(nil).Times(numMessages)

	// Create daemon with multiple writers
	daemon := newTestDaemonWithWriters(input, nil, output, numWriters)

	// Measure execution time
	start := time.Now()

	// Run daemon in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- daemon.Run(ctx)
	}()

	// Give the daemon a moment to start
	time.Sleep(50 * time.Millisecond)

	// Wait for all writes to complete or timeout
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	shutdownTriggered := false
	daemonCompleted := false
	var daemonErr error

	for !daemonCompleted {
		// Check if daemon completed first (prioritize over context cancellation)
		select {
		case err := <-errChan:
			daemonErr = err
			daemonCompleted = true
		default:
		}

		if daemonCompleted {
			break
		}

		select {
		case <-ctx.Done():
			if !shutdownTriggered {
				require.FailNow(t, "test timed out", "only %d/%d messages processed", atomic.LoadInt32(&writesCompleted), numMessages)
			}
			// Context was cancelled because we finished processing, wait for daemon to exit
		case err := <-errChan:
			daemonErr = err
			daemonCompleted = true
		case <-ticker.C:
			completed := atomic.LoadInt32(&writesCompleted)
			if completed >= numMessages && !shutdownTriggered {
				// All messages processed, trigger shutdown
				shutdownTriggered = true
				cancel()
			}
		}
	}

	require.NoError(t, daemonErr)

	elapsed := time.Since(start)

	// Verify all messages were processed
	assert.Equal(t, int32(numMessages), atomic.LoadInt32(&writesCompleted), "all messages should be processed")

	// Verify parallel processing occurred
	maxConcurrentObserved := atomic.LoadInt32(&maxConcurrent)
	assert.GreaterOrEqual(t, maxConcurrentObserved, int32(2), "should have at least 2 concurrent writes")
	assert.LessOrEqual(t, maxConcurrentObserved, int32(numWriters), "should not exceed number of writers")

	// Verify performance: with 5 workers, 50 messages at 10ms each should take ~100ms
	// Without parallelism, it would take 500ms (50 * 10ms)
	// We allow up to maxExpectedMs for test flakiness
	assert.Less(t, elapsed.Milliseconds(), int64(maxExpectedMs), "parallel processing should be significantly faster than sequential (elapsed: %v)", elapsed)

	// Verify mock expectations
	input.AssertExpectations(t)
	output.AssertExpectations(t)
}
