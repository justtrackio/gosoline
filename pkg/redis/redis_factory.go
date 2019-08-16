package redis

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/go-redis/redis"
	"net"
	"sync"
)

const (
	redisModeLocal    = "local"
	redisModeDiscover = "discover"
	DefaultClientName = "default"
)

var mutex sync.Mutex
var clients = map[string]Client{}

type selection struct {
	cfg.AppId
	name string
	mode string
	addr string
}

func readSelection(config cfg.Config, name string) *selection {
	modeStr := fmt.Sprintf("redis_%s_mode", name)
	addrStr := fmt.Sprintf("redis_%s_addr", name)

	sel := &selection{}
	sel.PadFromConfig(config)

	sel.name = name
	sel.mode = config.GetString(modeStr)
	sel.addr = config.GetString(addrStr)

	return sel
}

func GetClient(config cfg.Config, logger mon.Logger, name string) Client {
	sel := readSelection(config, name)

	switch sel.mode {
	case redisModeLocal:
		logger.Infof("using local redis %s with address %s", name, sel.addr)
		return GetClientWithAddress(sel.addr, sel.name)
	case redisModeDiscover:
		return GetClientFromDiscovery(logger, sel)
	}

	return nil
}

func GetClientFromDiscovery(logger mon.Logger, sel *selection) Client {
	addr := sel.addr

	if addr == "" {
		addr = fmt.Sprintf("%s.redis.%s.%s", sel.name, sel.Environment, sel.Family)
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
	logger.Infof("found redis server %s with address %s", sel.name, addr)

	return GetClientWithAddress(addr, sel.name)
}

func GetClientWithAddress(address, name string) Client {
	mutex.Lock()
	defer mutex.Unlock()

	if client, ok := clients[address]; ok {
		return client
	}

	baseClient := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    address,
	})

	clients[address] = NewRedisClient(baseClient, name)

	return clients[address]
}
