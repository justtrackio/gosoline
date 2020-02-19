//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_sqs(t *testing.T) {
	setup(t)

	pkgTest.Boot("test_configs/config.sqs.test.yml")
	defer pkgTest.Shutdown()

	sqsClient := pkgTest.ProvideSqsClient("sqs")
	o, err := sqsClient.ListQueues(&sqs.ListQueuesInput{})

	assert.NoError(t, err)
	assert.Len(t, o.QueueUrls, 0)
}
