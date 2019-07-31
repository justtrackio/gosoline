package sqs_test

import (
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsSqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestService_CreateQueue(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	client.On("GetQueueUrl", &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("my-queue-url"),
	}).Return(nil, awserr.New(awsSqs.ErrCodeQueueDoesNotExist, "", nil)).Once()

	client.On("CreateQueue", &awsSqs.CreateQueueInput{
		QueueName: aws.String("my-queue-url"),
		Attributes: map[string]*string{
			"RedrivePolicy": aws.String("{\"deadLetterTargetArn\":\"my-queue-url-dead.arn\",\"maxReceiveCount\":\"3\"}"),
		},
	}).Return(nil, nil)

	client.On("GetQueueUrl", &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("my-queue-url"),
	}).Return(&awsSqs.GetQueueUrlOutput{
		QueueUrl: aws.String("my-queue-url.url"),
	}, nil)

	client.On("GetQueueAttributes", &awsSqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String("QueueArn")},
		QueueUrl:       aws.String("my-queue-url.url"),
	}).Return(&awsSqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{
			"QueueArn": aws.String("my-queue-url.arn"),
		},
	}, nil)

	client.On("SetQueueAttributes", &awsSqs.SetQueueAttributesInput{
		QueueUrl: aws.String("my-queue-url.url"),
		Attributes: map[string]*string{
			"VisibilityTimeout": aws.String("30"),
		},
	}).Return(nil, nil)

	// dead letter queue
	client.On("CreateQueue", &awsSqs.CreateQueueInput{
		QueueName: aws.String("my-queue-url-dead"),
	}).Return(nil, nil)

	client.On("GetQueueUrl", &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("my-queue-url-dead"),
	}).Return(&awsSqs.GetQueueUrlOutput{
		QueueUrl: aws.String("my-queue-url-dead.url"),
	}, nil)

	client.On("GetQueueAttributes", &awsSqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String("QueueArn")},
		QueueUrl:       aws.String("my-queue-url-dead.url"),
	}).Return(&awsSqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{
			"QueueArn": aws.String("my-queue-url-dead.arn"),
		},
	}, nil)

	srv := sqs.NewServiceWithInterfaces(logger, client, &sqs.ServiceSettings{
		AutoCreate: true,
	})

	props, err := srv.CreateQueue(&sqs.CreateQueueInput{
		Name: "my-queue-url",
		RedrivePolicy: sqs.RedrivePolicy{
			Enabled:         true,
			MaxReceiveCount: 3,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "my-queue-url.url", props.Url)
	assert.Equal(t, "my-queue-url.arn", props.Arn)
	client.AssertExpectations(t)
}
