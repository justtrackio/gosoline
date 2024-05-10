package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cache"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	advisorMaxSize   = 65535
	advisorPruneSize = 1000
)

//go:generate go run github.com/vektra/mockery/v2 --name ConnectionLifeCycleAdvisor
type (
	ConnectionLifeCycleAdvisor interface {
		// ShouldCloseConnection checks whether the connection to the remote address should be closed.
		ShouldCloseConnection(remoteAddr string, headers http.Header) bool
	}

	noopConnectionLifeCycleAdvisor struct{}

	connectionLifeCycleAdvisor struct {
		clock    clock.Clock
		settings ConnectionLifeCycleAdvisorSettings
		tracker  cache.Cache[trafficEntry]
	}

	trafficEntry struct {
		requestCount int
		activeSince  time.Time
	}

	connectionLifeCycleKey string
)

// ProvideConnectionLifeCycleAdvisor returns a ConnectionLifeCycleAdvisor.
// This is a component that tracks the lifecycle of client connections based on configurable policies.
// On that basis it provides recommendations on whether a connection should be closed or not.
// This is useful to prevent clients from keeping connections open, eg in k8s where load balancing only happens on new connections.
func ProvideConnectionLifeCycleAdvisor(ctx context.Context, config cfg.Config, _ log.Logger, serverName string) (ConnectionLifeCycleAdvisor, error) {
	return appctx.Provide(ctx, connectionLifeCycleKey(serverName), func() (ConnectionLifeCycleAdvisor, error) {
		return NewConnectionLifeCycleAdvisor(config, serverName)
	})
}

func NewConnectionLifeCycleAdvisor(config cfg.Config, serverName string) (ConnectionLifeCycleAdvisor, error) {
	settings := ConnectionLifeCycleAdvisorSettings{}
	key := HttpserverSettingsKey(serverName) + ".connection_lifecycle"
	if err := config.UnmarshalKey(key, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connection lifecycle advisor settings: %w", err)
	}

	return NewConnectionLifeCycleAdvisorWithInterfaces(clock.Provider, settings), nil
}

func NewConnectionLifeCycleAdvisorWithInterfaces(
	providedClock clock.Clock,
	settings ConnectionLifeCycleAdvisorSettings,
) ConnectionLifeCycleAdvisor {
	ttl := 2 * settings.MaxConnectionAge
	if settings.MaxConnectionAge <= 0 {
		ttl = 24 * time.Hour
	}

	if !settings.Enabled {
		return noopConnectionLifeCycleAdvisor{}
	}

	return &connectionLifeCycleAdvisor{
		clock:    providedClock,
		settings: settings,
		tracker:  cache.New[trafficEntry](advisorMaxSize, advisorPruneSize, ttl),
	}
}

func (traffic connectionLifeCycleAdvisor) ShouldCloseConnection(remoteAddr string, _ http.Header) bool {
	shouldBeClosed := false

	if remoteAddr == "" {
		return false
	}

	traffic.tracker.Mutate(remoteAddr, func(entry *trafficEntry) trafficEntry {
		if entry == nil {
			entry = &trafficEntry{
				activeSince: traffic.clock.Now(),
			}
		}
		entry.requestCount++

		shouldBeClosed = entry.ShouldClose(traffic.settings.MaxConnectionAge, traffic.settings.MaxConnectionRequestCount, traffic.clock.Now())

		return *entry
	})

	if shouldBeClosed {
		traffic.tracker.Delete(remoteAddr)
	}

	return shouldBeClosed
}

func (entry trafficEntry) ShouldClose(maxAge time.Duration, maxRequestCount int, instant time.Time) bool {
	if maxRequestCount > 0 && entry.requestCount >= maxRequestCount {
		return true
	}

	return maxAge > 0 && entry.activeSince.Add(maxAge).Before(instant)
}

func (noopConnectionLifeCycleAdvisor) ShouldCloseConnection(_ string, _ http.Header) bool {
	return false
}

// ProvideConnectionLifeCycleInterceptor provides a ConnectionLifeCycleAdvisorInterceptor that
// controls closing of connections based on the ConnectionLifeCycleAdvisor.
func ProvideConnectionLifeCycleInterceptor(ctx context.Context, config cfg.Config, logger log.Logger, serverName string) (gin.HandlerFunc, error) {
	return appctx.Provide(ctx, connectionLifeCycleKey(serverName+"-interceptor"), func() (gin.HandlerFunc, error) {
		connectionLifeCycleAdvisor, err := ProvideConnectionLifeCycleAdvisor(ctx, config, logger, serverName)
		if err != nil {
			return nil, err
		}

		return NewConnectionLifeCycleInterceptor(connectionLifeCycleAdvisor), nil
	})
}

// NewConnectionLifeCycleInterceptor creates a gin.HandlerFunc that uses the ConnectionLifeCycleAdvisor to
// determine whether the connection to the remote address should be closed.
func NewConnectionLifeCycleInterceptor(connectionLifeCycleAdvisor ConnectionLifeCycleAdvisor) gin.HandlerFunc {
	return func(c *gin.Context) {
		remoteAddr := c.Request.RemoteAddr
		if connectionLifeCycleAdvisor.ShouldCloseConnection(remoteAddr, c.Request.Header) {
			// This works for both HTTP/1.1 and HTTP/2 connections.
			// see: https://github.com/golang/go/issues/20977
			c.Header("Connection", "close")
		}

		c.Next()
	}
}
