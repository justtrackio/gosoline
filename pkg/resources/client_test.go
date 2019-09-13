package resources_test

import (
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/resources"
	"testing"
)

func TestGetResourcesManagerClient(t *testing.T) {
	logger := monMocks.NewLoggerMockedAll()
	config := new(configMocks.Config)

	config.On("GetString", "aws_rgt_endpoint").Return("")
	config.On("GetInt", "aws_sdk_retries").Return(0)

	_ = resources.GetClient(config, logger)

	config.AssertExpectations(t)
}
