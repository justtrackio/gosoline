package sqs_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsSqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestService_CreateQueue(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	client.On("GetQueueUrl", &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue"),
	}).Return(nil, awserr.New(awsSqs.ErrCodeQueueDoesNotExist, "", nil)).Once()

	client.On("CreateQueue", &awsSqs.CreateQueueInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue"),
		Attributes: map[string]*string{
			"RedrivePolicy": aws.String("{\"deadLetterTargetArn\":\"applike-test-gosoline-sqs-my-queue-dead.arn\",\"maxReceiveCount\":\"3\"}"),
		},
	}).Return(nil, nil)

	client.On("GetQueueUrl", &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue"),
	}).Return(&awsSqs.GetQueueUrlOutput{
		QueueUrl: aws.String("applike-test-gosoline-sqs-my-queue.url"),
	}, nil)

	client.On("GetQueueAttributes", &awsSqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String("QueueArn")},
		QueueUrl:       aws.String("applike-test-gosoline-sqs-my-queue.url"),
	}).Return(&awsSqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{
			"QueueArn": aws.String("applike-test-gosoline-sqs-my-queue.arn"),
		},
	}, nil)

	client.On("SetQueueAttributes", &awsSqs.SetQueueAttributesInput{
		QueueUrl: aws.String("applike-test-gosoline-sqs-my-queue.url"),
		Attributes: map[string]*string{
			"VisibilityTimeout": aws.String("30"),
		},
	}).Return(nil, nil)

	// dead letter queue
	client.On("CreateQueue", &awsSqs.CreateQueueInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue-dead"),
	}).Return(nil, nil)

	client.On("GetQueueUrl", &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("applike-test-gosoline-sqs-my-queue-dead"),
	}).Return(&awsSqs.GetQueueUrlOutput{
		QueueUrl: aws.String("applike-test-gosoline-sqs-my-queue-dead.url"),
	}, nil)

	client.On("GetQueueAttributes", &awsSqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String("QueueArn")},
		QueueUrl:       aws.String("applike-test-gosoline-sqs-my-queue-dead.url"),
	}).Return(&awsSqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{
			"QueueArn": aws.String("applike-test-gosoline-sqs-my-queue-dead.arn"),
		},
	}, nil)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	props, err := srv.CreateQueue(&sqs.Settings{
		AppId: cfg.AppId{
			Project:     "applike",
			Environment: "test",
			Family:      "gosoline",
			Application: "sqs",
		},
		QueueId: "my-queue",
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
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	client.On("GetQueueUrl", mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(
		&awsSqs.GetQueueUrlOutput{
			QueueUrl: aws.String("https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id"),
		},
		nil,
	)

	client.On("GetQueueAttributes", mock.AnythingOfType("*sqs.GetQueueAttributesInput")).Return(
		&awsSqs.GetQueueAttributesOutput{
			Attributes: map[string]*string{
				"QueueArn": aws.String("arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id"),
			},
		},
		nil,
	)

	expected := &sqs.Properties{
		Name: "applike-test-gosoline-queue-id",
		Url:  "https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id",
		Arn:  "arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id",
	}

	props, err := srv.GetPropertiesByName("applike-test-gosoline-queue-id")

	assert.NoError(t, err)
	assert.EqualValues(t, expected, props)

	client.AssertExpectations(t)
}

func TestService_GetPropertiesByArn(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	client.On("GetQueueUrl", mock.AnythingOfType("*sqs.GetQueueUrlInput")).Return(
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

	props, err := srv.GetPropertiesByArn("arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id")

	assert.NoError(t, err)
	assert.EqualValues(t, expected, props)

	client.AssertExpectations(t)
}

func TestService_Purge(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	url := "https://sqs.eu-central-1.amazonaws.com/accountId/applike-test-gosoline-queue-id"

	client.On("PurgeQueue", mock.AnythingOfType("*sqs.PurgeQueueInput")).Return(&awsSqs.PurgeQueueOutput{}, nil)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	err := srv.Purge(url)

	assert.NoError(t, err)

	client.AssertExpectations(t)
}
