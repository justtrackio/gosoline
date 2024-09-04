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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
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

	s.ctx = context.Background()
	s.client = gosoSnsMocks.NewClient(s.T())
	s.topic = gosoSns.NewTopicWithInterfaces(logger, s.client, "topicArn")
}

func (s *TopicTestSuite) TestPublish() {
	input := &awsSns.PublishInput{
		TopicArn:          aws.String("topicArn"),
		Message:           aws.String("test"),
		MessageAttributes: map[string]types.MessageAttributeValue{},
	}

	s.client.EXPECT().Publish(mock.AnythingOfType("*context.valueCtx"), input).Return(nil, nil).Once()

	err := s.topic.Publish(s.ctx, "test", map[string]string{})
	s.NoError(err)
}

func (s *TopicTestSuite) TestPublishError() {
	input := &awsSns.PublishInput{
		TopicArn: aws.String("topicArn"),
		Message:  aws.String("test"),
	}

	s.client.EXPECT().Publish(mock.AnythingOfType("*context.valueCtx"), input).Return(nil, fmt.Errorf("error")).Once()

	err := s.topic.Publish(context.Background(), "test")
	s.Error(err)
}

func (s *TopicTestSuite) TestSubscribeSqs() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{}
	s.client.EXPECT().ListSubscriptionsByTopic(mock.AnythingOfType("*context.valueCtx"), listInput).Return(listOutput, nil).Once()

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]string{
			"FilterPolicy": `{"model":"goso","version":"1"}`,
		},
		TopicArn: aws.String("topicArn"),
		Protocol: aws.String("sqs"),
		Endpoint: aws.String("queueArn"),
	}
	s.client.EXPECT().Subscribe(mock.AnythingOfType("*context.valueCtx"), subInput).Return(nil, nil).Once()

	err := s.topic.SubscribeSqs(s.ctx, "queueArn", map[string]string{
		"model":   "goso",
		"version": "1",
	})
	s.NoError(err)
}

func (s *TopicTestSuite) TestSubscribeSqsExists() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{
		Subscriptions: []types.Subscription{
			{
				TopicArn:        aws.String("topicArn"),
				SubscriptionArn: aws.String("subscriptionArn"),
				Endpoint:        aws.String("queueArn"),
			},
		},
	}
	s.client.EXPECT().ListSubscriptionsByTopic(mock.AnythingOfType("*context.valueCtx"), listInput).Return(listOutput, nil).Once()

	getAttributesInput := &awsSns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String("subscriptionArn")}
	getAttributesOutput := &awsSns.GetSubscriptionAttributesOutput{
		Attributes: map[string]string{
			"FilterPolicy": `{"model":"goso","version":"1"}`,
		},
	}
	s.client.EXPECT().GetSubscriptionAttributes(mock.AnythingOfType("*context.valueCtx"), getAttributesInput).Return(getAttributesOutput, nil).Once()

	err := s.topic.SubscribeSqs(context.Background(), "queueArn", map[string]string{
		"model":   "goso",
		"version": "1",
	})
	s.NoError(err)
}

func (s *TopicTestSuite) TestSubscribeSqsExistsWithDifferentAttributes() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{
		Subscriptions: []types.Subscription{
			{
				TopicArn:        aws.String("topicArn"),
				SubscriptionArn: aws.String("subscriptionArn"),
				Endpoint:        aws.String("queueArn"),
			},
		},
	}
	s.client.EXPECT().ListSubscriptionsByTopic(mock.AnythingOfType("*context.valueCtx"), listInput).Return(listOutput, nil).Once()

	getAttributesInput := &awsSns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String("subscriptionArn")}
	getAttributesOutput := &awsSns.GetSubscriptionAttributesOutput{
		Attributes: map[string]string{
			"FilterPolicy": `{"model":"mismatch"}`,
		},
	}
	s.client.EXPECT().GetSubscriptionAttributes(mock.AnythingOfType("*context.valueCtx"), getAttributesInput).Return(getAttributesOutput, nil).Once()

	unsubscribeInput := &awsSns.UnsubscribeInput{SubscriptionArn: aws.String("subscriptionArn")}
	unsubscribeOutput := &awsSns.UnsubscribeOutput{}
	s.client.EXPECT().Unsubscribe(mock.AnythingOfType("*context.valueCtx"), unsubscribeInput).Return(unsubscribeOutput, nil).Once()

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]string{
			"FilterPolicy": `{"model":"goso"}`,
		},
		Endpoint: aws.String("queueArn"),
		Protocol: aws.String("sqs"),
		TopicArn: aws.String("topicArn"),
	}
	s.client.EXPECT().Subscribe(mock.AnythingOfType("*context.valueCtx"), subInput).Return(nil, nil).Once()

	err := s.topic.SubscribeSqs(context.Background(), "queueArn", map[string]string{
		// err := s.topic.SubscribeSqs(s.ctx, "queueArn", map[string]interface{}{
		"model": "goso",
	})
	s.NoError(err)
}

func (s *TopicTestSuite) TestSubscribeSqsError() {
	subErr := errors.New("subscribe error")

	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{}
	s.client.EXPECT().ListSubscriptionsByTopic(mock.AnythingOfType("*context.valueCtx"), listInput).Return(listOutput, nil).Once()

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]string{},
		TopicArn:   aws.String("topicArn"),
		Protocol:   aws.String("sqs"),
		Endpoint:   aws.String("queueArn"),
	}
	s.client.EXPECT().Subscribe(mock.AnythingOfType("*context.valueCtx"), subInput).Return(nil, subErr).Once()

	err := s.topic.SubscribeSqs(s.ctx, "queueArn", map[string]string{})
	s.EqualError(err, "could not subscribe to topic arn topicArn for sqs queue arn queueArn: subscribe error")
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

	s.client.EXPECT().PublishBatch(mock.AnythingOfType("*context.valueCtx"), firstBatch).Return(nil, nil).Once().Once()

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

	s.client.EXPECT().PublishBatch(mock.AnythingOfType("*context.valueCtx"), secondBatch).Return(nil, nil).Once().Once()

	err := s.topic.PublishBatch(s.ctx, messages, attributes)
	s.NoError(err)
}
