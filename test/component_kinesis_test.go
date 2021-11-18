//go:build integration
// +build integration

package test_test

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"testing"

	"github.com/aws/aws-sdk-go/service/kinesis"
	pkgTest "github.com/justtrackio/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
)

func Test_kinesis(t *testing.T) {
	t.Parallel()
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.kinesis.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	kinClient := mocks.ProvideKinesisClient("kinesis")

	_, err = kinClient.CreateStream(&kinesis.CreateStreamInput{
		ShardCount: aws.Int64(1),
		StreamName: aws.String("foo"),
	})
	assert.NoError(t, err)

	listOutput, err := kinClient.ListStreams(&kinesis.ListStreamsInput{})

	assert.NoError(t, err)
	assert.Len(t, listOutput.StreamNames, 1)
}
