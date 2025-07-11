package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetAppIdFromConfig(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app_project").Return("prj", nil)
	config.EXPECT().GetString("app_family").Return("fam", nil)
	config.EXPECT().GetString("app_group").Return("grp", nil)
	config.EXPECT().GetString("app_name").Return("name", nil)
	config.EXPECT().GetString("env").Return("test", nil)

	appId, err := cfg.GetAppIdFromConfig(config)
	assert.NoError(t, err)

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
	config.EXPECT().GetString("app_project").Return("prj", nil)
	config.EXPECT().GetString("app_family").Return("fam", nil)
	config.EXPECT().GetString("app_group").Return("grp", nil)
	config.EXPECT().GetString("app_name").Return("name", nil)
	config.EXPECT().GetString("env").Return("test", nil)

	appId := cfg.AppId{}
	err := appId.PadFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
	}, appId)

	config.AssertExpectations(t)
}
