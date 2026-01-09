package exec

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

// ElapsedTimeTracker defines the strategy for tracking elapsed time during backoff execution.
// Different implementations can measure time from different starting points.
type ElapsedTimeTracker interface {
	// Start is called at the beginning of execution.
	Start()
	// OnError is called when a retryable error occurs.
	OnError(err error)
	// OnSuccess is called when execution succeeds.
	OnSuccess()
	// Elapsed returns the current elapsed duration based on the tracking strategy.
	Elapsed() time.Duration
}

// DefaultElapsedTimeTracker measures time from when execution starts.
// This is the default behavior that preserves backward compatibility.
type DefaultElapsedTimeTracker struct {
	clock clock.Clock
	start time.Time
}

// NewDefaultElapsedTimeTracker creates a tracker that measures elapsed time from the start of execution.
func NewDefaultElapsedTimeTracker() *DefaultElapsedTimeTracker {
	return NewDefaultElapsedTimeTrackerWithInterfaces(clock.Provider)
}

// NewDefaultElapsedTimeTrackerWithInterfaces creates a DefaultElapsedTimeTracker with injected dependencies for testing.
func NewDefaultElapsedTimeTrackerWithInterfaces(clock clock.Clock) *DefaultElapsedTimeTracker {
	return &DefaultElapsedTimeTracker{
		clock: clock,
	}
}

func (t *DefaultElapsedTimeTracker) Start() {
	t.start = t.clock.Now()
}

func (t *DefaultElapsedTimeTracker) OnError(_ error) {
	// Default: no-op, we track from start
}

func (t *DefaultElapsedTimeTracker) OnSuccess() {
	// Default: no-op
}

func (t *DefaultElapsedTimeTracker) Elapsed() time.Duration {
	return t.clock.Since(t.start)
}

// ErrorTriggeredElapsedTimeTracker measures time from when the first error occurs.
// This is useful for long-blocking operations (like Kafka poll) where you want the
// MaxElapsedTime budget to only start being consumed when actual errors occur,
// not during normal blocking waits for data.
//
// On success, the error timer is reset so subsequent errors get a fresh budget.
type ErrorTriggeredElapsedTimeTracker struct {
	clock      clock.Clock
	errorStart time.Time
}

// NewErrorTriggeredElapsedTimeTracker creates a tracker that measures elapsed time from the first error.
func NewErrorTriggeredElapsedTimeTracker() *ErrorTriggeredElapsedTimeTracker {
	return NewErrorTriggeredElapsedTimeTrackerWithInterfaces(clock.Provider)
}

// NewErrorTriggeredElapsedTimeTrackerWithInterfaces creates an ErrorTriggeredElapsedTimeTracker with injected dependencies for testing.
func NewErrorTriggeredElapsedTimeTrackerWithInterfaces(clock clock.Clock) *ErrorTriggeredElapsedTimeTracker {
	return &ErrorTriggeredElapsedTimeTracker{
		clock: clock,
	}
}

func (t *ErrorTriggeredElapsedTimeTracker) Start() {
	t.errorStart = time.Time{}
}

func (t *ErrorTriggeredElapsedTimeTracker) OnError(_ error) {
	if t.errorStart.IsZero() {
		t.errorStart = t.clock.Now()
	}
}

func (t *ErrorTriggeredElapsedTimeTracker) OnSuccess() {
	// Reset error tracking on success - subsequent errors get a fresh budget
	t.errorStart = time.Time{}
}

func (t *ErrorTriggeredElapsedTimeTracker) Elapsed() time.Duration {
	if t.errorStart.IsZero() {
		return 0
	}

	return t.clock.Since(t.errorStart)
}
