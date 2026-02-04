package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
)

func TestGetAppIdentityFromConfig(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "name",
			"env":  "test",
			"tags": map[string]any{
				"project": "prj",
				"family":  "fam",
				"group":   "grp",
			},
		},
	})

	identity, err := cfg.GetAppIdentity(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppIdentity{
		Name: "name",
		Env:  "test",
		Tags: map[string]string{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}, identity)
}

func TestAppIdentity_PadFromConfig(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "name",
			"env":  "test",
			"tags": map[string]any{
				"project": "prj",
				"family":  "fam",
				"group":   "grp",
			},
		},
	})

	identity := cfg.AppIdentity{}
	err := identity.PadFromConfig(config)
	assert.NoError(t, err)

	assert.Equal(t, cfg.AppIdentity{
		Name: "name",
		Env:  "test",
		Tags: map[string]string{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}, identity)
}
