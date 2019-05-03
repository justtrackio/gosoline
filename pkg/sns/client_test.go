package sns_test

import (
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/sns"
	"testing"
)

func TestGetClient(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "aws_sns_endpoint").Return("127.0.0.1")
	config.On("GetInt", "aws_sdk_retries").Return(0)

	logger := mocks.NewLoggerMockedAll()

	sns.GetClient(config, logger)

	config.AssertExpectations(t)
}
