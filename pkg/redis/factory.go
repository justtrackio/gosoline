package redis

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
	"time"
)

type Settings struct {
	cfg.AppId
	Name            string          `cfg:"name"`
	Dialer          string          `cfg:"dialer" default:"tcp"`
	Address         string          `cfg:"address" default:"127.0.0.1:6379"`
	BackoffSettings BackoffSettings `cfg:"backoff"`
}

type BackoffSettings struct {
	InitialInterval     time.Duration `cfg:"initial_interval" default:"1s"`
	RandomizationFactor float64       `cfg:"randomization_factor" default:"0.2"`
	Multiplier          float64       `cfg:"multiplier" default:"3.0"`
	MaxInterval         time.Duration `cfg:"max_interval" default:"30s"`
	MaxElapsedTime      time.Duration `cfg:"max_elapsed_time" default:"0s"`
}

var mutex sync.Mutex
var clients = map[string]Client{}

func ProvideClient(config cfg.Config, logger mon.Logger, name string) Client {
	mutex.Lock()
	defer mutex.Unlock()

	if client, ok := clients[name]; ok {
		return client
	}

	clients[name] = NewClient(config, logger, name)

	return clients[name]
}

func ReadSettings(config cfg.Config, name string) *Settings {
	key := fmt.Sprintf("redis.%s", name)

	settings := &Settings{}

	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey("redis.default", "."))
	settings.PadFromConfig(config)

	if settings.Name == "" {
		settings.Name = name
	}

	return settings
}
