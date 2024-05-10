package reslife

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type Settings struct {
	Create   SettingsCycle `cfg:"create"`
	Init     SettingsCycle `cfg:"init"`
	Register SettingsCycle `cfg:"register"`
	Purge    SettingsCycle `cfg:"purge"`
}

type SettingsCycle struct {
	Enabled  bool     `cfg:"enabled"`
	Excludes []string `cfg:"excludes"`
}

func ReadSettings(config cfg.Config) (*Settings, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey("resource_lifecycles", settings, []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultForKey("init.enabled", true),
		cfg.UnmarshalWithDefaultForKey("register.enabled", true),
	}...); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource_lifecycles in ReadSettings: %w", err)
	}

	return settings, nil
}
