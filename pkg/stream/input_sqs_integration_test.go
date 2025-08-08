package stream

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	sqsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
)

func TestSqsInput_RunLoop_IntegrationTest(t *testing.T) {
	t.Run("fatal error terminates loop", func(t *testing.T) {
		ctx := context.Background()
		
		// Create mocks
		mockQueue := &sqsMocks.Queue{}
		mockLogger := &logMocks.Logger{}

		// Set up fatal error expectation
		fatalErr := &types.QueueDoesNotExist{Message: aws.String("Queue does not exist")}
		mockQueue.On("Receive", mock.Anything, mock.Anything, mock.Anything).Return(nil, fatalErr)
		
		// Expect error logging
		mockLogger.On("Error", "fatal error while receiving messages from sqs, terminating: %w", fatalErr).Return()
		mockLogger.On("Info", "leaving sqs input runner").Return()
		
		// Create SQS input with mocks
		input := &sqsInput{
			logger:           mockLogger,
			queue:            mockQueue,
			settings:         &SqsInputSettings{MaxNumberOfMessages: 10, WaitTime: 1},
			unmarshaler:      unmarshallers[UnmarshallerMsg],
			healthCheckTimer: &mockHealthCheckTimer{},
			channel:          make(chan *Message),
			stopped:          0,
		}

		// Run the loop and expect it to return an error
		err := input.runLoop(ctx)
		
		// Assertions
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fatal sqs receive error")
		assert.Contains(t, err.Error(), "Queue does not exist")
		
		// Verify that the mocks were called as expected
		mockQueue.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("recoverable error continues loop until stopped", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		
		// Create mocks
		mockQueue := &sqsMocks.Queue{}
		mockLogger := &logMocks.Logger{}

		// Set up recoverable error followed by context cancellation
		recoverableErr := fmt.Errorf("network timeout")
		mockQueue.On("Receive", mock.Anything, mock.Anything, mock.Anything).Return(nil, recoverableErr)
		
		// Expect error logging but no fatal error return
		mockLogger.On("Error", "could not get messages from sqs: %w", recoverableErr).Return()
		mockLogger.On("Info", "leaving sqs input runner").Return()
		
		// Create SQS input with mocks
		input := &sqsInput{
			logger:           mockLogger,
			queue:            mockQueue,
			settings:         &SqsInputSettings{MaxNumberOfMessages: 10, WaitTime: 1},
			unmarshaler:      unmarshallers[UnmarshallerMsg],
			healthCheckTimer: &mockHealthCheckTimer{},
			channel:          make(chan *Message),
			stopped:          0,
		}

		// Cancel context after a short delay to stop the loop
		go func() {
			time.Sleep(100 * time.Millisecond)
			atomic.StoreInt32(&input.stopped, 1)
			cancel()
		}()

		// Run the loop and expect it to return nil (graceful shutdown)
		err := input.runLoop(ctx)
		
		// Assertions - should return nil for graceful shutdown, not the recoverable error
		assert.NoError(t, err)
		
		// Verify that error was logged at least once
		mockLogger.AssertExpectations(t)
	})
}

// Simple mock for health check timer
type mockHealthCheckTimer struct{}

func (m *mockHealthCheckTimer) MarkHealthy() {}
func (m *mockHealthCheckTimer) IsHealthy() bool { return true }