package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetAppIdentityFromConfig_WithMocks(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app.name").Return("name", nil)
	config.EXPECT().GetString("app.env").Return("test", nil)
	config.EXPECT().GetStringMapString("app.tags", map[string]string{}).Return(map[string]string{
		"project": "prj",
		"family":  "fam",
		"group":   "grp",
	}, nil)

	identity, err := cfg.GetAppIdentityFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppIdentity{
		Name: "name",
		Env:  "test",
		Tags: cfg.AppTags{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}, identity)
}

func TestAppIdentity_PadFromConfig_WithMocks(t *testing.T) {
	config := cfgMocks.NewConfig(t)
	config.EXPECT().GetString("app.name").Return("name", nil)
	config.EXPECT().GetString("app.env").Return("test", nil)
	config.EXPECT().GetStringMapString("app.tags", map[string]string{}).Return(map[string]string{
		"project": "prj",
		"family":  "fam",
		"group":   "grp",
	}, nil)

	identity := cfg.AppIdentity{}
	err := identity.PadFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppIdentity{
		Name: "name",
		Env:  "test",
		Tags: cfg.AppTags{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}, identity)

	config.AssertExpectations(t)
}
