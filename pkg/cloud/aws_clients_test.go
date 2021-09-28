package cloud_test

import (
	"testing"

	configMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/cloud"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetDynamoDbClient(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()

	config := new(configMocks.Config)
	config.On("GetString", "aws_dynamoDb_endpoint").Return("127.0.0.1")
	config.On("GetInt", "aws_sdk_retries").Return(0)

	_ = cloud.GetDynamoDbClient(config, logger)

	config.AssertExpectations(t)
}

func TestGetKinesisClient(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()

	config := new(configMocks.Config)
	config.On("GetString", "aws_kinesis_endpoint").Return("127.0.0.1")
	config.On("GetInt", "aws_sdk_retries").Return(0)

	_ = cloud.GetKinesisClient(config, logger)

	config.AssertExpectations(t)
}

func TestPrefixedLogger(t *testing.T) {
	l := logMocks.NewLoggerMock()
	l.On("WithFields", log.Fields{
		"aws_service": "myService",
	}).Return(l)
	l.On("Info", "log")

	logger := cloud.PrefixedLogger(l, "myService")

	l.AssertNotCalled(t, "Info", "log")

	assert.NotPanics(t, func() {
		logger("log")
	})

	l.AssertCalled(t, "Info", "log")
	l.AssertExpectations(t)
}
