package assert

import (
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"testing"
)

func SnsTopicExists(t *testing.T, client *sns.SNS, topicArn string) {
	getTopicAttributesOutput, err := client.GetTopicAttributes(&sns.GetTopicAttributesInput{
		TopicArn: &topicArn,
	})

	assert.NotNil(t, getTopicAttributesOutput)
	assert.NoError(t, err)
}
