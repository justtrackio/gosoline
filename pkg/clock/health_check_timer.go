package clock

import (
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

	return h.clock.Since(lastMark) <= h.timeout
}

func (h *healthCheckTimer) MarkHealthy() {
	h.lastMarkMilli.Store(h.clock.Now().UnixMilli())
}
