//+build integration

package test_test

import (
	"fmt"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_sns_sqs(t *testing.T) {
	setup(t)

	pkgTest.Boot("test_configs/config.sns_sqs.test.yml")
	defer pkgTest.Shutdown()

	queueName := "my-queue"
	topicName := "my-topic"

	snsClient := pkgTest.ProvideSnsClient("sns_sqs")
	topicsOutput, err := snsClient.ListTopics(&sns.ListTopicsInput{})

	assert.NoError(t, err)
	assert.NotNil(t, topicsOutput)
	assert.Len(t, topicsOutput.Topics, 0)

	sqsClient := pkgTest.ProvideSqsClient("sns_sqs")

	// create a queue
	createQueueOutput, err := sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})

	assert.NoError(t, err)
	assert.NotNil(t, createQueueOutput)
	assert.NotNil(t, createQueueOutput.QueueUrl)
	assert.Equal(t, *createQueueOutput.QueueUrl, fmt.Sprintf("http://localhost:4576/queue/%s", queueName))

	// create a topic
	createTopicOutput, err := snsClient.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String(topicName),
	})

	assert.NoError(t, err)
	assert.NotNil(t, createTopicOutput)
	assert.NotNil(t, createTopicOutput.TopicArn)
	assert.Equal(t, *createTopicOutput.TopicArn, fmt.Sprintf("arn:aws:sns:us-east-1:000000000000:%s", topicName))

	// create a topic subscription
	subscriptionOutput, err := snsClient.Subscribe(&sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		Endpoint: aws.String(fmt.Sprintf("http://localhost:4576/queue/%s", queueName)),
		TopicArn: aws.String(fmt.Sprintf("arn:aws:sns:us-east-1:000000000000:%s", topicName)),
	})

	assert.NoError(t, err)
	assert.NotNil(t, subscriptionOutput)
	assert.NotNil(t, subscriptionOutput.SubscriptionArn)
	assert.Contains(t, *subscriptionOutput.SubscriptionArn, fmt.Sprintf("arn:aws:sns:us-east-1:000000000000:%s:", topicName))

	// send a message to a topic
	publishOutput, err := snsClient.Publish(&sns.PublishInput{
		Message:  aws.String("Hello there."),
		TopicArn: aws.String(fmt.Sprintf("arn:aws:sns:us-east-1:000000000000:%s", topicName)),
	})

	assert.NoError(t, err)
	assert.NotNil(t, publishOutput)
	assert.NotNil(t, publishOutput.MessageId)

	// wait for localstack to forward the message to sqs (race condition)
	time.Sleep(1 * time.Second)

	// receive the message from sqs
	receiveOutput, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: aws.String(fmt.Sprintf("http://localhost:4576/queue/%s", queueName)),
	})

	assert.NoError(t, err)
	assert.NotNil(t, receiveOutput)
	assert.Len(t, receiveOutput.Messages, 1)
	assert.Contains(t, *receiveOutput.Messages[0].Body, "Hello there.")
}
