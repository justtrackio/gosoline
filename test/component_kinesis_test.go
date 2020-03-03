//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_kinesis(t *testing.T) {
	setup(t)

	mocks := pkgTest.Boot("test_configs/config.kinesis.test.yml")
	defer mocks.Shutdown()

	kinClient := mocks.ProvideClient("kinesis", "kinesis").(*kinesis.Kinesis)
	o, err := kinClient.ListStreams(&kinesis.ListStreamsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.StreamNames, 0)
}
