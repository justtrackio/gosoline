//+build integration

package test_test

import (
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/test"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_kinesis(t *testing.T) {
	setup(t)

	test.Boot(mdl.String("test_configs/config.kinesis.test.yml"))
	defer test.Shutdown()

	kinClient := test.ProvideKinesisClient("kinesis")
	o, err := kinClient.ListStreams(&kinesis.ListStreamsInput{})

	assert.NoError(t, err)
	assert.Len(t, o.StreamNames, 0)
}
