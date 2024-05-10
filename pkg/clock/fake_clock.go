package clock

import (
	"sync"
	"time"
)

// A FakeClock provides the functionality of a Clock with the added functionality to Advance said Clock and block until
// at least a given number of timers, tickers, or channels (Clock.After and Clock.Sleep) wait for the time to Advance.
//
//go:generate go run github.com/vektra/mockery/v2 --name FakeClock
type FakeClock interface {
	Clock
	// Advance advances the FakeClock to a new point in time as well as any tickers and timers created from it.
	Advance(d time.Duration)
	// BlockUntil will block until the FakeClock has the given number of calls to Clock.Sleep or Clock.After.
	BlockUntil(n int)
	// BlockUntilTimers will block until the FakeClock has at least the given number of timers created (similar to BlockUntil).
	// Only timers which are currently not stopped or expired are counted.
	BlockUntilTimers(n int)
	// BlockUntilTickers will block until the FakeClock has at least the given number of tickers created (similar to BlockUntil).
	// Only tickers which are currently not stopped are counted (expired tickers are still counted, they will trigger
	// again after more time passes).
	BlockUntilTickers(n int)
}

type fakeClock struct {
	now              time.Time
	lck              sync.RWMutex
	sleepers         []*fakeSleeper
	timers           []*fakeTimer
	tickers          []*fakeTicker
	blockOnSleepers  blockerMap
	blockOnTimers    blockerMap
	blockOnTickers   blockerMap
	nonBlockingSleep bool
}

type blockerMap map[int][]chan struct{}

type fakeSleeper struct {
	c         chan time.Time
	remaining time.Duration
}

// NewFakeClock creates a new FakeClock at a non-zero and fixed date.
func NewFakeClock(options ...FakeClockOption) FakeClock {
	return NewFakeClockAt(time.Date(1984, time.April, 4, 0, 0, 0, 0, time.UTC), options...)
}

// NewFakeClockAt creates a new FakeClock at the given time.Time.
func NewFakeClockAt(t time.Time, options ...FakeClockOption) FakeClock {
	clock := &fakeClock{
		now:             t,
		blockOnSleepers: blockerMap{},
		blockOnTickers:  blockerMap{},
		blockOnTimers:   blockerMap{},
	}

	for _, opt := range options {
		opt(clock)
	}

	return clock
}

func (f *fakeClock) Advance(d time.Duration) {
	f.lck.Lock()
	defer f.lck.Unlock()

	f.now = f.now.Add(d)

	newSleepers := make([]*fakeSleeper, 0, len(f.sleepers))
	for _, sleeper := range f.sleepers {
		if keep := sleeper.advance(f.now, d); keep {
			newSleepers = append(newSleepers, sleeper)
		}
	}
	f.sleepers = newSleepers

	for _, timer := range f.timers {
		timer.advance(f.now, d)
	}

	for _, ticker := range f.tickers {
		ticker.advance(f.now, d)
	}
}

func (f *fakeSleeper) advance(t time.Time, d time.Duration) bool {
	if f.remaining > d {
		f.remaining -= d

		return true
	}

	if f.remaining == 0 {
		return false
	}

	f.remaining = 0
	f.c <- t

	return false
}

func (f *fakeClock) Now() time.Time {
	f.lck.RLock()
	defer f.lck.RUnlock()

	return f.now
}

func (f *fakeClock) Since(t time.Time) time.Duration {
	return f.Now().Sub(t)
}

func (f *fakeClock) After(d time.Duration) <-chan time.Time {
	f.lck.Lock()
	defer f.lck.Unlock()

	sleeper := &fakeSleeper{
		// important: the channel needs to be buffered with at least one element capacity, otherwise you can't advance
		// the clock and read the channel in the same thread without hanging.
		c:         make(chan time.Time, 1),
		remaining: d,
	}
	f.sleepers = append(f.sleepers, sleeper)
	f.blockOnSleepers = f.notifyBlockers(f.blockOnSleepers, f.waitingSleepers())

	if d <= 0 {
		// if someone calls After with an expiry of <= 0, trigger it immediately
		sleeper.c <- f.now
	}

	return sleeper.c
}

func (f *fakeClock) Sleep(d time.Duration) {
	if f.nonBlockingSleep {
		f.Advance(d)

		return
	}

	<-f.After(d)
}

func (f *fakeClock) BlockUntil(n int) {
	f.lck.Lock()

	if f.waitingSleepers() >= n {
		f.lck.Unlock()

		return
	}

	ch := make(chan struct{})
	f.blockOnSleepers[n] = append(f.blockOnSleepers[n], ch)

	f.lck.Unlock()

	<-ch
}

func (f *fakeClock) BlockUntilTimers(n int) {
	f.lck.Lock()

	if f.waitingTimers() >= n {
		f.lck.Unlock()

		return
	}

	ch := make(chan struct{})
	f.blockOnTimers[n] = append(f.blockOnTimers[n], ch)

	f.lck.Unlock()

	<-ch
}

func (f *fakeClock) BlockUntilTickers(n int) {
	f.lck.Lock()

	if f.waitingTickers() >= n {
		f.lck.Unlock()

		return
	}

	ch := make(chan struct{})
	f.blockOnTickers[n] = append(f.blockOnTickers[n], ch)

	f.lck.Unlock()

	<-ch
}

func (*fakeClock) notifyBlockers(blockers blockerMap, waiting int) blockerMap {
	result := make(blockerMap, len(blockers))

	for waitingFor, waitingChannels := range blockers {
		if waitingFor <= waiting {
			for _, ch := range waitingChannels {
				close(ch)
			}
		} else {
			result[waitingFor] = waitingChannels
		}
	}

	return result
}

func (f *fakeClock) waitingSleepers() int {
	result := 0

	for _, sleeper := range f.sleepers {
		if sleeper.remaining > 0 {
			result++
		}
	}

	return result
}

func (f *fakeClock) waitingTimers() int {
	result := 0

	for _, timer := range f.timers {
		timer.lck.Lock()
		if timer.remaining > 0 {
			result++
		}
		timer.lck.Unlock()
	}

	return result
}

func (f *fakeClock) waitingTickers() int {
	result := 0

	for _, ticker := range f.tickers {
		ticker.lck.Lock()
		if ticker.duration > 0 {
			result++
		}
		ticker.lck.Unlock()
	}

	return result
}
