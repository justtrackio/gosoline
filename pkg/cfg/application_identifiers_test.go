package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetAppIdFromConfig(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app_project").Return("prj")
	config.EXPECT().GetString("app_family").Return("fam")
	config.EXPECT().GetString("app_group").Return("grp")
	config.EXPECT().GetString("app_name").Return("name")
	config.EXPECT().GetString("env").Return("test")

	appId := cfg.GetAppIdFromConfig(config)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
	}, appId)
}

func TestAppId_PadFromConfig(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app_project").Return("prj")
	config.EXPECT().GetString("app_family").Return("fam")
	config.EXPECT().GetString("app_group").Return("grp")
	config.EXPECT().GetString("app_name").Return("name")
	config.EXPECT().GetString("env").Return("test")

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
