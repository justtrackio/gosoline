package clock

import (
	"time"
)

// A Clock provides the most commonly needed functions from the time package while allowing you to substitute them for unit
// and integration tests.
//
//go:generate mockery --name Clock
type Clock interface {
	// After waits for the duration to elapse and then sends the current time on the returned channel.
	// It is equivalent to NewTimer(d).Chan().
	// The underlying Timer is not recovered by the garbage collector until the timer fires.
	// If efficiency is a concern, use NewTimer instead and call Timer.Stop if the timer is no longer needed.
	//
	// If you enabled UTC, the timestamps are converted to UTC before you receive them.
	After(d time.Duration) <-chan time.Time
	// NewTicker returns a new Ticker containing a channel that will send the time on the channel after each tick.
	// The period of the ticks is specified by the duration argument. The ticker will drop ticks for slow receivers.
	// The duration d must be greater than zero; if not, NewTicker will panic. Stop the ticker to release associated resources.
	//
	// If you enabled UTC, the timestamps are converted to UTC before you receive them.
	NewTicker(d time.Duration) Ticker
	// NewTimer creates a new Timer that will send the current time on its channel after at least duration d. If you specify
	// a duration less than or equal to 0, the timer will fire immediately.
	//
	// If you enabled UTC, the timestamps are converted to UTC before you receive them.
	NewTimer(d time.Duration) Timer
	// Now will return the current time either in local time or UTC, if you enabled that.
	Now() time.Time
	// Since returns the time which passed since t.
	Since(t time.Time) time.Duration
	// Sleep blocks execution of your go routine for at least the given duration.
	Sleep(d time.Duration)
}

type realClock struct{}

// NewRealClock creates a new Clock which uses the current system time. If you enabled UTC, all timestamps will be converted
// to UTC before they are returned. Use clock.Provider instead if you need the ability to replace the clock your code uses.
func NewRealClock() Clock {
	return realClock{}
}

func (c realClock) After(d time.Duration) <-chan time.Time {
	return c.NewTimer(d).Chan()
}

func (c realClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (c realClock) Now() time.Time {
	if shouldUseUTC() {
		return time.Now().UTC()
	}

	return time.Now()
}

func (c realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c realClock) NewTimer(d time.Duration) Timer {
	return NewRealTimer(d)
}

func (c realClock) NewTicker(d time.Duration) Ticker {
	return NewRealTicker(d)
}
