package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppTags_Get(t *testing.T) {
	t.Run("returns value for existing key", func(t *testing.T) {
		tags := cfg.AppTags{"project": "myproject", "custom": "value"}
		assert.Equal(t, "myproject", tags.Get("project"))
		assert.Equal(t, "value", tags.Get("custom"))
	})

	t.Run("returns empty string for missing key", func(t *testing.T) {
		tags := cfg.AppTags{"project": "myproject"}
		assert.Empty(t, tags.Get("missing"))
	})

	t.Run("handles nil map", func(t *testing.T) {
		var tags cfg.AppTags
		assert.Empty(t, tags.Get("anything"))
	})
}

// Tests for GetAppIdentityFromConfig

func TestGetAppIdentityFromConfig(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "production",
			"name": "myapp",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
	})

	identity, err := cfg.GetAppIdentityFromConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "myapp", identity.Name)
	assert.Equal(t, "production", identity.Env)
	assert.Equal(t, "myproject", identity.Tags.Get("project"))
	assert.Equal(t, "myfamily", identity.Tags.Get("family"))
	assert.Equal(t, "mygroup", identity.Tags.Get("group"))
}

func TestGetAppIdentityFromConfig_CustomTags(t *testing.T) {
	// Test that custom tags beyond project/family/group work
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "myapp",
			"tags": map[string]any{
				"project":    "myproject",
				"family":     "myfamily",
				"group":      "mygroup",
				"custom_tag": "custom_value",
				"team":       "platform",
			},
		},
	})

	identity, err := cfg.GetAppIdentityFromConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "myapp", identity.Name)
	assert.Equal(t, "myproject", identity.Tags.Get("project"))
	assert.Equal(t, "myfamily", identity.Tags.Get("family"))
	assert.Equal(t, "mygroup", identity.Tags.Get("group"))
	assert.Equal(t, "custom_value", identity.Tags.Get("custom_tag"))
	assert.Equal(t, "platform", identity.Tags.Get("team"))
}

func TestGetAppIdentityFromConfig_NoTagsAllowed(t *testing.T) {
	// AppIdentity does NOT require project/family/group tags
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "myapp",
			// no tags - should succeed
		},
	})

	identity, err := cfg.GetAppIdentityFromConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "myapp", identity.Name)
	assert.Equal(t, "test", identity.Env)
	assert.Empty(t, identity.Tags)
}

func TestGetAppIdentityFromConfig_MissingName(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env": "test",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
		// missing app.name
	})

	_, err := cfg.GetAppIdentityFromConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.name")
}

func TestGetAppIdentityFromConfig_EmptyName(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
	})

	_, err := cfg.GetAppIdentityFromConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.name")
	assert.Contains(t, err.Error(), "empty")
}

func TestGetAppIdentityFromConfig_MissingEnv(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "myapp",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
		// missing app.env
	})

	_, err := cfg.GetAppIdentityFromConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.env")
}

func TestGetAppIdentityFromConfig_EmptyEnv(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "",
			"name": "myapp",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
	})

	_, err := cfg.GetAppIdentityFromConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.env")
}

func TestGetAppIdentityFromConfig_WhitespaceEnv(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "   ",
			"name": "myapp",
		},
	})

	_, err := cfg.GetAppIdentityFromConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.env")
}

func TestGetAppIdentityFromConfig_WhitespaceName(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "   ",
		},
	})

	_, err := cfg.GetAppIdentityFromConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app.name")
}

func TestAppIdentity_PadFromConfig(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "myapp",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
	})

	identity := cfg.AppIdentity{}
	err := identity.PadFromConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "myapp", identity.Name)
	assert.Equal(t, "test", identity.Env)
	assert.Equal(t, "myproject", identity.Tags.Get("project"))
	assert.Equal(t, "myfamily", identity.Tags.Get("family"))
	assert.Equal(t, "mygroup", identity.Tags.Get("group"))
}

