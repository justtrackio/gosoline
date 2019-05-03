package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAppIdFromConfig(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "app_project").Return("prj")
	config.On("GetString", "app_family").Return("fam")
	config.On("GetString", "app_name").Return("name")
	config.On("GetString", "env").Return("test")

	appId := cfg.GetAppIdFromConfig(config)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Application: "name",
	}, appId)

	config.AssertExpectations(t)
}

func TestAppId_PadFromConfig(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "app_project").Return("prj")
	config.On("GetString", "app_family").Return("fam")
	config.On("GetString", "app_name").Return("name")
	config.On("GetString", "env").Return("test")

	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Application: "name",
	}, appId)

	config.AssertExpectations(t)
}
