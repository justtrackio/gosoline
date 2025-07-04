package stream_test

import (
	"context"
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
)

func TestSqsInput_Run(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	var count int32
	waitReadDone := make(chan struct{})
	waitStopDone := make(chan struct{})
	waitRunDone := make(chan struct{})
	msg := &stream.Message{}

	queue := sqsMocks.NewQueue(t)
	queue.EXPECT().Receive(ctx, int32(1), int32(3)).
		RunAndReturn(func(_ context.Context, mrc int32, wt int32) ([]types.Message, error) {
			newCount := atomic.AddInt32(&count, 1)

			if newCount > mrc {
				<-waitStopDone

				return []types.Message{}, nil
			}

			return []types.Message{
				{
					Body:          aws.String(`{"body": "foobar"}`),
					MessageId:     aws.String(""),
					ReceiptHandle: aws.String(""),
				},
			}, nil
		})

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
		MaxNumberOfMessages: 1,
		WaitTime:            3,
		RunnerCount:         3,
	})

	go func() {
		err := input.Run(ctx)
		assert.NoError(t, err)

		close(waitRunDone)
	}()

	go func() {
		msg = <-input.Data()
		close(waitReadDone)
	}()

	<-waitReadDone
	input.Stop()
	close(waitStopDone)

	<-waitRunDone

	assert.Equal(t, "foobar", msg.Body)
}

func TestSqsInput_Run_Failure(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	var count int32
	waitRunDone := make(chan struct{})

	queue := sqsMocks.NewQueue(t)
	queue.EXPECT().Receive(matcher.Context, int32(10), int32(3)).
		RunAndReturn(func(_ context.Context, mrc int32, wt int32) ([]types.Message, error) {
			newCount := atomic.AddInt32(&count, 1)

			if newCount == 1 {
				return []types.Message{
					{
						Body:          aws.String(`{"body": "foobar"}`),
						ReceiptHandle: nil,
					},
				}, nil
			}

			return []types.Message{}, nil
		})

	healthCheckTimer := clock.NewHealthCheckTimerWithInterfaces(clock.NewFakeClock(), time.Minute)

	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, healthCheckTimer, &stream.SqsInputSettings{
		WaitTime:            3,
		RunnerCount:         3,
		MaxNumberOfMessages: 10,
	})

	go func() {
		err := input.Run(context.TODO())
		assert.Error(t, err)

		close(waitRunDone)
	}()

	<-waitRunDone
}
