package exec

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// TrackedBackOff wraps an ExponentialBackOff and uses an ElapsedTimeTracker
// to determine when MaxElapsedTime has been exceeded instead of the internal clock.
// This allows the elapsed time measurement strategy to be customized (e.g., measure
// from when the first error occurs rather than from when the backoff was created).
type TrackedBackOff struct {
	backOff        *backoff.ExponentialBackOff
	tracker        ElapsedTimeTracker
	maxElapsedTime time.Duration
}

// NewTrackedBackOff creates a backoff that delegates elapsed time checking to the provided tracker.
// The underlying ExponentialBackOff is configured with MaxElapsedTime=0 to disable its internal
// elapsed time check, allowing the TrackedBackOff to handle it via the tracker.
func NewTrackedBackOff(settings *BackoffSettings, tracker ElapsedTimeTracker) *TrackedBackOff {
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = settings.InitialInterval
	backoffConfig.MaxInterval = settings.MaxInterval
	// Disable the internal elapsed time check - we handle it ourselves
	backoffConfig.MaxElapsedTime = 0

	return &TrackedBackOff{
		backOff:        backoffConfig,
		tracker:        tracker,
		maxElapsedTime: settings.MaxElapsedTime,
	}
}

// NextBackOff returns the next backoff interval, or backoff.Stop if MaxElapsedTime
// (as measured by the tracker) has been exceeded.
func (b *TrackedBackOff) NextBackOff() time.Duration {
	// Check if we've exceeded the max elapsed time using our tracker
	if b.maxElapsedTime > 0 && b.tracker.Elapsed() > b.maxElapsedTime {
		return backoff.Stop
	}

	return b.backOff.NextBackOff()
}

// Reset resets the backoff interval back to the initial value.
func (b *TrackedBackOff) Reset() {
	b.backOff.Reset()
}
