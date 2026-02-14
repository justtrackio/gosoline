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
	config.EXPECT().GetString("realm").Return("rlm", nil)

	appId, err := cfg.GetAppIdFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
		Realm:       "rlm",
	}, appId)
}

func TestAppId_PadFromConfig(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app_project").Return("prj", nil)
	config.EXPECT().GetString("app_family").Return("fam", nil)
	config.EXPECT().GetString("app_group").Return("grp", nil)
	config.EXPECT().GetString("app_name").Return("name", nil)
	config.EXPECT().GetString("env").Return("test", nil)
	config.EXPECT().GetString("realm").Return("rlm", nil)

	appId := cfg.AppId{}
	err := appId.PadFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppId{
		Project:     "prj",
		Environment: "test",
		Family:      "fam",
		Group:       "grp",
		Application: "name",
		Realm:       "rlm",
	}, appId)

	config.AssertExpectations(t)
}

func TestAppId_ReplaceMacros(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "{project}-{env}-{family}-{group}",
	}

	pattern := "{realm}-{app}"
	result := appId.ReplaceMacros(pattern)
	assert.Equal(t, "myproject-test-myfamily-mygroup-myapp", result)
}

func TestAppId_ReplaceMacros_EmptyValues(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "",
		Family:      "myfamily",
		Group:       "",
		Application: "myapp",
		Realm:       "{project}-{env}-{family}-{group}",
	}

	pattern := "{realm}-{app}"
	result := appId.ReplaceMacros(pattern)
	assert.Equal(t, "myproject--myfamily--myapp", result)
}

func TestAppId_ReplaceMacros_WithRealm(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "{project}-{env}-{family}-{group}",
	}

	pattern := "{realm}-{streamName}"
	extraMacros := []cfg.MacroValue{
		{"streamName", "mystream"},
	}
	result := appId.ReplaceMacros(pattern, extraMacros...)
	assert.Equal(t, "myproject-test-myfamily-mygroup-mystream", result)
}

func TestAppId_ReplaceMacros_ExtraMacrosOrdering(t *testing.T) {
	appId := cfg.AppId{
		Project:     "myproject",
		Environment: "test",
		Family:      "myfamily",
		Group:       "mygroup",
		Application: "myapp",
		Realm:       "{project}-{env}",
	}

	// Test that extra macros are replaced before and after AppId macros
	pattern := "{prefix}-{realm}-{suffix}"
	extraMacros := []cfg.MacroValue{
		{"prefix", "before-{env}"},
		{"suffix", "after-{env}"},
	}
	result := appId.ReplaceMacros(pattern, extraMacros...)
	assert.Equal(t, "before-test-myproject-test-after-test", result)
}
