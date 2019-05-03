package sqs_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/aws/aws-sdk-go/aws"
	awsSqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQueueExists(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	inputList := &awsSqs.ListQueuesInput{
		QueueNamePrefix: aws.String("project-env-family-app-my-test-queue"),
	}
	outputList := &awsSqs.ListQueuesOutput{
		QueueUrls: []*string{},
	}
	client.On("ListQueues", inputList).Return(outputList, nil)

	sqs.QueueExists(logger, client, sqs.Settings{
		AppId: cfg.AppId{
			Project:     "project",
			Environment: "env",
			Family:      "family",
			Application: "app",
		},
		QueueId:    "my-test-queue",
		AutoCreate: true,
	})

	client.AssertExpectations(t)
}

func TestGetUrl(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	input := &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("project-env-family-app-my-test-queue"),
	}
	output := &awsSqs.GetQueueUrlOutput{
		QueueUrl: aws.String("my-returned-queue-url"),
	}
	client.On("GetQueueUrl", input).Return(output, nil)

	url := sqs.GetUrl(logger, client, sqs.Settings{
		AppId: cfg.AppId{
			Project:     "project",
			Environment: "env",
			Family:      "family",
			Application: "app",
		},
		QueueId:    "my-test-queue",
		AutoCreate: true,
	})

	assert.Equal(t, "my-returned-queue-url", url, "the urls should match")
	client.AssertExpectations(t)
}

func TestGetArn(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	input := &awsSqs.GetQueueAttributesInput{
		AttributeNames: []*string{aws.String("QueueArn")},
		QueueUrl:       aws.String("my-returned-queue-url"),
	}
	output := &awsSqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{"QueueArn": aws.String("my-returned-queue-arn")},
	}
	client.On("GetQueueAttributes", input).Return(output, nil)

	arn := sqs.GetArn(logger, client, sqs.Settings{
		AppId: cfg.AppId{
			Project:     "project",
			Environment: "env",
			Family:      "family",
			Application: "app",
		},
		QueueId:    "my-test-queue",
		AutoCreate: true,
		Url:        "my-returned-queue-url",
	})

	assert.Equal(t, "my-returned-queue-arn", arn, "the arns should match")
	client.AssertExpectations(t)
}
