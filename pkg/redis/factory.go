package redis

import (
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

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

	// TODO: This is a hack to ensure default redis config is populated,
	// 		 because cfg.UnmarshalWithDefaultsFromKey does only read from already set config but not from env vars
	config.UnmarshalKey("redis.default", &Settings{})

	settings := &Settings{}
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("redis.default", "."))

	settings.BackoffSettings = exec.ReadBackoffSettings(config, key, "redis.default")

	if settings.Name == "" {
		settings.Name = name
	}

	return settings
}
