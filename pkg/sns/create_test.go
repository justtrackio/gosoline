package sns_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/sns"
	snsMocks "github.com/applike/gosoline/pkg/sns/mocks"
	"github.com/aws/aws-sdk-go/aws"
	awsSns "github.com/aws/aws-sdk-go/service/sns"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateTopic(t *testing.T) {
	s := &sns.Settings{
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	client := new(snsMocks.Client)
	client.On("CreateTopic", &awsSns.CreateTopicInput{
		Name: aws.String("mcoins-test-analytics-topicker-topic"),
	}).Return(&awsSns.CreateTopicOutput{
		TopicArn: aws.String("arn"),
	}, nil)

	logger := logMocks.NewLoggerMockedAll()

	arn, err := sns.CreateTopic(logger, client, s)

	assert.Equal(t, "arn", arn)
	assert.NoError(t, err)

	client.AssertExpectations(t)
}

func TestCreateTopicFailing(t *testing.T) {
	s := &sns.Settings{
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "topicker",
		},
		TopicId: "topic",
	}

	client := new(snsMocks.Client)
	client.On("CreateTopic", &awsSns.CreateTopicInput{
		Name: aws.String("mcoins-test-analytics-topicker-topic"),
	}).Return(nil, errors.New(""))

	logger := logMocks.NewLoggerMockedAll()

	arn, err := sns.CreateTopic(logger, client, s)

	assert.Equal(t, "", arn)
	assert.Error(t, err)

	client.AssertExpectations(t)
}
