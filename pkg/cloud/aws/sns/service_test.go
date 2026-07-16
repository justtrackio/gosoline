package sns_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsSns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	gosoSns "github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	gosoSnsMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/sns/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type ServiceTestSuite struct {
	suite.Suite

	ctx     context.Context
	client  *gosoSnsMocks.Client
	service *gosoSns.Service
}

func (s *ServiceTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.ctx = s.T().Context()
	s.client = gosoSnsMocks.NewClient(s.T())
	s.service = gosoSns.NewServiceWithInterfaces(logger, s.client)
}

func (s *ServiceTestSuite) TestSubscribeSqs() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{}
	s.client.EXPECT().ListSubscriptionsByTopic(matcher.Context, listInput).Return(listOutput, nil).Once()

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]string{
			"FilterPolicy": `{"model":["goso"],"version":["1"]}`,
		},
		TopicArn: aws.String("topicArn"),
		Protocol: aws.String("sqs"),
		Endpoint: aws.String("queueArn"),
	}
	s.client.EXPECT().Subscribe(matcher.Context, subInput).Return(nil, nil).Once()

	err := s.service.SubscribeSqs(s.ctx, "queueArn", "topicArn", map[string][]string{
		"model":   {"goso"},
		"version": {"1"},
	})
	s.NoError(err)
}

func (s *ServiceTestSuite) TestSubscribeSqsExists() {
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
	s.client.EXPECT().ListSubscriptionsByTopic(matcher.Context, listInput).Return(listOutput, nil).Once()

	getAttributesInput := &awsSns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String("subscriptionArn")}
	getAttributesOutput := &awsSns.GetSubscriptionAttributesOutput{
		Attributes: map[string]string{
			"FilterPolicy": `{"model":["goso"],"version":["1"]}`,
		},
	}
	s.client.EXPECT().GetSubscriptionAttributes(matcher.Context, getAttributesInput).Return(getAttributesOutput, nil).Once()

	err := s.service.SubscribeSqs(s.T().Context(), "queueArn", "topicArn", map[string][]string{
		"model":   {"goso"},
		"version": {"1"},
	})
	s.NoError(err)
}

func (s *ServiceTestSuite) TestSubscribeSqsExistsWithDifferentAttributes() {
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
	s.client.EXPECT().ListSubscriptionsByTopic(matcher.Context, listInput).Return(listOutput, nil).Once()

	getAttributesInput := &awsSns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String("subscriptionArn")}
	getAttributesOutput := &awsSns.GetSubscriptionAttributesOutput{
		Attributes: map[string]string{
			"FilterPolicy": `{"model":"mismatch"}`,
		},
	}
	s.client.EXPECT().GetSubscriptionAttributes(matcher.Context, getAttributesInput).Return(getAttributesOutput, nil).Once()

	setAttributesInput := &awsSns.SetSubscriptionAttributesInput{
		AttributeName:   aws.String("FilterPolicy"),
		AttributeValue:  aws.String(`{"model":["goso"]}`),
		SubscriptionArn: aws.String("subscriptionArn"),
	}
	s.client.EXPECT().SetSubscriptionAttributes(matcher.Context, setAttributesInput).Return(&awsSns.SetSubscriptionAttributesOutput{}, nil).Once()

	err := s.service.SubscribeSqs(s.T().Context(), "queueArn", "topicArn", map[string][]string{
		"model": {"goso"},
	})
	s.NoError(err)
}

func (s *ServiceTestSuite) TestSubscribeSqsError() {
	subErr := errors.New("subscribe error")

	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{}
	s.client.EXPECT().ListSubscriptionsByTopic(matcher.Context, listInput).Return(listOutput, nil).Once()

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]string{},
		TopicArn:   aws.String("topicArn"),
		Protocol:   aws.String("sqs"),
		Endpoint:   aws.String("queueArn"),
	}
	s.client.EXPECT().Subscribe(matcher.Context, subInput).Return(nil, subErr).Once()

	err := s.service.SubscribeSqs(s.ctx, "queueArn", "topicArn", map[string][]string{})
	s.EqualError(err, "could not subscribe to topic arn topicArn for sqs queue arn queueArn: subscribe error")
}

func (s *ServiceTestSuite) TestSubscribeSqsWithArrayFilterPolicy() {
	listInput := &awsSns.ListSubscriptionsByTopicInput{TopicArn: aws.String("topicArn")}
	listOutput := &awsSns.ListSubscriptionsByTopicOutput{}
	s.client.EXPECT().ListSubscriptionsByTopic(matcher.Context, listInput).Return(listOutput, nil).Once()

	subInput := &awsSns.SubscribeInput{
		Attributes: map[string]string{
			"FilterPolicy": `{"modelId":["model.a","model.b","model.c"]}`,
		},
		TopicArn: aws.String("topicArn"),
		Protocol: aws.String("sqs"),
		Endpoint: aws.String("queueArn"),
	}
	s.client.EXPECT().Subscribe(matcher.Context, subInput).Return(nil, nil).Once()

	err := s.service.SubscribeSqs(s.ctx, "queueArn", "topicArn", map[string][]string{
		"modelId": {"model.a", "model.b", "model.c"},
	})
	s.NoError(err)
}

func (s *ServiceTestSuite) TestSubscribeSqsExistsWithArrayFilterPolicy() {
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
	s.client.EXPECT().ListSubscriptionsByTopic(matcher.Context, listInput).Return(listOutput, nil).Once()

	getAttributesInput := &awsSns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String("subscriptionArn")}
	getAttributesOutput := &awsSns.GetSubscriptionAttributesOutput{
		Attributes: map[string]string{
			"FilterPolicy": `{"modelId":["model.a","model.b"]}`,
		},
	}
	s.client.EXPECT().GetSubscriptionAttributes(matcher.Context, getAttributesInput).Return(getAttributesOutput, nil).Once()

	setAttributesInput := &awsSns.SetSubscriptionAttributesInput{
		AttributeName:   aws.String("FilterPolicy"),
		AttributeValue:  aws.String(`{"modelId":["model.a","model.b","model.c"]}`),
		SubscriptionArn: aws.String("subscriptionArn"),
	}
	s.client.EXPECT().SetSubscriptionAttributes(matcher.Context, setAttributesInput).Return(&awsSns.SetSubscriptionAttributesOutput{}, nil).Once()

	err := s.service.SubscribeSqs(s.ctx, "queueArn", "topicArn", map[string][]string{
		"modelId": {"model.a", "model.b", "model.c"},
	})
	s.NoError(err)
}
