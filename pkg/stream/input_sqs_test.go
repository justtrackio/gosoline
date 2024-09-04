package stream_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sqsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSqsInput_Run(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	var count int32
	waitReadDone := make(chan struct{})
	waitStopDone := make(chan struct{})
	waitRunDone := make(chan struct{})
	msg := &stream.Message{}

	queue := new(sqsMocks.Queue)
	queue.On("Receive", ctx, int32(1), int32(3)).Return(func(_ context.Context, mrc int32, wt int32) []types.Message {
		newCount := atomic.AddInt32(&count, 1)

		if newCount > mrc {
			<-waitStopDone
			return []types.Message{}
		}

		return []types.Message{
			{
				Body:          aws.String(`{"body": "foobar"}`),
				MessageId:     aws.String(""),
				ReceiptHandle: aws.String(""),
			},
		}
	}, nil)

	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, &stream.SqsInputSettings{
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

	count := 0
	waitRunDone := make(chan struct{})

	queue := new(sqsMocks.Queue)
	queue.On("Receive", mock.AnythingOfType("*context.emptyCtx"), int32(10), int32(3)).Return(func(_ context.Context, mrc int32, wt int32) []types.Message {
		count++

		if count == 1 {
			return []types.Message{
				{
					Body:          aws.String(`{"body": "foobar"}`),
					ReceiptHandle: nil,
				},
			}
		}

		return []types.Message{}
	}, nil)

	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.MessageUnmarshaller, &stream.SqsInputSettings{
		WaitTime:    3,
		RunnerCount: 3,
	})

	go func() {
		err := input.Run(context.TODO())
		assert.Error(t, err)

		close(waitRunDone)
	}()

	<-waitRunDone
}
