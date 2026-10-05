package metric

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaemonThrottleErrorExpiresAfterTicker(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	daemon := newThrottleTestDaemon(fakeClock)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	assert.True(t, daemon.throttleError(ctx, "invalid datum"))
	assert.False(t, daemon.throttleError(ctx, "invalid datum"))

	fakeClock.BlockUntilTickers(2)
	fakeClock.Advance(time.Minute)

	require.Eventually(t, func() bool {
		return daemon.throttleError(ctx, "invalid datum")
	}, time.Second, time.Millisecond)
}

func TestDaemonThrottleErrorReleasesOnContextCancellation(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	daemon := newThrottleTestDaemon(fakeClock)
	ctx, cancel := context.WithCancel(t.Context())

	assert.True(t, daemon.throttleError(ctx, "invalid datum"))
	fakeClock.BlockUntilTickers(2)
	cancel()

	nextCtx, nextCancel := context.WithCancel(t.Context())
	t.Cleanup(nextCancel)
	require.Eventually(t, func() bool {
		return daemon.throttleError(nextCtx, "invalid datum")
	}, time.Second, time.Millisecond)
}

func newThrottleTestDaemon(clk clock.Clock) *Daemon {
	return newMetricDaemonWithInterfaces(log.NewCliLogger(), nil, nil, nil, &Settings{
		Interval: time.Minute,
	}, clk)
}
