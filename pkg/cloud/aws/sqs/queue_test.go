package sqs_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsSqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	gosoSqs "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	sqsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestRunQueueTestSuite(t *testing.T) {
	suite.Run(t, new(queueTestSuite))
}

type queueTestSuite struct {
	suite.Suite
	ctx    context.Context
	client *sqsMocks.Client
	queue  gosoSqs.Queue
}

func (s *queueTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.ctx = s.T().Context()
	s.client = new(sqsMocks.Client)
	s.queue = gosoSqs.NewQueueWithInterfaces(logger, s.client, &gosoSqs.Properties{
		Url: "http://foo.bar.baz",
	})
}

func (s *queueTestSuite) TestSendBatch_EmptyBatch() {
	msgs := make([]*gosoSqs.Message, 0)

	err := s.queue.SendBatch(s.ctx, msgs)
	s.Nil(err)
}

func (s *queueTestSuite) TestSendBatch_OneMsgRequestTooLongError() {
	msgs := make([]*gosoSqs.Message, 0)

	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           int32(1),
		MessageGroupId:         aws.String("group-1"),
		MessageDeduplicationId: aws.String("hash-1"),
		Body:                   aws.String("foo"),
	})

	errRequestTooLong := &types.BatchRequestTooLong{}
	s.client.EXPECT().SendMessageBatch(matcher.Context, mock.AnythingOfType("*sqs.SendMessageBatchInput")).Once().Return(nil, errRequestTooLong)

	err := s.queue.SendBatch(s.ctx, msgs)
	s.Equal(errRequestTooLong, err)
}

func (s *queueTestSuite) TestSendBatch_ThreeMsgRequestTooLongError() {
	msgs := make([]*gosoSqs.Message, 0)

	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           int32(1),
		MessageGroupId:         aws.String("group-1"),
		MessageDeduplicationId: aws.String("hash-1"),
		Body:                   aws.String("foo"),
	})
	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           int32(1),
		MessageGroupId:         aws.String("group-2"),
		MessageDeduplicationId: aws.String("hash-2"),
		Body:                   aws.String("bar"),
	})
	msgs = append(msgs, &gosoSqs.Message{
		DelaySeconds:           int32(1),
		MessageGroupId:         aws.String("group-3"),
		MessageDeduplicationId: aws.String("hash-3"),
		Body:                   aws.String("baz"),
	})

	errRequestTooLong := &types.BatchRequestTooLong{}
	s.client.EXPECT().SendMessageBatch(matcher.Context, mock.AnythingOfType("*sqs.SendMessageBatchInput")).Once().Return(nil, errRequestTooLong)
	s.client.EXPECT().SendMessageBatch(matcher.Context, mock.AnythingOfType("*sqs.SendMessageBatchInput")).Twice().Return(nil, nil)

	err := s.queue.SendBatch(s.ctx, msgs)
	s.Nil(err)
}

func (s *queueTestSuite) TestDeleteMessageBatch_ReturnsEntryFailures() {
	s.client.EXPECT().DeleteMessageBatch(matcher.Context, mock.AnythingOfType("*sqs.DeleteMessageBatchInput")).Return(&awsSqs.DeleteMessageBatchOutput{
		Failed: []types.BatchResultErrorEntry{{
			Id:      aws.String("entry-id"),
			Message: aws.String("receipt handle is invalid"),
		}},
	}, nil).Once()

	err := s.queue.DeleteMessageBatch(s.ctx, []string{"receipt-handle"})

	s.EqualError(err, "1 error occurred:\n\t* failed to delete message batch entry entry-id: receipt handle is invalid\n\n")
}
