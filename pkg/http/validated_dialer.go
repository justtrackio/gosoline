package http

import (
	"context"
	"fmt"
	"net"

	"github.com/justtrackio/gosoline/pkg/funk"
)

// fixedDialerEntry pins the ip addresses a host resolved to at validation time
// so the transport dials exactly those and never re-resolves the name (which
// would reopen a dns-rebinding / time-of-check-time-of-use window).
type fixedDialerEntry struct {
	host string
	ips  []net.IP
}

func (d *fixedDialerEntry) set(host string, addresses []net.IPAddr) {
	d.host = host
	d.ips = funk.Map(addresses, func(ip net.IPAddr) net.IP {
		return ip.IP
	})
}

type fixedDialerEntryKey struct{}

func withFixedDialerEntry(ctx context.Context, d *fixedDialerEntry) context.Context {
	return context.WithValue(ctx, fixedDialerEntryKey{}, d)
}

func getFixedDialerEntry(ctx context.Context) (*fixedDialerEntry, bool) {
	entry, ok := ctx.Value(fixedDialerEntryKey{}).(*fixedDialerEntry)

	return entry, ok
}

type dialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// dialValidated wraps the base dial function so that, whenever the request
// context carries a fixedDialerEntry, the pinned host is dialed using only the
// ip addresses it was pinned to at validation time. Any other host (e.g. a
// configured proxy) is dialed as usual.
func dialValidated(dial dialFunc) dialFunc {
	return func(ctx context.Context, network string, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return dial(ctx, network, addr)
		}

		entry, ok := getFixedDialerEntry(ctx)
		if !ok || entry.host != host || len(entry.ips) == 0 {
			return dial(ctx, network, addr)
		}

		var lastErr error
		for _, ip := range entry.ips {
			conn, err := dial(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}

			lastErr = err
		}

		return nil, fmt.Errorf("dialing the validated addresses of %q: %w", host, lastErr)
	}
}
