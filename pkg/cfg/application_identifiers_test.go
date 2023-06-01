package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetAppIdFromConfig(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "app_project").Return("prj")
	config.On("GetString", "app_family").Return("fam")
	config.On("GetString", "app_group").Return("grp")
	config.On("GetString", "app_name").Return("name")
	config.On("GetString", "env").Return("test")

	appId := cfg.GetAppIdFromConfig(config)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
	}, appId)

	config.AssertExpectations(t)
}

func TestAppId_PadFromConfig(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "app_project").Return("prj")
	config.On("GetString", "app_family").Return("fam")
	config.On("GetString", "app_group").Return("grp")
	config.On("GetString", "app_name").Return("name")
	config.On("GetString", "env").Return("test")

	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
	}, appId)

	config.AssertExpectations(t)
}
