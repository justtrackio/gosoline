//+build integration

package test_test

import (
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_sns(t *testing.T) {
	setup(t)

	test.Boot(mdl.String("test_configs/config.sns.test.yml"))

	snsClient := test.ProvideSnsClient("sns")
	o, err := snsClient.ListTopics(&sns.ListTopicsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.Topics, 0)

	test.Shutdown()
}

func Test_sns_sqs(t *testing.T) {
	setup(t)

	test.Boot(mdl.String("test_configs/config.sns_sqs.test.yml"))

	snsClient := test.ProvideSnsClient("sns")
	topicsOutput, err := snsClient.ListTopics(&sns.ListTopicsInput{})

	assert.NoError(t, err)
	assert.Len(t, topicsOutput.Topics, 0)

	sqsClient := test.ProvideSqsClient("sqs")
	queuesOutput, err := sqsClient.ListQueues(&sqs.ListQueuesInput{})

	assert.NoError(t, err)
	assert.Len(t, queuesOutput.QueueUrls, 0)

	// create a queue
	createQueueOutput, err := sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("my-queue"),
	})

	assert.NoError(t, err)
	assert.Equal(t, *createQueueOutput.QueueUrl, "http://localhost:9871/queue/my-queue")

	// create a topic
	createTopicOutput, err := snsClient.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String("my-topic"),
	})

	assert.NoError(t, err)
	assert.Equal(t, *createTopicOutput.TopicArn, "arn:aws:sns:eu-central-1:000000000000:my-topic")

	// create a topic subscription
	subscriptionOutput, err := snsClient.Subscribe(&sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		Endpoint: aws.String("http://localhost:9871/queue/my-queue"),
		TopicArn: aws.String("arn:aws:sns:eu-central-1:000000000000:my-topic"),
	})

	assert.NoError(t, err)
	assert.Contains(t, *subscriptionOutput.SubscriptionArn, "arn:aws:sns:eu-central-1:000000000000:my-topic:")

	// send a message to a topic
	_, err = snsClient.Publish(&sns.PublishInput{
		Message:  aws.String("Hello there."),
		TopicArn: aws.String("arn:aws:sns:eu-central-1:000000000000:my-topic"),
	})

	assert.NoError(t, err)

	// receive the message from sqs
	receiveOutput, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: aws.String("http://localhost:9871/queue/my-queue"),
	})

	assert.NoError(t, err)
	assert.Len(t, receiveOutput.Messages, 1)
	assert.Contains(t, *receiveOutput.Messages[0].Body, "Hello there.")

	test.Shutdown()
}

func Test_sns_kinesis(t *testing.T) {
	setup(t)

	test.Boot(mdl.String("config.sns_sqs.test.yml"))

	snsClient := test.ProvideSnsClient("sns")
	topicsOutput, err := snsClient.ListTopics(&sns.ListTopicsInput{})

	assert.NoError(t, err)
	assert.Len(t, topicsOutput.Topics, 0)

	sqsClient := test.ProvideSqsClient("sqs")
	queuesOutput, err := sqsClient.ListQueues(&sqs.ListQueuesInput{})

	assert.NoError(t, err)
	assert.Len(t, queuesOutput.QueueUrls, 0)

	// create a queue
	createQueueOutput, err := sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("my-queue"),
	})

	assert.NoError(t, err)
	assert.Equal(t, *createQueueOutput.QueueUrl, "http://localhost:9871/queue/my-queue")

	// create a topic
	createTopicOutput, err := snsClient.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String("my-topic"),
	})

	assert.NoError(t, err)
	assert.Equal(t, *createTopicOutput.TopicArn, "arn:aws:sns:eu-central-1:000000000000:my-topic")

	// create a topic subscription
	subscriptionOutput, err := snsClient.Subscribe(&sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		Endpoint: aws.String("http://localhost:9871/queue/my-queue"),
		TopicArn: aws.String("arn:aws:sns:eu-central-1:000000000000:my-topic"),
	})

	assert.NoError(t, err)
	assert.Contains(t, *subscriptionOutput.SubscriptionArn, "arn:aws:sns:eu-central-1:000000000000:my-topic:")

	// send a message to a topic
	_, err = snsClient.Publish(&sns.PublishInput{
		Message:  aws.String("Hello there."),
		TopicArn: aws.String("arn:aws:sns:eu-central-1:000000000000:my-topic"),
	})

	assert.NoError(t, err)

	// receive the message from sqs
	receiveOutput, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl: aws.String("http://localhost:9871/queue/my-queue"),
	})

	assert.NoError(t, err)
	assert.Len(t, receiveOutput.Messages, 1)
	assert.Contains(t, *receiveOutput.Messages[0].Body, "Hello there.")

	test.Shutdown()
}
