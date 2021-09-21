package resources_test

import (
	"testing"

	configMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/resources"
)

func TestGetResourcesManagerClient(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	config := new(configMocks.Config)

	config.On("GetString", "aws_rgt_endpoint").Return("")
	config.On("GetInt", "aws_sdk_retries").Return(0)

	_ = resources.GetClient(config, logger)

	config.AssertExpectations(t)
}
