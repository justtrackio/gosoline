package sns_test

import (
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/sns"
	"testing"
)

func TestGetClient(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "aws_sns_endpoint").Return("127.0.0.1")

	logger := logMocks.NewLoggerMockedAll()

	sns.ProvideClient(config, logger, &sns.Settings{})

	config.AssertExpectations(t)
}
