package sns_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	gosoAwsMocks "github.com/applike/gosoline/pkg/cloud/aws/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/sns"
	snsMocks "github.com/applike/gosoline/pkg/sns/mocks"
	"github.com/aws/aws-sdk-go/aws"
	awsSns "github.com/aws/aws-sdk-go/service/sns"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestTopic_Publish(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	input := &awsSns.PublishInput{
		TopicArn: aws.String("arn"),
		Message:  aws.String("test"),
	}

	exec := gosoAws.NewTestableExecutor([]gosoAws.TestExecution{{
		Output: nil,
		Err:    nil,
	}})

	client := new(snsMocks.Client)
	client.On("PublishRequest", input).Return(nil, nil)

	s := &sns.Settings{
		Arn: "arn",
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	topic := sns.NewTopicWithInterfaces(logger, client, exec, s)
	err := topic.Publish(context.Background(), aws.String("test"))

	assert.NoError(t, err)

	client.AssertExpectations(t)
}

func TestTopic_PublishError(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	input := &awsSns.PublishInput{
		TopicArn: aws.String("arn"),
		Message:  aws.String("test"),
	}

	exec := gosoAws.NewTestableExecutor([]gosoAws.TestExecution{{
		Output: nil,
		Err:    errors.New("error"),
	}})

	client := new(snsMocks.Client)
	client.On("PublishRequest", input).Return(nil, nil)

	s := &sns.Settings{
		Arn: "arn",
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	topic := sns.NewTopicWithInterfaces(logger, client, exec, s)
	err := topic.Publish(context.Background(), aws.String("test"))

	assert.Error(t, err)

	client.AssertExpectations(t)
}

func TestTopic_SubscribeSqs(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	client := new(snsMocks.Client)

	s := &sns.Settings{
		Arn: "arn",
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	executor := new(gosoAwsMocks.Executor)
	executor.On("Execute", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("aws.RequestFunction")).Return(&awsSns.ListSubscriptionsByTopicOutput{}, nil).Twice()

	topic := sns.NewTopicWithInterfaces(logger, client, executor, s)
	err := topic.SubscribeSqs("arn")

	assert.NoError(t, err)

	executor.AssertExpectations(t)
}

func TestTopic_SubscribeSqsExists(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	client := new(snsMocks.Client)

	s := &sns.Settings{
		Arn: "arn",
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	executor := new(gosoAwsMocks.Executor)
	executor.On("Execute", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("aws.RequestFunction")).Return(&awsSns.ListSubscriptionsByTopicOutput{
		Subscriptions: []*awsSns.Subscription{
			{
				Endpoint: aws.String("arn"),
			},
		},
	}, nil).Once()

	topic := sns.NewTopicWithInterfaces(logger, client, executor, s)
	err := topic.SubscribeSqs("arn")

	assert.NoError(t, err)

	executor.AssertExpectations(t)
}

func TestTopic_SubscribeSqsError(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	client := new(snsMocks.Client)

	executor := new(gosoAwsMocks.Executor)
	executor.On("Execute", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("aws.RequestFunction")).Return(&awsSns.ListSubscriptionsByTopicOutput{}, nil).Once()
	executor.On("Execute", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("aws.RequestFunction")).Return(nil, errors.New("error")).Once()

	s := &sns.Settings{
		Arn: "arn",
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	topic := sns.NewTopicWithInterfaces(logger, client, executor, s)
	err := topic.SubscribeSqs("arn")

	assert.Error(t, err)

	executor.AssertExpectations(t)
}
