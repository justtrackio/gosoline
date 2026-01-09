package exec_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestDefaultElapsedTimeTracker_MeasuresFromStart(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewDefaultElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	fakeClock.Advance(5 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, 5*time.Second, elapsed)
}

func TestDefaultElapsedTimeTracker_OnErrorDoesNotAffectElapsed(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewDefaultElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	fakeClock.Advance(3 * time.Second)
	tracker.OnError(assert.AnError)
	fakeClock.Advance(2 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, 5*time.Second, elapsed)
}

func TestDefaultElapsedTimeTracker_OnSuccessDoesNotAffectElapsed(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewDefaultElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	fakeClock.Advance(3 * time.Second)
	tracker.OnSuccess()
	fakeClock.Advance(2 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, 5*time.Second, elapsed)
}

func TestErrorTriggeredElapsedTimeTracker_ReturnsZeroBeforeError(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	fakeClock.Advance(10 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, time.Duration(0), elapsed)
}

func TestErrorTriggeredElapsedTimeTracker_MeasuresFromFirstError(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	fakeClock.Advance(10 * time.Second) // Blocking time - should not count
	tracker.OnError(assert.AnError)
	fakeClock.Advance(3 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, 3*time.Second, elapsed)
}

func TestErrorTriggeredElapsedTimeTracker_SecondErrorDoesNotResetTimer(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	tracker.OnError(assert.AnError)
	fakeClock.Advance(2 * time.Second)
	tracker.OnError(assert.AnError) // Second error should not reset
	fakeClock.Advance(3 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, 5*time.Second, elapsed)
}

func TestErrorTriggeredElapsedTimeTracker_SuccessResetsTimer(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	tracker.OnError(assert.AnError)
	fakeClock.Advance(5 * time.Second)
	tracker.OnSuccess() // Should reset
	fakeClock.Advance(10 * time.Second)

	// After success, elapsed should be zero again
	elapsed := tracker.Elapsed()
	assert.Equal(t, time.Duration(0), elapsed)
}

func TestErrorTriggeredElapsedTimeTracker_ErrorAfterSuccessStartsFresh(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	tracker.OnError(assert.AnError)
	fakeClock.Advance(5 * time.Second)
	tracker.OnSuccess() // Reset
	fakeClock.Advance(10 * time.Second)
	tracker.OnError(assert.AnError) // New error
	fakeClock.Advance(2 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, 2*time.Second, elapsed)
}

func TestErrorTriggeredElapsedTimeTracker_StartResetsState(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tracker := exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(fakeClock)

	tracker.Start()
	tracker.OnError(assert.AnError)
	fakeClock.Advance(5 * time.Second)

	// Start again should reset everything
	tracker.Start()
	fakeClock.Advance(10 * time.Second)

	elapsed := tracker.Elapsed()
	assert.Equal(t, time.Duration(0), elapsed)
}
