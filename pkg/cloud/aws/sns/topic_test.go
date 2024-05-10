package sns_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsSns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	gosoSns "github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	gosoSnsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sns/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
)

func TestTopicTestSuite(t *testing.T) {
	suite.Run(t, new(TopicTestSuite))
}

type TopicTestSuite struct {
	suite.Suite
	ctx    context.Context
	client *gosoSnsMocks.Client
	topic  gosoSns.Topic
}

func (s *TopicTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.ctx = s.T().Context()
	s.client = gosoSnsMocks.NewClient(s.T())
	s.topic = gosoSns.NewTopicWithInterfaces(logger, s.client, "topicArn")
}

func (s *TopicTestSuite) TestPublish() {
	input := &awsSns.PublishInput{
		TopicArn:          aws.String("topicArn"),
		Message:           aws.String("test"),
		MessageAttributes: map[string]types.MessageAttributeValue{},
	}

	s.client.EXPECT().Publish(matcher.Context, input).Return(nil, nil).Once()

	err := s.topic.Publish(s.ctx, "test", map[string]string{})
	s.NoError(err)
}

func (s *TopicTestSuite) TestPublishError() {
	input := &awsSns.PublishInput{
		TopicArn: aws.String("topicArn"),
		Message:  aws.String("test"),
	}

	s.client.EXPECT().Publish(matcher.Context, input).Return(nil, fmt.Errorf("error")).Once()

	err := s.topic.Publish(s.T().Context(), "test")
	s.Error(err)
}

func (s *TopicTestSuite) TestPublishBatch() {
	messages := []string{
		"1",
		"2",
		"3",
		"4",
		"5",
		"6",
		"7",
		"8",
		"9",
		"10",
		"11",
	}
	attributes := make([]map[string]string, len(messages))
	entries := make([]types.PublishBatchRequestEntry, 10)
	for i := 0; i < 10; i++ {
		entries[i] = types.PublishBatchRequestEntry{
			Id:                mdl.Box(fmt.Sprintf("%d", i)),
			Message:           mdl.Box(fmt.Sprintf("%d", i+1)),
			MessageAttributes: make(map[string]types.MessageAttributeValue),
		}
	}
	firstBatch := &awsSns.PublishBatchInput{
		TopicArn:                   aws.String("topicArn"),
		PublishBatchRequestEntries: entries,
	}

	s.client.EXPECT().PublishBatch(matcher.Context, firstBatch).Return(nil, nil).Once().Once()

	secondBatch := &awsSns.PublishBatchInput{
		TopicArn: aws.String("topicArn"),
		PublishBatchRequestEntries: []types.PublishBatchRequestEntry{
			{
				Id:                mdl.Box("10"),
				Message:           mdl.Box("11"),
				MessageAttributes: make(map[string]types.MessageAttributeValue),
			},
		},
	}

	s.client.EXPECT().PublishBatch(matcher.Context, secondBatch).Return(nil, nil).Once().Once()

	err := s.topic.PublishBatch(s.ctx, messages, attributes)
	s.NoError(err)
}
