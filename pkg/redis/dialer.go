package redis

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	DialerSrv = "srv"
	DialerTcp = "tcp"
)

var dialers = map[string]Dialer{
	DialerSrv: dialerSrv,
	DialerTcp: dialerTcp,
}

type (
	Dialer           func(logger log.Logger, settings *Settings) func(context.Context, string, string) (net.Conn, error)
	SrvNamingFactory func(appId cfg.AppId, name string) string
)

func dialerSrv(logger log.Logger, settings *Settings) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, _ string, _ string) (net.Conn, error) {
		address := settings.Address

		if address == "" {
			values := map[string]string{
				"project": settings.AppId.Project,
				"env":     settings.AppId.Environment,
				"family":  settings.AppId.Family,
				"group":   settings.AppId.Group,
				"app":     settings.AppId.Application,
				"name":    settings.Name,
			}

			address = settings.Naming.Pattern
			for key, val := range values {
				templ := fmt.Sprintf("{%s}", key)
				address = strings.ReplaceAll(address, templ, val)
			}

			logger.Info("no address provided for redis %s: using %s", settings.Name, address)
		}

		_, srvs, err := net.LookupSRV("", "", address)
		if err != nil {
			return nil, fmt.Errorf("can't lookup srv query for address %s: %w", address, err)
		}

		if len(srvs) != 1 {
			return nil, fmt.Errorf("redis instance count mismatch. there should be exactly one redis instance, found: %v", len(srvs))
		}

		address = fmt.Sprintf("%v:%v", srvs[0].Target, srvs[0].Port)
		logger.Info("using address %s for redis %s", address, settings.Name)

		var d net.Dialer
		return d.DialContext(ctx, "tcp", address)
	}
}

func dialerTcp(logger log.Logger, settings *Settings) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(_ context.Context, _ string, _ string) (net.Conn, error) {
		logger.Info("using address %s for redis %s", settings.Address, settings.Name)

		return net.Dial("tcp", settings.Address)
	}
}
