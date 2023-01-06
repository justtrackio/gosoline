package redis

import (
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Settings struct {
	cfg.AppId
	Name            string `cfg:"name"`
	Dialer          string `cfg:"dialer" default:"tcp"`
	Address         string `cfg:"address" default:"127.0.0.1:6379"`
	BackoffSettings exec.BackoffSettings
}

var (
	mutex   sync.Mutex
	clients = map[string]Client{}
)

func ProvideClient(config cfg.Config, logger log.Logger, name string) (Client, error) {
	mutex.Lock()
	defer mutex.Unlock()

	settings := ReadSettings(config, name)
	cacheKey := fmt.Sprintf("%s:%s", settings.Address, name)

	if client, ok := clients[cacheKey]; ok {
		return client, nil
	}

	var err error
	if clients[cacheKey], err = NewClient(config, logger, name); err != nil {
		return nil, err
	}

	return clients[cacheKey], nil
}

func ReadSettings(config cfg.Config, name string) *Settings {
	key := fmt.Sprintf("redis.%s", name)

	settings := &Settings{}
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("redis.default", "."))

	settings.BackoffSettings = exec.ReadBackoffSettings(config, key, "redis.default")

	if settings.Name == "" {
		settings.Name = name
	}

	return settings
}
