package sns_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/sns"
	snsMocks "github.com/applike/gosoline/pkg/sns/mocks"
	"github.com/aws/aws-sdk-go/aws"
	awsSns "github.com/aws/aws-sdk-go/service/sns"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"testing"
)

type TopicTestSuite struct {
	suite.Suite
	client   *snsMocks.Client
	executor *gosoAws.TestableExecutor
	topic    sns.Topic
}

func (s *TopicTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMockedAll()

	settings := &sns.Settings{
		Arn: "topicArn",
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	s.client = new(snsMocks.Client)
	s.executor = gosoAws.NewTestableExecutor(&s.client.Mock)
	s.topic = sns.NewTopicWithInterfaces(logger, s.client, s.executor, settings)
}

func (s *TopicTestSuite) TestPublish() {
	input := &awsSns.PublishInput{
		TopicArn:          aws.String("topicArn"),
		Message:           aws.String("test"),
		MessageAttributes: map[string]*awsSns.MessageAttributeValue{},
	}
	s.executor.ExpectExecution("PublishRequest", input, nil, nil)

	err := s.topic.Publish(context.Background(), aws.String("test"), map[string]interface{}{})
	s.NoError(err)

	s.executor.AssertExpectations(s.T())
}

func (s *TopicTestSuite) TestPublishError() {
	input := &awsSns.PublishInput{
		TopicArn: aws.String("topicArn"),
		Message:  aws.String("test"),
	}
	s.executor.ExpectExecution("PublishRequest", input, nil, errors.New("error"))

	err := s.topic.Publish(context.Background(), aws.String("test"))
	s.Error(err)

	s.executor.AssertExpectations(s.T())
}

func (s *TopicTestSuite) TestSubscribeSqs() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{}
	s.executor.ExpectExecution("ListSubscriptionsByTopicRequest", listInput, listOutput, nil)

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]*string{
			"FilterPolicy": aws.String(`{"model":"goso","version":1}`),
		},
		TopicArn: aws.String("topicArn"),
		Protocol: aws.String("sqs"),
		Endpoint: aws.String("queueArn"),
	}
	s.executor.ExpectExecution("SubscribeRequest", subInput, nil, nil)

	err := s.topic.SubscribeSqs("queueArn", map[string]interface{}{
		"model":   "goso",
		"version": 1,
	})
	s.NoError(err)

	s.executor.AssertExpectations(s.T())
}

func (s *TopicTestSuite) TestSubscribeSqsExists() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{
		Subscriptions: []*awsSns.Subscription{
			{
				TopicArn:        aws.String("topicArn"),
				SubscriptionArn: aws.String("subscriptionArn"),
				Endpoint:        aws.String("queueArn"),
			},
		},
	}
	s.executor.ExpectExecution("ListSubscriptionsByTopicRequest", listInput, listOutput, nil)

	getAttributesInput := &awsSns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String("subscriptionArn")}
	getAttributesOutput := &awsSns.GetSubscriptionAttributesOutput{
		Attributes: map[string]*string{
			"FilterPolicy": aws.String(`{"model":"goso","version":1}`),
		},
	}
	s.executor.ExpectExecution("GetSubscriptionAttributesRequest", getAttributesInput, getAttributesOutput, nil)

	err := s.topic.SubscribeSqs("queueArn", map[string]interface{}{
		"model":   "goso",
		"version": 1,
	})
	s.NoError(err)

	s.executor.AssertExpectations(s.T())
}

func (s *TopicTestSuite) TestSubscribeSqsExistsWithDifferentAttributes() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{
		Subscriptions: []*awsSns.Subscription{
			{
				TopicArn:        aws.String("topicArn"),
				SubscriptionArn: aws.String("subscriptionArn"),
				Endpoint:        aws.String("queueArn"),
			},
		},
	}
	s.executor.ExpectExecution("ListSubscriptionsByTopicRequest", listInput, listOutput, nil)

	getAttributesInput := &awsSns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String("subscriptionArn")}
	getAttributesOutput := &awsSns.GetSubscriptionAttributesOutput{
		Attributes: map[string]*string{
			"FilterPolicy": aws.String(`{"model":"mismatch"}`),
		},
	}
	s.executor.ExpectExecution("GetSubscriptionAttributesRequest", getAttributesInput, getAttributesOutput, nil)

	unsubscribeInput := &awsSns.UnsubscribeInput{SubscriptionArn: aws.String("subscriptionArn")}
	unsubscribeOutput := &awsSns.UnsubscribeOutput{}
	s.executor.ExpectExecution("UnsubscribeRequest", unsubscribeInput, unsubscribeOutput, nil)

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]*string{
			"FilterPolicy": aws.String(`{"model":"goso"}`),
		},
		Endpoint: aws.String("queueArn"),
		Protocol: aws.String("sqs"),
		TopicArn: aws.String("topicArn"),
	}
	s.executor.ExpectExecution("SubscribeRequest", subInput, nil, nil)

	err := s.topic.SubscribeSqs("queueArn", map[string]interface{}{
		"model": "goso",
	})
	s.NoError(err)

	s.executor.AssertExpectations(s.T())
}

func (s *TopicTestSuite) TestSubscribeSqsError() {
	subErr := errors.New("subscribe error")

	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{}
	s.executor.ExpectExecution("ListSubscriptionsByTopicRequest", listInput, listOutput, nil)

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]*string{},
		TopicArn:   aws.String("topicArn"),
		Protocol:   aws.String("sqs"),
		Endpoint:   aws.String("queueArn"),
	}
	s.executor.ExpectExecution("SubscribeRequest", subInput, nil, subErr)

	err := s.topic.SubscribeSqs("queueArn", map[string]interface{}{})
	s.EqualError(err, "subscribe error")

	s.executor.AssertExpectations(s.T())
	s.client.AssertExpectations(s.T())
}

func TestTopicTestSuite(t *testing.T) {
	suite.Run(t, new(TopicTestSuite))
}
