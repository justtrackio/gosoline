package redis

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"sync"
)

type Settings struct {
	cfg.AppId
	Name            string               `cfg:"name"`
	Dialer          string               `cfg:"dialer" default:"tcp"`
	Address         string               `cfg:"address" default:"127.0.0.1:6379"`
	BackoffSettings exec.BackoffSettings `cfg:"backoff"`
}

var mutex sync.Mutex
var clients = map[string]Client{}

func ProvideClient(config cfg.Config, logger log.Logger, name string) (Client, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if client, ok := clients[name]; ok {
		return client, nil
	}

	var err error
	if clients[name], err = NewClient(config, logger, name); err != nil {
		return nil, err
	}

	return clients[name], nil
}

func ReadSettings(config cfg.Config, name string) *Settings {
	key := fmt.Sprintf("redis.%s", name)

	settings := &Settings{}
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("redis.default", "."))

	if settings.Name == "" {
		settings.Name = name
	}

	return settings
}
