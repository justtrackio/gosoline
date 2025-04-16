package reslife

import "github.com/justtrackio/gosoline/pkg/cfg"

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

func ReadSettings(config cfg.Config) *Settings {
	settings := &Settings{}
	config.UnmarshalKey("resource_lifecycles", settings, []cfg.UnmarshalDefaults{
		cfg.UnmarshalWithDefaultForKey("init.enabled", true),
		cfg.UnmarshalWithDefaultForKey("register.enabled", true),
	}...)

	return settings
}
