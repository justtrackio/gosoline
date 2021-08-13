package sns_test

import (
	"context"
	"testing"

	gosoSns "github.com/applike/gosoline/pkg/cloud/aws/sns"
	gosoSnsMocks "github.com/applike/gosoline/pkg/cloud/aws/sns/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsSns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCreateTopic(t *testing.T) {
	ctx := context.Background()
	client := new(gosoSnsMocks.Client)
	client.On("CreateTopic", ctx, &awsSns.CreateTopicInput{
		Name: aws.String("mcoins-test-analytics-topicker-topic"),
	}).Return(&awsSns.CreateTopicOutput{
		TopicArn: aws.String("arn"),
	}, nil)

	logger := logMocks.NewLoggerMockedAll()

	arn, err := gosoSns.CreateTopic(ctx, logger, client, "mcoins-test-analytics-topicker-topic")

	assert.Equal(t, "arn", arn)
	assert.NoError(t, err)

	client.AssertExpectations(t)
}

func TestCreateTopicFailing(t *testing.T) {
	ctx := context.Background()
	client := new(gosoSnsMocks.Client)
	client.On("CreateTopic", ctx, &awsSns.CreateTopicInput{
		Name: aws.String("mcoins-test-analytics-topicker-topic"),
	}).Return(nil, errors.New(""))

	logger := logMocks.NewLoggerMockedAll()

	arn, err := gosoSns.CreateTopic(ctx, logger, client, "mcoins-test-analytics-topicker-topic")

	assert.Equal(t, "", arn)
	assert.Error(t, err)

	client.AssertExpectations(t)
}
