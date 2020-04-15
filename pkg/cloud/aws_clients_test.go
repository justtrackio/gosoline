package cloud_test

import (
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/cloud"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetDynamoDbClient(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	config := new(configMocks.Config)
	config.On("GetString", "aws_dynamoDb_endpoint").Return("127.0.0.1")
	config.On("GetInt", "aws_sdk_retries").Return(0)

	_ = cloud.GetDynamoDbClient(config, logger)

	config.AssertExpectations(t)
}

func TestGetKinesisClient(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	config := new(configMocks.Config)
	config.On("GetString", "aws_kinesis_endpoint").Return("127.0.0.1")
	config.On("GetInt", "aws_sdk_retries").Return(0)

	_ = cloud.GetKinesisClient(config, logger)

	config.AssertExpectations(t)
}

func TestGetEcsClient(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	config := new(configMocks.Config)
	config.On("GetString", "aws_ecs_endpoint").Return("127.0.0.1")

	_ = cloud.GetEcsClient(config, logger)
}

func TestGetServiceDiscoveryClient(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()

	_ = cloud.GetServiceDiscoveryClient(logger, "")
}

func TestPrefixedLogger(t *testing.T) {
	l := monMocks.NewLoggerMock()
	l.On("WithFields", map[string]interface{}{
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
