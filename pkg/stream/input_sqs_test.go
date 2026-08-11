package stream_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	sqsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSqsInput_Run(t *testing.T) {
	testCases := []struct {
		name       string
		message    types.Message
		assertions func(t *testing.T, msg *stream.Message)
	}{
		{
			name: "basic functionality",
			message: types.Message{
				Body:          aws.String(`{"body": "foobar"}`),
				MessageId:     aws.String("message-id"),
				ReceiptHandle: aws.String("receipt-handle"),
			},
			assertions: func(t *testing.T, msg *stream.Message) {
				assert.Equal(t, "foobar", msg.Body)
			},
		},
		{
			name: "with ApproximateReceiveCount",
			message: types.Message{
				Body:          aws.String(`{"body": "foobar"}`),
				MessageId:     aws.String("test-message-id"),
				ReceiptHandle: aws.String("test-receipt-handle"),
				Attributes: map[string]string{
					"ApproximateReceiveCount": "2",
				},
			},
			assertions: func(t *testing.T, msg *stream.Message) {
				assert.Equal(t, "foobar", msg.Body)
				assert.Equal(t, "test-message-id", msg.Attributes[stream.AttributeSqsMessageId])
				assert.Equal(t, "test-receipt-handle", msg.Attributes[stream.AttributeSqsReceiptHandle])
				assert.Equal(t, "2", msg.Attributes[stream.AttributeSqsApproximateReceiveCount])
			},
		},
		{
			name: "without ApproximateReceiveCount",
			message: types.Message{
				Body:          aws.String(`{"body": "foobar"}`),
				MessageId:     aws.String("test-message-id"),
				ReceiptHandle: aws.String("test-receipt-handle"),
			},
			assertions: func(t *testing.T, msg *stream.Message) {
				assert.Equal(t, "foobar", msg.Body)
				assert.Equal(t, "test-message-id", msg.Attributes[stream.AttributeSqsMessageId])
				assert.Equal(t, "test-receipt-handle", msg.Attributes[stream.AttributeSqsReceiptHandle])
				// ApproximateReceiveCount should not be present when not provided by SQS
				_, exists := msg.Attributes[stream.AttributeSqsApproximateReceiveCount]
				assert.False(t, exists)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

			var count int32
			waitStopDone := make(chan struct{})
			waitRunDone := make(chan struct{})

			queue := sqsMocks.NewQueue(t)
			queue.EXPECT().Receive(ctx, int32(1), int32(3)).
				RunAndReturn(func(_ context.Context, mrc, wt int32) ([]types.Message, error) {
					newCount := atomic.AddInt32(&count, 1)

					if newCount > mrc {
						<-waitStopDone

						return []types.Message{}, nil
					}

					return []types.Message{tc.message}, nil
				})

			healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)
			receivedMessages := make(chan *stream.Message, 3)
			if tc.message.ReceiptHandle != nil && *tc.message.ReceiptHandle != "" {
				queue.EXPECT().DeleteMessage(matcher.Context, *tc.message.ReceiptHandle).Return(nil).Once()
			}

			input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
				MaxNumberOfMessages: 1,
				WaitTime:            3,
				GraceTime:           time.Second,
				RunnerCount:         3,
			})

			go func() {
				err := input.Run(ctx, func(_ context.Context, received *stream.Message) bool {
					receivedMessages <- received

					return true
				})
				assert.NoError(t, err)

				close(waitRunDone)
			}()

			msg := <-receivedMessages
			input.Stop(ctx)
			close(waitStopDone)

			<-waitRunDone

			tc.assertions(t, msg)
		})
	}
}

func TestSqsInput_Run_InvalidMetadata(t *testing.T) {
	testCases := []struct {
		name       string
		messageId  *string
		receipt    *string
		logMessage string
	}{
		{
			name:       "nil message id",
			messageId:  nil,
			receipt:    aws.String("receipt-handle"),
			logMessage: "could not process sqs message: message id is missing",
		},
		{
			name:       "empty message id",
			messageId:  aws.String(""),
			receipt:    aws.String("receipt-handle"),
			logMessage: "could not process sqs message: message id is missing",
		},
		{
			name:       "nil receipt handle",
			messageId:  aws.String("message-id"),
			receipt:    nil,
			logMessage: "could not process sqs message: receipt handle is missing",
		},
		{
			name:       "empty receipt handle",
			messageId:  aws.String("message-id"),
			receipt:    aws.String(""),
			logMessage: "could not process sqs message: receipt handle is missing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
			messages := []types.Message{
				{
					Body:          aws.String(`{"body": "invalid"}`),
					MessageId:     tc.messageId,
					ReceiptHandle: tc.receipt,
				},
				{
					Body:          aws.String(`{"body": "valid"}`),
					MessageId:     aws.String("valid-message-id"),
					ReceiptHandle: aws.String("valid-receipt-handle"),
				},
			}

			queue := sqsMocks.NewQueue(t)
			queue.EXPECT().Receive(ctx, int32(2), int32(3)).Return(messages, nil).Once()
			queue.EXPECT().DeleteMessage(matcher.Context, "valid-receipt-handle").Return(nil).Once()

			healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)
			input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
				MaxNumberOfMessages: 2,
				WaitTime:            3,
				GraceTime:           time.Second,
				RunnerCount:         1,
			})

			processed := make([]string, 0, 1)
			err := input.Run(ctx, func(_ context.Context, msg *stream.Message) bool {
				processed = append(processed, msg.Body)
				input.Stop(ctx)

				return true
			})

			assert.NoError(t, err)
			assert.Equal(t, []string{"valid"}, processed)
			logger.AssertCalled(t, "Error", mock.Anything, tc.logMessage)
		})
	}
}

