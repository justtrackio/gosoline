package clock

import (
	"fmt"
	"sync/atomic"
	"time"
)

type HealthCheckTimer interface {
	IsHealthy() bool
	MarkHealthy()
}

type healthCheckTimer struct {
	clock         Clock
	timeout       time.Duration
	lastMarkMilli atomic.Int64
}

// NewHealthCheckTimer creates a new HealthCheckTimer which turns unhealthy if not marked healthy before the timeout occurs.
func NewHealthCheckTimer(timeout time.Duration) (HealthCheckTimer, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("health check timeout must be a positive value, got %v", timeout)
	}

	return NewHealthCheckTimerWithInterfaces(Provider, timeout), nil
}

func NewHealthCheckTimerWithInterfaces(clock Clock, timeout time.Duration) HealthCheckTimer {
	t := &healthCheckTimer{
		clock:   clock,
		timeout: timeout,
	}

	// mark us as initially healthy
	t.MarkHealthy()

	return t
}

func (h *healthCheckTimer) IsHealthy() bool {
	lastMark := time.UnixMilli(h.lastMarkMilli.Load())
	lastMarkAge := h.clock.Since(lastMark)

	return lastMarkAge <= h.timeout
}

func (h *healthCheckTimer) MarkHealthy() {
	h.lastMarkMilli.Store(h.clock.Now().UnixMilli())
}
