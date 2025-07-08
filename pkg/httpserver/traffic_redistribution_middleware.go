package httpserver

import (
	"context"
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
	distributorMaxSize   int64  = 10000
	distributorPruneSize uint32 = 1000
)

//go:generate mockery --name TrafficDistributor
type (
	TrafficDistributor interface {
		// ShouldCloseConnection checks whether the connection to the remote address should be closed (after handling any currently in flight requests).
		ShouldCloseConnection(remoteAddr string, headers http.Header) bool
	}

	noopDistributor struct{}

	distributor struct {
		clock    clock.Clock
		settings TrafficDistributorSettings
		tracker  cache.Cache[trafficEntry]
	}

	trafficEntry struct {
		requestCount int
		activeSince  time.Time
	}

	trafficDistributionKey string
)

// ProvideTrafficDistributor returns a TrafficDistributor.
// This is a component that tracks the lifecycle of client connections
// based on configurable policies. On that basis it provides recommendations on whether a connection
// should be closed or not. This is useful to prevent clients from keeping connections open, eg in k8s
// where load balancing only happens on new connections.
func ProvideTrafficDistributor(ctx context.Context, config cfg.Config, _ log.Logger) (TrafficDistributor, error) {
	return appctx.Provide(ctx, trafficDistributionKey("TrafficDistributor"), func() (TrafficDistributor, error) {
		return NewTrafficDistributor(config)
	})
}

func NewTrafficDistributor(config cfg.Config) (TrafficDistributor, error) {
	settings := &TrafficDistributorSettings{}
	config.UnmarshalKey("trafficDistributor", settings)

	if !settings.Enabled {
		return noopDistributor{}, nil
	}

	return NewTrafficDistributorWithInterfaces(clock.Provider, *settings), nil
}

func NewTrafficDistributorWithInterfaces(
	providedClock clock.Clock,
	settings TrafficDistributorSettings,
) TrafficDistributor {
	ttl := 2 * settings.MaxConnectionAge
	if settings.MaxConnectionAge <= 0 {
		ttl = 24 * time.Hour
	}

	return &distributor{
		clock:    providedClock,
		settings: settings,
		tracker:  cache.New[trafficEntry](distributorMaxSize, distributorPruneSize, ttl),
	}
}

func (traffic distributor) ShouldCloseConnection(remoteAddr string, _ http.Header) bool {
	shouldBeClosed := false

	if remoteAddr == "" {
		return false
	}

	// this is not always a mutation. we are using the fact that Mutate does not store values that
	// are their type's zero value if the notFoundTtl is not set. this aborts the "transaction" if we don't
	// need or want a modification of the entry.
	traffic.tracker.Mutate(remoteAddr, func(entry *trafficEntry) trafficEntry {
		if entry == nil {
			return trafficEntry{
				requestCount: 1,
				activeSince:  traffic.clock.Now(),
			}
		}

		shouldBeClosed = entry.ShouldClose(traffic.settings.MaxConnectionAge, traffic.settings.MaxConnectionRequestCount, traffic.clock.Now())

		entry.requestCount++

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

func (noopDistributor) ShouldCloseConnection(_ string, _ http.Header) bool {
	return false
}

// ProvideTrafficDistributionInterceptor provides a TrafficDistributorInterceptor that
// controls closing of connections based on the traffic distributor.
func ProvideTrafficDistributionInterceptor(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	return appctx.Provide(ctx, trafficDistributionKey("TrafficDistributionInterceptor"), func() (gin.HandlerFunc, error) {
		distributor, err := ProvideTrafficDistributor(ctx, config, logger)
		if err != nil {
			return nil, err
		}

		return NewTrafficDistributionInterceptor(distributor), nil
	})
}

// NewTrafficDistributionInterceptor creates a gin.HandlerFunc that uses the TrafficDistributor to
// determine whether the connection to the remote address should be closed.
func NewTrafficDistributionInterceptor(distributor TrafficDistributor) gin.HandlerFunc {
	return func(c *gin.Context) {
		remoteAddr := c.Request.RemoteAddr
		if distributor.ShouldCloseConnection(remoteAddr, c.Request.Header) {
			// This works for both HTTP/1.1 and HTTP/2 connections.
			// see: https://github.com/golang/go/issues/20977
			c.Header("Connection", "close")
		}

		c.Next()
	}
}
