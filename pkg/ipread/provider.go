package ipread

import (
	"context"
	"net"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/oschwald/geoip2-golang"
)

type Provider interface {
	City(ipAddress net.IP) (*geoip2.City, error)
	Refresh(ctx context.Context) error
	Close() error
}

type ProviderFactory func(ctx context.Context, config cfg.Config, logger log.Logger, name string) (Provider, error)

var providers = map[string]ProviderFactory{
	"maxmind": NewMaxmindProvider,
	"memory":  NewMemoryProvider,
}
