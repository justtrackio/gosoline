package redis

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Naming struct {
	Pattern string `cfg:"pattern,nodecode" default:"{name}.{group}.redis.{env}.{family}"`
}

type Settings struct {
	cfg.AppId
	DB              int    `cfg:"db" default:"0"`
	Name            string `cfg:"name"`
	Dialer          string `cfg:"dialer" default:"tcp"`
	Address         string `cfg:"address" default:"127.0.0.1:6379"`
	Naming          Naming `cfg:"naming"`
	BackoffSettings exec.BackoffSettings
}

type redisCacheKey string

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Client, error) {
	settings := ReadSettings(config, name)
	cacheKey := fmt.Sprintf("%s:%s", settings.Address, name)

	return appctx.Provide(ctx, redisCacheKey(cacheKey), func() (Client, error) {
		return NewClient(ctx, config, logger, name)
	})
}

func GetRedisConfigKey(name string) string {
	return fmt.Sprintf("redis.%s", name)
}

func ReadSettings(config cfg.Config, name string) (*Settings, error) {
	key := GetRedisConfigKey(name)

	// This is a hack to ensure default redis config is populated,
	// because cfg.UnmarshalWithDefaultsFromKey does only read from already set config but not from env vars
	config.UnmarshalKey("redis.default", &Settings{})

	settings := &Settings{}
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("redis.default", "."))

	backoffSettings, err := exec.ReadBackoffSettings(config, key, "redis.default")
	if err != nil {
		return nil, fmt.Errorf("failed to read backoff settings for redis client %s: %w", name, err)
	}
	settings.BackoffSettings = backoffSettings

	if settings.Name == "" {
		settings.Name = name
	}

	return settings, nil
}
