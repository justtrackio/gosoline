package redis

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/go-redis/redis"
	"net"
	"sync"
	"time"
)

const (
	redisModeLocal    = "local"
	redisModeDiscover = "discover"
	DefaultClientName = "default"
)

var mutex sync.Mutex
var clients = map[string]Client{}

type Selection struct {
	cfg.AppId
	Addr     string
	Settings *Settings
	mode     string
}

func readSelection(config cfg.Config, name string) *Selection {
	modeStr := fmt.Sprintf("redis_%s_mode", name)
	addrStr := fmt.Sprintf("redis_%s_addr", name)

	sel := &Selection{}
	sel.PadFromConfig(config)

	sel.Addr = config.GetString(addrStr)
	sel.mode = config.GetString(modeStr)

	sel.Settings = &Settings{
		Name:    name,
		Backoff: readBackoffConfig(config, name),
	}

	return sel
}

func readBackoffConfig(config cfg.Config, name string) SettingsBackoff {
	initialIntervalStr := fmt.Sprintf("redis_%s_backoff_initial_interval", name)
	randomizationFactorStr := fmt.Sprintf("redis_%s_backoff_randomization_factor", name)
	multiplierStr := fmt.Sprintf("redis_%s_backoff_multiplier", name)
	maxIntervalStr := fmt.Sprintf("redis_%s_backoff_max_interval", name)
	maxElapsedTimeStr := fmt.Sprintf("redis_%s_backoff_max_elapsed_time", name)

	initialInterval := 1 * time.Second
	randomizationFactor := 0.2
	multiplier := 3.0
	maxInterval := 30 * time.Second
	maxElapsedTime := 0 * time.Second

	if config.IsSet(initialIntervalStr) {
		initialInterval = config.GetDuration(initialIntervalStr) * time.Second
	}

	if config.IsSet(randomizationFactorStr) {
		randomizationFactor = config.GetFloat64(randomizationFactorStr)
	}

	if config.IsSet(multiplierStr) {
		multiplier = config.GetFloat64(multiplierStr)
	}

	if config.IsSet(maxIntervalStr) {
		maxInterval = config.GetDuration(maxIntervalStr) * time.Second
	}

	if config.IsSet(maxElapsedTimeStr) {
		maxElapsedTime = config.GetDuration(maxElapsedTimeStr) * time.Second
	}

	return SettingsBackoff{
		InitialInterval:     initialInterval,
		RandomizationFactor: randomizationFactor,
		Multiplier:          multiplier,
		MaxInterval:         maxInterval,
		MaxElapsedTime:      maxElapsedTime,
	}
}

func GetClient(config cfg.Config, logger mon.Logger, name string) Client {
	sel := readSelection(config, name)

	switch sel.mode {
	case redisModeLocal:
		logger.Infof("using local redis %s with address %s", name, sel.Addr)
		return GetClientWithAddress(logger, sel)
	case redisModeDiscover:
		return GetClientFromDiscovery(logger, sel)
	}

	return nil
}

func GetClientFromDiscovery(logger mon.Logger, sel *Selection) Client {
	addr := sel.Addr

	if addr == "" {
		addr = fmt.Sprintf("%s.redis.%s.%s", sel.Settings.Name, sel.Environment, sel.Family)
	}

	_, srvs, err := net.LookupSRV("", "", addr)

	if err != nil {
		logger.Fatal(err, "could not lookup the redis src dns record")
	}

	if len(srvs) != 1 {
		msg := fmt.Sprintf("there should be exactly one redis instance, found: %v", len(srvs))
		logger.Fatal(errors.New("redis instance count mismatch"), msg)
	}

	addr = fmt.Sprintf("%v:%v", srvs[0].Target, srvs[0].Port)
	logger.Infof("found redis server %s with address %s", sel.Settings.Name, addr)

	return GetClientWithAddress(logger, sel)
}

func GetClientWithAddress(logger mon.Logger, sel *Selection) Client {
	mutex.Lock()
	defer mutex.Unlock()

	if client, ok := clients[sel.Addr]; ok {
		return client
	}

	baseClient := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    sel.Addr,
	})

	clients[sel.Addr] = NewRedisClient(logger, baseClient, *sel.Settings)

	return clients[sel.Addr]
}
