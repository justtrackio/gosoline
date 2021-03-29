package redis

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"net"
)

const (
	DialerSrv = "srv"
	DialerTcp = "tcp"
)

var dialers = map[string]Dialer{
	DialerSrv: dialerSrv,
	DialerTcp: dialerTcp,
}

type Dialer func(logger mon.Logger, settings *Settings) func(context.Context, string, string) (net.Conn, error)
type SrvNamingFactory func(appId cfg.AppId, name string) string

var srvNamingStrategy = func(appId cfg.AppId, name string) string {
	return fmt.Sprintf("%s.%s.redis.%s.%s", name, appId.Application, appId.Environment, appId.Family)
}

func dialerSrv(logger mon.Logger, settings *Settings) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, _ string, _ string) (net.Conn, error) {
		address := settings.Address

		if address == "" {
			address = srvNamingStrategy(settings.AppId, settings.Name)
			logger.Infof("no address provided for redis %s: using %s", settings.Name, address)
		}

		_, srvs, err := net.LookupSRV("", "", address)

		if err != nil {
			return nil, fmt.Errorf("can't lookup srv query for address %s: %w", address, err)
		}

		if len(srvs) != 1 {
			return nil, fmt.Errorf("redis instance count mismatch. there should be exactly one redis instance, found: %v", len(srvs))
		}

		address = fmt.Sprintf("%v:%v", srvs[0].Target, srvs[0].Port)
		logger.Infof("using address %s for redis %s", address, settings.Name)

		var d net.Dialer
		return d.DialContext(ctx, "tcp", address)
	}
}

func dialerTcp(logger mon.Logger, settings *Settings) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(_ context.Context, _ string, _ string) (net.Conn, error) {
		logger.Infof("using address %s for redis %s", settings.Address, settings.Name)

		return net.Dial("tcp", settings.Address)
	}
}
