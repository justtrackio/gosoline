package sqs_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/sqs/mocks"
	"github.com/aws/aws-sdk-go/aws"
	awsSqs "github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pkg/errors"
	"testing"
)

func TestCreateQueue(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	client := new(mocks.SQSAPI)

	inputList := &awsSqs.GetQueueUrlInput{
		QueueName: aws.String("project-env-family-app-my-test-queue"),
	}
	client.On("GetQueueUrl", inputList).Return(nil, errors.New("blah"))

	inputCreate := &awsSqs.CreateQueueInput{
		QueueName: aws.String("project-env-family-app-my-test-queue"),
	}
	client.On("CreateQueue", inputCreate).Return(nil, nil)

	sqs.CreateQueue(logger, client, sqs.Settings{
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
