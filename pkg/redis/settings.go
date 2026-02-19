package redis

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
)

type Naming struct {
	AddressPattern   string `cfg:"address_pattern,nodecode" default:"{name}.{app.tags.group}.redis.{app.env}.{app.tags.family}"`
	AddressDelimiter string `cfg:"address_delimiter,nodecode" default:"."`
	KeyPattern       string `cfg:"key_pattern,nodecode" default:"{key}"` // e.g. {app.namespace}-{app.name}-{key}
	KeyDelimiter     string `cfg:"key_delimiter,nodecode" default:"-"`
}

type Settings struct {
	cfg.Identity
	DB              int    `cfg:"db" default:"0"`
	Name            string `cfg:"name"`
	Dialer          string `cfg:"dialer" default:"tcp"`
	Address         string `cfg:"address" default:"127.0.0.1:6379"`
	Naming          Naming `cfg:"naming"`
	BackoffSettings exec.BackoffSettings
}

func GetRedisConfigKey(name string) string {
	return fmt.Sprintf("redis.%s", name)
}

func ReadSettings(config cfg.Config, name string) (*Settings, error) {
	var err error
	key := GetRedisConfigKey(name)

	// This is a hack to ensure default redis config is populated,
	// because cfg.UnmarshalWithDefaultsFromKey does only read from already set config but not from env vars
	if err = config.UnmarshalKey("redis.default", &Settings{}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal redis.default in ReadSettings: %w", err)
	}

	settings := &Settings{}
	if err = config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("redis.default", ".")); err != nil {
		return nil, fmt.Errorf("failed to unmarshal redis settings for key %q in ReadSettings: %w", key, err)
	}

	if settings.Name == "" {
		settings.Name = name
	}

	if err = settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad app identity from config for redis %q: %w", key, err)
	}

	if settings.Address == "" {
		settings.Address, err = settings.Format(settings.Naming.AddressPattern, settings.Naming.AddressDelimiter, map[string]string{
			"name": settings.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to format address for redis %q: %w", key, err)
		}
	}

	settings.BackoffSettings, err = exec.ReadBackoffSettings(config, key, "redis.default")
	if err != nil {
		return nil, fmt.Errorf("failed to read backoff settings for redis %q: %w", key, err)
	}

	return settings, nil
}
