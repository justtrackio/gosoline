package sqs_test

import (
	"context"
	awsMocks "github.com/applike/gosoline/pkg/cloud/aws/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	gosoSqs "github.com/applike/gosoline/pkg/sqs"
	sqsMocks "github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
)

func TestRunQueueTestSuite(t *testing.T) {
	suite.Run(t, new(queueTestSuite))
}

type queueTestSuite struct {
	suite.Suite
	executor *awsMocks.Executor
	queue    gosoSqs.Queue
}

func (qs *queueTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMockedAll()
	sqsClient := new(sqsMocks.SQSAPI)
	qs.executor = new(awsMocks.Executor)
	qs.queue = gosoSqs.NewWithInterfaces(logger, sqsClient, qs.executor, &gosoSqs.Properties{
		Url: "http://foo.bar.baz",
	})
}

func (qs *queueTestSuite) TestSendBatch_EmptyBatch() {
	msgs := make([]*gosoSqs.Message, 0)

	err := qs.queue.SendBatch(context.Background(), msgs)
	qs.Nil(err)
}

func (qs *queueTestSuite) TestSendBatch_OneMsgRequestTooLongError() {
	msgs := make([]*gosoSqs.Message, 0)

	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           aws.Int64(int64(1)),
		MessageGroupId:         aws.String("group-1"),
		MessageDeduplicationId: aws.String("hash-1"),
		Body:                   aws.String("foo"),
	})
	awsErr := awserr.New(sqs.ErrCodeBatchRequestTooLong, "foo", nil)
	qs.executor.
		On("Execute", context.Background(), mock.AnythingOfType("aws.RequestFunction")).
		Once().
		Return(nil, awsErr)
	err := qs.queue.SendBatch(context.Background(), msgs)
	qs.Equal(awsErr, err)
}

func (qs *queueTestSuite) TestSendBatch_ThreeMsgRequestTooLongError() {
	msgs := make([]*gosoSqs.Message, 0)

	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           aws.Int64(int64(1)),
		MessageGroupId:         aws.String("group-1"),
		MessageDeduplicationId: aws.String("hash-1"),
		Body:                   aws.String("foo"),
	})
	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           aws.Int64(int64(1)),
		MessageGroupId:         aws.String("group-2"),
		MessageDeduplicationId: aws.String("hash-2"),
		Body:                   aws.String("bar"),
	})
	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           aws.Int64(int64(1)),
		MessageGroupId:         aws.String("group-3"),
		MessageDeduplicationId: aws.String("hash-3"),
		Body:                   aws.String("baz"),
	})
	awsErr := awserr.New(sqs.ErrCodeBatchRequestTooLong, "foo", nil)
	qs.executor.
		On("Execute", context.Background(), mock.AnythingOfType("aws.RequestFunction")).
		Once().
		Return(nil, awsErr)
	qs.executor.
		On("Execute", context.Background(), mock.AnythingOfType("aws.RequestFunction")).
		Twice().
		Return(nil, nil)
	err := qs.queue.SendBatch(context.Background(), msgs)
	qs.Nil(err)
}
