package assert

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
)

func SqsQueueExists(t *testing.T, client *sqs.SQS, queueName string) {
	queueUrlOutput, err := client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})

	assert.NotNil(t, queueUrlOutput)
	assert.NoError(t, err)
}

func SqsQueueContainsMessages(t *testing.T, client *sqs.SQS, queueName string, count int) []*sqs.Message {
	queueUrlOutput, err := client.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queueName,
	})

	assert.NotNil(t, queueUrlOutput)
	assert.NoError(t, err)

	messages, err := client.ReceiveMessage(&sqs.ReceiveMessageInput{
		MaxNumberOfMessages: mdl.Int64(10),
		QueueUrl:            queueUrlOutput.QueueUrl,
	})

	assert.Nil(t, err)
	assert.NotNil(t, messages)
	assert.Len(t, messages.Messages, count)

	return messages.Messages
}
