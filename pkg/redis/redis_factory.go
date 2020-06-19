package redis

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"sync"
	"time"
)

const (
	RedisModeDiscover = "discover"
	RedisModeLocal    = "local"
)

var mutex sync.Mutex
var clients = map[string]Client{}

func GetClient(config cfg.Config, logger mon.Logger, name string) Client {
	settings := readSettings(config, name)

	return GetClientFromSettings(logger, settings)
}

func GetClientFromSettings(logger mon.Logger, settings *Settings) Client {
	mutex.Lock()
	defer mutex.Unlock()

	if client, ok := clients[settings.Name]; ok {
		return client
	}

	clients[settings.Name] = NewRedisClient(logger, settings)

	return clients[settings.Name]
}

func readSettings(config cfg.Config, name string) *Settings {
	modeStr := fmt.Sprintf("redis_%s_mode", name)
	addrStr := fmt.Sprintf("redis_%s_addr", name)

	settings := &Settings{}
	settings.PadFromConfig(config)

	settings.Name = name
	settings.Mode = config.GetString(modeStr)
	settings.Address = config.GetString(addrStr)
	settings.BackoffSettings = readBackoffSettings(config, name)

	return settings
}

func readBackoffSettings(config cfg.Config, name string) BackoffSettings {
	backoffSettingsStr := fmt.Sprintf("redis_%s_backoff", name)

	settings := BackoffSettings{
		InitialInterval:     1 * time.Second,
		RandomizationFactor: 0.2,
		Multiplier:          3.0,
		MaxInterval:         30 * time.Second,
		MaxElapsedTime:      0 * time.Second,
	}

	if config.IsSet(backoffSettingsStr) {
		config.UnmarshalKey(backoffSettingsStr, &settings)
	}

	return settings
}