func TestAppIdentity_PadFromConfig_PartiallyFilled(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "myapp",
			"tags": map[string]any{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		},
	})

	// Pre-fill some fields
	identity := cfg.AppIdentity{
		Name: "existing-name",
		Env:  "existing-env",
		Tags: cfg.AppTags{"project": "existing-project"},
	}
	err := identity.PadFromConfig(config)
	require.NoError(t, err)

	// Pre-filled values should be preserved
	assert.Equal(t, "existing-name", identity.Name)
	assert.Equal(t, "existing-env", identity.Env)
	assert.Equal(t, "existing-project", identity.Tags.Get("project"))
	// Missing tags should be filled from config
	assert.Equal(t, "myfamily", identity.Tags.Get("family"))
	assert.Equal(t, "mygroup", identity.Tags.Get("group"))
}

func TestAppIdentity_PadFromConfig_AllFieldsSet(t *testing.T) {
	config := cfg.New(map[string]any{
		// Config values don't matter if all fields are set
	})

	identity := cfg.AppIdentity{
		Name: "app",
		Env:  "env",
		Tags: cfg.AppTags{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}
	err := identity.PadFromConfig(config)
	require.NoError(t, err)

	// All values should remain unchanged
	assert.Equal(t, "app", identity.Name)
	assert.Equal(t, "env", identity.Env)
	assert.Equal(t, "prj", identity.Tags.Get("project"))
	assert.Equal(t, "fam", identity.Tags.Get("family"))
	assert.Equal(t, "grp", identity.Tags.Get("group"))
}

func TestAppIdentity_PadFromConfig_NoTagsRequired(t *testing.T) {
	// PadFromConfig does NOT require project/family/group tags
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "myapp",
			// no tags
		},
	})

	identity := cfg.AppIdentity{}
	err := identity.PadFromConfig(config)
	require.NoError(t, err)

	assert.Equal(t, "myapp", identity.Name)
	assert.Equal(t, "test", identity.Env)
}

func TestAppIdentity_RequireTags(t *testing.T) {
	t.Run("succeeds when all required tags present", func(t *testing.T) {
		identity := cfg.AppIdentity{
			Tags: cfg.AppTags{
				"project": "myproject",
				"family":  "myfamily",
				"group":   "mygroup",
			},
		}

		err := identity.RequireTags("project", "family", "group")
		require.NoError(t, err)
	})

	t.Run("fails when tags missing", func(t *testing.T) {
		identity := cfg.AppIdentity{
			Tags: cfg.AppTags{
				"project": "myproject",
				// missing family and group
			},
		}

		err := identity.RequireTags("project", "family", "group")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required tags")
		assert.Contains(t, err.Error(), "family")
		assert.Contains(t, err.Error(), "group")
		assert.NotContains(t, err.Error(), "project")
	})

	t.Run("treats whitespace-only as missing", func(t *testing.T) {
		identity := cfg.AppIdentity{
			Tags: cfg.AppTags{
				"project": "myproject",
				"family":  "   ", // whitespace only
				"group":   "",    // empty
			},
		}

		err := identity.RequireTags("project", "family", "group")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required tags")
		assert.Contains(t, err.Error(), "family")
		assert.Contains(t, err.Error(), "group")
	})

	t.Run("sorts missing tags in error message", func(t *testing.T) {
		identity := cfg.AppIdentity{
			Tags: cfg.AppTags{},
		}

		err := identity.RequireTags("zebra", "apple", "mango")
		require.Error(t, err)
		// Should be sorted: apple, mango, zebra
		assert.Contains(t, err.Error(), "missing required tags: apple, mango, zebra")
	})

	t.Run("handles nil tags", func(t *testing.T) {
		identity := cfg.AppIdentity{}

		err := identity.RequireTags("project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required tags: project")
	})
}

func TestAppIdentity_String(t *testing.T) {
	identity := cfg.AppIdentity{
		Name: "app",
		Env:  "env",
		Tags: cfg.AppTags{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}

	assert.Equal(t, "prj-env-fam-grp-app", identity.String())
}

func TestAppIdentity_StringWithEmptyFields(t *testing.T) {
	identity := cfg.AppIdentity{
		Name: "app",
		Env:  "",
		Tags: cfg.AppTags{
			"project": "prj",
			"family":  "fam",
			// group missing
		},
	}

	assert.Equal(t, "prj-fam-app", identity.String())
}

func TestAppIdentity_StringMinimal(t *testing.T) {
	identity := cfg.AppIdentity{
		Name: "app",
		Env:  "production",
		// no tags
	}

	assert.Equal(t, "production-app", identity.String())
}
