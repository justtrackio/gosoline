package httpserver

import (
	"context"
	"net"
	"sync"

	"github.com/justtrackio/gosoline/pkg/log"
)

type connectionLimitListener struct {
	net.Listener

	ctx     context.Context
	logger  log.Logger
	limit   chan struct{}
	manager ConnectionPressureManager
	closed  chan struct{}
	once    sync.Once
}

type limitedConn struct {
	net.Conn
	release func()
	once    sync.Once
}

func NewConnectionLimitListener(
	ctx context.Context,
	logger log.Logger,
	listener net.Listener,
	settings ConcurrencySettings,
	manager ConnectionPressureManager,
) net.Listener {
	if settings.MaxConnections <= 0 {
		return listener
	}

	return &connectionLimitListener{
		Listener: listener,
		ctx:      ctx,
		logger:   logger,
		limit:    make(chan struct{}, settings.MaxConnections),
		manager:  manager,
		closed:   make(chan struct{}),
	}
}

func (l *connectionLimitListener) Accept() (net.Conn, error) {
	if err := l.acquire(); err != nil {
		return nil, err
	}

	conn, err := l.Listener.Accept()
	if err != nil {
		l.release()

		return nil, err
	}

	return &limitedConn{
		Conn:    conn,
		release: l.release,
	}, nil
}

func (l *connectionLimitListener) acquire() error {
	select {
	case l.limit <- struct{}{}:
		return nil
	case <-l.closed:
		return net.ErrClosed
	default:
	}

	for {
		for {
			closedIdleConnection, err := l.manager.CloseOldestIdleConnection()
			if err != nil {
				l.logger.Warn(l.ctx, "failed to close idle http connection under connection pressure: %s", err)
			}

			if err != nil && closedIdleConnection {
				continue
			}

			break
		}

		select {
		case l.limit <- struct{}{}:
			return nil
		case <-l.manager.IdleConnectionAvailable():
		case <-l.closed:
			return net.ErrClosed
		}
	}
}

func (l *connectionLimitListener) Close() error {
	l.once.Do(func() {
		close(l.closed)
	})

	return l.Listener.Close()
}

func (l *connectionLimitListener) release() {
	<-l.limit
}

func (c *limitedConn) Close() error {
	defer c.once.Do(c.release)

	return c.Conn.Close()
}
