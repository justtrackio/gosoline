//+build integration

package test_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_sns_sqs(t *testing.T) {
	setup(t)

	mocks := pkgTest.Boot("test_configs/config.sns_sqs.test.yml")
	defer mocks.Shutdown()

	queueName := "my-queue"
	topicName := "my-topic"

	topicArn := fmt.Sprintf("arn:aws:sns:us-east-1:000000000000:%s", topicName)
	queueUrl := fmt.Sprintf("http://localhost:4576/queue/%s", queueName)

	snsClient := mocks.ProvideClient("sns_sqs", "sns").(*sns.SNS)
	sqsClient := mocks.ProvideClient("sns_sqs", "sqs").(*sqs.SQS)

	logger := mon.NewLogger()
	res := &cloud.BackoffResource{
		Type: "sns",
		Name: topicName,
	}

	var executor = cloud.NewBackoffExecutor(logger, res, &cloud.BackoffSettings{
		Enabled:             true,
		Blocking:            true,
		CancelDelay:         time.Second * 1,
		InitialInterval:     time.Millisecond * 50,
		RandomizationFactor: 0.5,
		Multiplier:          1.5,
		MaxInterval:         time.Second * 2,
		MaxElapsedTime:      time.Second * 10,
	})

	// create a topic
	_, err := executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		return snsClient.CreateTopicRequest(&sns.CreateTopicInput{
			Name: aws.String(topicName),
		})
	})

	assert.NoError(t, err)

	// create a queue
	_, err = executor.Execute(context.Background(), func() (request *request.Request, i interface{}) {
		return sqsClient.CreateQueueRequest(&sqs.CreateQueueInput{
			QueueName: aws.String(queueName),
		})
	})

	assert.NoError(t, err)

	// create a topic subscription
	_, err = executor.Execute(context.Background(), func() (request *request.Request, i interface{}) {
		return snsClient.SubscribeRequest(&sns.SubscribeInput{
			Protocol: aws.String("sqs"),
			Endpoint: aws.String(queueUrl),
			TopicArn: aws.String(topicArn),
		})
	})

	assert.NoError(t, err)

	// send a message to a topic
	_, err = executor.Execute(context.Background(), func() (r *request.Request, i interface{}) {
		return snsClient.PublishRequest(&sns.PublishInput{
			Message:  aws.String("Hello there."),
			TopicArn: aws.String(topicArn),
		})
	})
	assert.NoError(t, err)

	// receive the message from sqs
	receive, err := executor.Execute(context.Background(), func() (r *request.Request, i interface{}) {
		return sqsClient.ReceiveMessageRequest(&sqs.ReceiveMessageInput{
			QueueUrl: aws.String(queueUrl),
		})
	})

	assert.NoError(t, err)
	if !assert.NotNil(t, receive) {
		return
	}

	receiveOutput := receive.(*sqs.ReceiveMessageOutput)

	if !assert.NotNil(t, receiveOutput) {
		return
	}
	if assert.Len(t, receiveOutput.Messages, 1) {
		assert.Contains(t, *receiveOutput.Messages[0].Body, "Hello there.")
	}
}
