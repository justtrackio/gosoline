package ipread

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/oschwald/geoip2-golang"
	"net"
)

type Provider interface {
	City(ipAddress net.IP) (*geoip2.City, error)
}

type ProviderFactory func(config cfg.Config, logger log.Logger, name string) (Provider, error)

var providers = map[string]ProviderFactory{
	"maxmind": NewMaxmindProvider,
	"memory":  NewMemoryProvider,
}
