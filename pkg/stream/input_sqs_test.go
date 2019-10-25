package stream_test

import (
	"context"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	sqsMocks "github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestSqsInput_Run(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	count := 0
	maxReceiveCount := 1
	waitReadDone := make(chan struct{})
	waitStopDone := make(chan struct{})
	waitRunDone := make(chan struct{})
	msg := &stream.Message{}

	queue := new(sqsMocks.Queue)
	queue.On("Receive", mock.AnythingOfType("*context.emptyCtx"), int64(3)).Return(func(_ context.Context, wt int64) []*sqs.Message {
		count++

		if count > maxReceiveCount {
			<-waitStopDone
			return []*sqs.Message{}
		}

		return []*sqs.Message{
			{
				Body:          aws.String(`{"body": "foobar"}`),
				ReceiptHandle: aws.String(""),
			},
		}
	}, nil)

	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.SqsInputSettings{
		WaitTime:    int64(3),
		RunnerCount: 3,
	})

	go func() {
		err := input.Run(context.TODO())
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
	logger := monMocks.NewLoggerMockedAll()

	count := 0
	waitRunDone := make(chan struct{})

	queue := new(sqsMocks.Queue)
	queue.On("Receive", mock.AnythingOfType("*context.emptyCtx"), int64(3)).Return(func(_ context.Context, wt int64) []*sqs.Message {
		count++

		if count == 1 {
			return []*sqs.Message{
				{
					Body:          aws.String(`{"body": "foobar"}`),
					ReceiptHandle: nil,
				},
			}
		}

		return []*sqs.Message{}
	}, nil)

	input := stream.NewSqsInputWithInterfaces(logger, queue, stream.SqsInputSettings{
		WaitTime:    int64(3),
		RunnerCount: 3,
	})

	go func() {
		err := input.Run(context.TODO())
		assert.Error(t, err)

		close(waitRunDone)
	}()

	<-waitRunDone
}
