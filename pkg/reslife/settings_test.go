package reslife_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/stretchr/testify/assert"
)

func TestSettingsDefaults(t *testing.T) {
	var tests = []struct {
		name     string
		config   map[string]any
		expected *reslife.Settings
	}{
		{
			name:   "all defaults",
			config: map[string]any{},
			expected: &reslife.Settings{
				Create:   reslife.SettingsCycle{Enabled: false},
				Init:     reslife.SettingsCycle{Enabled: true},
				Register: reslife.SettingsCycle{Enabled: true},
				Purge:    reslife.SettingsCycle{Enabled: false},
			},
		},
		{
			name: "all disabled",
			config: map[string]any{
				"create": map[string]any{
					"enabled": false,
				},
				"init": map[string]any{
					"enabled": false,
				},
				"register": map[string]any{
					"enabled": false,
				},
				"purge": map[string]any{
					"enabled": false,
				},
			},
			expected: &reslife.Settings{
				Create:   reslife.SettingsCycle{Enabled: false},
				Init:     reslife.SettingsCycle{Enabled: false},
				Register: reslife.SettingsCycle{Enabled: false},
				Purge:    reslife.SettingsCycle{Enabled: false},
			},
		},
		{
			name: "all enabled",
			config: map[string]any{
				"create": map[string]any{
					"enabled": true,
				},
				"init": map[string]any{
					"enabled": true,
				},
				"register": map[string]any{
					"enabled": true,
				},
				"purge": map[string]any{
					"enabled": true,
				},
			},
			expected: &reslife.Settings{
				Create:   reslife.SettingsCycle{Enabled: true},
				Init:     reslife.SettingsCycle{Enabled: true},
				Register: reslife.SettingsCycle{Enabled: true},
				Purge:    reslife.SettingsCycle{Enabled: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"resource_lifecycles": test.config})
			settings := reslife.ReadSettings(config)

			assert.Equal(t, test.expected, settings)
		})
	}
}
