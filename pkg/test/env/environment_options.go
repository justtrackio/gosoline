package env

import (
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type (
	Option          func(env *Environment)
	ComponentOption func(componentConfigManger *ComponentsConfigManager) error
	ConfigOption    func(config cfg.GosoConf) error
	LoggerOption    func(settings *LoggerSettings) error
)

func WithComponent(settings ComponentBaseSettingsAware) Option {
	return func(env *Environment) {
		env.addComponentOption(func(componentConfigManger *ComponentsConfigManager) error {
			return componentConfigManger.Add(settings)
		})
	}
}

func WithConfigFile(file string) Option {
	return func(env *Environment) {
		env.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigFile(file, "yml"))
		})
	}
}

func WithConfigEnvKeyReplacer(replacer *strings.Replacer) Option {
	return func(env *Environment) {
		env.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithEnvKeyReplacer(replacer))
		})
	}
}

func WithConfigMap(settings map[string]any) Option {
	return func(env *Environment) {
		env.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigMap(settings))
		})
	}
}

func WithConfigSetting(key string, settings any) Option {
	return func(env *Environment) {
		env.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigSetting(key, settings))
		})
	}
}

func WithContainerExpireAfter(expireAfter time.Duration) Option {
	return func(env *Environment) {
		env.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigSetting("test.container_runner.expire_after", expireAfter.String()))
		})
	}
}

func WithLoggerLevel(level string) Option {
	return func(env *Environment) {
		env.addLoggerOption(func(settings *LoggerSettings) error {
			settings.Level = level

			return nil
		})
	}
}

func WithLogRecording() Option {
	return func(env *Environment) {
		env.addLoggerOption(func(settings *LoggerSettings) error {
			settings.RecordLogs = true

			return nil
		})
	}
}

func WithoutAutoDetectedComponents(components ...string) Option {
	return func(env *Environment) {
		env.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigSetting("test.auto_detect.skip_components", components))
		})
	}
}
