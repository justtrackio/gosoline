//+build integration

package test_test

import (
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/test"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_sqs(t *testing.T) {
	setup(t)

	test.Boot(mdl.String("test_configs/config.sqs.test.yml"))
	defer test.Shutdown()

	sqsClient := test.ProvideSqsClient("sqs")
	o, err := sqsClient.ListQueues(&sqs.ListQueuesInput{})

	assert.NoError(t, err)
	assert.Len(t, o.QueueUrls, 0)
}