func TestSqsInput_Run_ContinuesAfterDeleteMessageFailure(t *testing.T) {
	ctx := t.Context()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	deleteErr := errors.New("delete failed")
	messages := []types.Message{
		{
			Body:          aws.String(`{"body": "first"}`),
			MessageId:     aws.String("first-message-id"),
			ReceiptHandle: aws.String("first-receipt-handle"),
		},
		{
			Body:          aws.String(`{"body": "second"}`),
			MessageId:     aws.String("second-message-id"),
			ReceiptHandle: aws.String("second-receipt-handle"),
		},
	}

	queue := sqsMocks.NewQueue(t)
	queue.EXPECT().Receive(ctx, int32(2), int32(3)).Return(messages, nil).Once()
	queue.EXPECT().DeleteMessage(matcher.Context, "first-receipt-handle").Return(deleteErr).Once()
	queue.EXPECT().DeleteMessage(matcher.Context, "second-receipt-handle").Return(nil).Once()

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)
	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
		MaxNumberOfMessages: 2,
		WaitTime:            3,
		GraceTime:           time.Second,
		RunnerCount:         1,
	})

	processed := make([]string, 0, len(messages))
	err := input.Run(ctx, func(_ context.Context, msg *stream.Message) bool {
		processed = append(processed, msg.Body)
		if len(processed) == len(messages) {
			input.Stop(ctx)
		}

		return true
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, processed)
}

func TestSqsInput_Run_BatchAcknowledgement(t *testing.T) {
	ctx := t.Context()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	messages := []types.Message{
		{
			Body:          aws.String(`{"body": "first"}`),
			MessageId:     aws.String("first-message-id"),
			ReceiptHandle: aws.String("first-receipt-handle"),
		},
		{
			Body:          aws.String(`{"body": "second"}`),
			MessageId:     aws.String("second-message-id"),
			ReceiptHandle: aws.String("second-receipt-handle"),
		},
	}

	queue := sqsMocks.NewQueue(t)
	queue.EXPECT().Receive(ctx, int32(2), int32(3)).Return(messages, nil).Once()
	queue.EXPECT().DeleteMessageBatch(matcher.Context, []string{"first-receipt-handle"}).Return(nil).Once()

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)
	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
		MaxNumberOfMessages: 2,
		WaitTime:            3,
		GraceTime:           time.Second,
		RunnerCount:         1,
		AcknowledgementMode: stream.SqsAcknowledgementModeBatch,
	})

	processed := make([]string, 0, len(messages))
	err := input.Run(ctx, func(_ context.Context, msg *stream.Message) bool {
		processed = append(processed, msg.Body)
		if len(processed) == len(messages) {
			input.Stop(ctx)
		}

		return msg.Body == "first"
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, processed)
}

func TestSqsInput_Run_DeleteMessageCanceledDuringShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	message := types.Message{
		Body:          aws.String(`{"body": "foobar"}`),
		MessageId:     aws.String("message-id"),
		ReceiptHandle: aws.String("receipt-handle"),
	}

	queue := sqsMocks.NewQueue(t)
	queue.EXPECT().Receive(ctx, int32(1), int32(3)).Return([]types.Message{message}, nil).Once()
	queue.EXPECT().DeleteMessage(matcher.Context, "receipt-handle").RunAndReturn(func(deleteCtx context.Context, _ string) error {
		assert.NoError(t, deleteCtx.Err(), "acknowledgement context should remain active during shutdown grace period")

		return context.Canceled
	}).Once()

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)
	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
		MaxNumberOfMessages: 1,
		WaitTime:            3,
		GraceTime:           time.Second,
		RunnerCount:         1,
	})

	err := input.Run(ctx, func(context.Context, *stream.Message) bool {
		cancel()
		input.Stop(ctx)

		return true
	})

	assert.NoError(t, err)
}

func TestSqsInput_HealthWhileIdleAndStuck(t *testing.T) {
	ctx := t.Context()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	fakeClock := clock.NewFakeClock()
	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(fakeClock, time.Minute)
	receiveStarted := make(chan struct{})
	releaseReceive := make(chan struct{})

	queue := sqsMocks.NewQueue(t)
	queue.EXPECT().Receive(ctx, int32(10), int32(3)).RunAndReturn(func(context.Context, int32, int32) ([]types.Message, error) {
		receiveStarted <- struct{}{}
		<-releaseReceive

		return nil, nil
	})

	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
		WaitTime:            3,
		GraceTime:           time.Second,
		RunnerCount:         1,
		MaxNumberOfMessages: 10,
	})
	runDone := make(chan error, 1)
	go func() {
		runDone <- input.Run(ctx, func(context.Context, *stream.Message) bool {
			return true
		})
	}()

	<-receiveStarted
	for range 3 {
		fakeClock.Advance(45 * time.Second)
		assert.True(t, input.IsHealthy())
		releaseReceive <- struct{}{}
		<-receiveStarted
	}

	fakeClock.Advance(time.Minute + time.Millisecond)
	assert.False(t, input.IsHealthy())

	input.Stop(ctx)
	releaseReceive <- struct{}{}
	assert.NoError(t, <-runDone)
}
