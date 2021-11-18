package sqs_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsSqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_CreateQueue(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.Client)

	client.On("GetQueueUrl", ctx, &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue"),
	}).Return(nil, &types.QueueDoesNotExist{}).Once()

	client.On("CreateQueue", ctx, &awsSqs.CreateQueueInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue"),
		Attributes: map[string]string{
			"RedrivePolicy": "{\"deadLetterTargetArn\":\"applike-test-gosoline-sqs-my-queue-dead.arn\",\"maxReceiveCount\":\"3\"}",
		},
	}).Return(nil, nil)

	client.On("GetQueueUrl", ctx, &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue"),
	}).Return(&awsSqs.GetQueueUrlOutput{
		QueueUrl: aws.String("applike-test-gosoline-sqs-my-queue.url"),
	}, nil)

	client.On("GetQueueAttributes", ctx, &awsSqs.GetQueueAttributesInput{
		AttributeNames: []types.QueueAttributeName{"QueueArn"},
		QueueUrl:       aws.String("applike-test-gosoline-sqs-my-queue.url"),
	}).Return(&awsSqs.GetQueueAttributesOutput{
		Attributes: map[string]string{
			"QueueArn": "applike-test-gosoline-sqs-my-queue.arn",
		},
	}, nil)

	client.On("SetQueueAttributes", ctx, &awsSqs.SetQueueAttributesInput{
		QueueUrl: aws.String("applike-test-gosoline-sqs-my-queue.url"),
		Attributes: map[string]string{
			"VisibilityTimeout": "30",
		},
	}).Return(nil, nil)

	// dead letter queue
	client.On("CreateQueue", ctx, &awsSqs.CreateQueueInput{
		Attributes: map[string]string{},
		QueueName:  aws.String("applike-test-gosoline-sqs-my-queue-dead"),
	}).Return(nil, nil)

	client.On("GetQueueUrl", ctx, &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue-dead"),
	}).Return(&awsSqs.GetQueueUrlOutput{
		QueueUrl: aws.String("applike-test-gosoline-sqs-my-queue-dead.url"),
	}, nil)

	client.On("GetQueueAttributes", ctx, &awsSqs.GetQueueAttributesInput{
		AttributeNames: []types.QueueAttributeName{"QueueArn"},
		QueueUrl:       aws.String("applike-test-gosoline-sqs-my-queue-dead.url"),
	}).Return(&awsSqs.GetQueueAttributesOutput{
		Attributes: map[string]string{
			"QueueArn": "applike-test-gosoline-sqs-my-queue-dead.arn",
		},
	}, nil)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	props, err := srv.CreateQueue(ctx, &sqs.Settings{
		QueueName: "applike-test-gosoline-sqs-my-queue",
		RedrivePolicy: sqs.RedrivePolicy{
			Enabled:         true,
			MaxReceiveCount: 3,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "applike-test-gosoline-sqs-my-queue.url", props.Url)
	assert.Equal(t, "applike-test-gosoline-sqs-my-queue.arn", props.Arn)
	client.AssertExpectations(t)
}

func TestService_GetPropertiesByName(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.Client)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	client.On("GetQueueUrl", ctx, mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(
		&awsSqs.GetQueueUrlOutput{
			QueueUrl: aws.String("https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id"),
		},
		nil,
	)

	client.On("GetQueueAttributes", ctx, mock.AnythingOfType("*sqs.GetQueueAttributesInput")).Return(
		&awsSqs.GetQueueAttributesOutput{
			Attributes: map[string]string{
				"QueueArn": "arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id",
			},
		},
		nil,
	)

	expected := &sqs.Properties{
		Name: "applike-test-gosoline-queue-id",
		Url:  "https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id",
		Arn:  "arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id",
	}

	props, err := srv.GetPropertiesByName(ctx, "applike-test-gosoline-queue-id")

	assert.NoError(t, err)
	assert.EqualValues(t, expected, props)

	client.AssertExpectations(t)
}

func TestService_GetPropertiesByArn(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.Client)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	client.On("GetQueueUrl", ctx, mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(
		&awsSqs.GetQueueUrlOutput{
			QueueUrl: aws.String("https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id"),
		},
		nil,
	)

	expected := &sqs.Properties{
		Name: "applike-test-gosoline-queue-id",
		Url:  "https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id",
		Arn:  "arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id",
	}

	props, err := srv.GetPropertiesByArn(ctx, "arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id")

	assert.NoError(t, err)
	assert.EqualValues(t, expected, props)

	client.AssertExpectations(t)
}

func TestService_Purge(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.Client)

	url := "https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id"

	client.On("PurgeQueue", ctx, mock.AnythingOfType("*sqs.PurgeQueueInput")).Return(&awsSqs.PurgeQueueOutput{}, nil)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	err := srv.Purge(ctx, url)

	assert.NoError(t, err)

	client.AssertExpectations(t)
}
