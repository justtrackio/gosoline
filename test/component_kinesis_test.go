//go:build integration
// +build integration

package test_test

import (
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
	o, err := kinClient.ListStreams(&kinesis.ListStreamsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.StreamNames, 0)
}
