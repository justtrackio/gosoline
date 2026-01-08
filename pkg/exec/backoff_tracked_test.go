package exec_test

import (
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestTrackedBackOff_NextBackOff_WithDefaultTracker(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewDefaultElapsedTimeTrackerWithInterfaces(fakeClock)
	tracker.Start()

	settings := &exec.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxElapsedTime:  5 * time.Second,
	}

	bo := exec.NewTrackedBackOff(settings, tracker)

	// First backoff should return an interval
	interval := bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval)
	assert.GreaterOrEqual(t, interval, 100*time.Millisecond)

	// Advance time past max elapsed
	fakeClock.Advance(6 * time.Second)

	// Now it should stop
	interval = bo.NextBackOff()
	assert.Equal(t, backoff.Stop, interval)
}

func TestTrackedBackOff_NextBackOff_WithErrorTriggeredTracker(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)
	tracker.Start()

	settings := &exec.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxElapsedTime:  5 * time.Second,
	}

	bo := exec.NewTrackedBackOff(settings, tracker)

	// Simulate blocking for 10 seconds before first error (e.g., Kafka poll)
	fakeClock.Advance(10 * time.Second)

	// No error yet, so elapsed should be 0 - should NOT stop
	interval := bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval, "should not stop when no error has occurred yet")

	// Now an error occurs
	tracker.OnError(assert.AnError)

	// Should still be able to get intervals (we just started the error clock)
	interval = bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval, "should not stop immediately after first error")

	// Advance 3 seconds (still within budget)
	fakeClock.Advance(3 * time.Second)
	interval = bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval, "should not stop within max elapsed time")

	// Advance past max elapsed time from first error
	fakeClock.Advance(3 * time.Second) // total 6s since error
	interval = bo.NextBackOff()
	assert.Equal(t, backoff.Stop, interval, "should stop after max elapsed time since first error")
}

func TestTrackedBackOff_NextBackOff_ErrorTriggeredTracker_ResetOnSuccess(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)
	tracker.Start()

	settings := &exec.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxElapsedTime:  5 * time.Second,
	}

	bo := exec.NewTrackedBackOff(settings, tracker)

	// Error occurs
	tracker.OnError(assert.AnError)
	fakeClock.Advance(3 * time.Second)

	interval := bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval)

	// Success resets the error clock
	tracker.OnSuccess()
	bo.Reset()

	// Even after 10 more seconds, we should not stop (no error active)
	fakeClock.Advance(10 * time.Second)
	interval = bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval, "should not stop after success reset")

	// New error occurs
	tracker.OnError(assert.AnError)

	// New budget starts from this error
	fakeClock.Advance(3 * time.Second)
	interval = bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval, "should not stop within new budget")

	fakeClock.Advance(3 * time.Second) // 6s since new error
	interval = bo.NextBackOff()
	assert.Equal(t, backoff.Stop, interval, "should stop after exceeding new budget")
}

func TestTrackedBackOff_NextBackOff_NoMaxElapsedTime(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewDefaultElapsedTimeTrackerWithInterfaces(fakeClock)
	tracker.Start()

	settings := &exec.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxElapsedTime:  0, // disabled
	}

	bo := exec.NewTrackedBackOff(settings, tracker)

	// Even after a very long time, should not stop
	fakeClock.Advance(24 * time.Hour)

	interval := bo.NextBackOff()
	assert.NotEqual(t, backoff.Stop, interval, "should never stop when MaxElapsedTime is 0")
}
