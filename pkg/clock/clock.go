package clock

import (
	"time"

	"github.com/jonboulle/clockwork"
)

//go:generate mockery --name Clock
type Clock interface {
	clockwork.Clock
}

type FakeClock interface {
	clockwork.FakeClock
}

func NewRealClock() Clock {
	return realClock{}
}

func NewFakeClock() FakeClock {
	return clockwork.NewFakeClock()
}

func NewFakeClockAt(t time.Time) FakeClock {
	return clockwork.NewFakeClockAt(t)
}

type realClock struct{}

func (c realClock) After(d time.Duration) <-chan time.Time {
	if shouldUseUTC() {
		c := time.After(d)
		// use a small buffered channel so our go routine can
		// terminate as soon as the timeout expires even if there
		// is no one receiving the time (anymore)
		utcChan := make(chan time.Time, 1)

		go func() {
			t := <-c
			utcChan <- t.UTC()
		}()

		return utcChan
	}

	return time.After(d)
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
