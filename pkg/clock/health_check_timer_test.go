package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
)

func TestHealthcheckTimer(t *testing.T) {
	c := clock.NewFakeClock()
	timer := clock.NewHealthCheckTimerWithInterfaces(c, time.Minute)
	assert.True(t, timer.IsHealthy(), "timer should be initially healthy")

	c.Advance(time.Minute)
	assert.True(t, timer.IsHealthy(), "timer should be healthy at the edge of the timeout")

	c.Advance(time.Millisecond)
	assert.False(t, timer.IsHealthy(), "timer should not be healthy after the timeout")

	timer.MarkHealthy()
	assert.True(t, timer.IsHealthy(), "timer should be healthy after being marked as healthy")

	c.Advance(time.Minute + time.Millisecond)
	assert.False(t, timer.IsHealthy(), "timer should not be healthy after the timeout again")
}
