package clock

import (
	"sync"
	"time"
)

type fakeTimer struct {
	clock     *fakeClock
	c         chan time.Time
	lck       sync.Mutex
	remaining time.Duration
}

func (f *fakeClock) NewTimer(d time.Duration) Timer {
	f.lck.Lock()
	defer f.lck.Unlock()

	timer := &fakeTimer{
		clock: f,
		// important: the channel needs to be buffered with at least one element capacity, otherwise you can't advance
		// the clock and read the channel in the same thread without hanging.
		c:         make(chan time.Time, 1),
		remaining: d,
	}
	f.timers = append(f.timers, timer)
	f.blockOnTimers = f.notifyBlockers(f.blockOnTimers, f.waitingTimers())

	if d <= 0 {
		// if someone creates a timer with an expiry of <= 0, trigger it immediately
		timer.sendTick(f.now)
	}

	return timer
}

func (f *fakeTimer) Chan() <-chan time.Time {
	return f.c
}

func (f *fakeTimer) Stop() bool {
	f.lck.Lock()
	defer f.lck.Unlock()

	oldRemaining := f.remaining
	f.remaining = 0

	return oldRemaining != 0
}

func (f *fakeTimer) Reset(d time.Duration) {
	f.lck.Lock()
	f.remaining = d
	f.lck.Unlock()

	f.clock.lck.Lock()
	defer f.clock.lck.Unlock()

	// if we are reset to <= 0, we have to trigger a tick immediately
	if d <= 0 {
		f.sendTick(f.clock.now)
	}

	f.clock.blockOnTimers = f.clock.notifyBlockers(f.clock.blockOnTimers, f.clock.waitingTimers())
}

func (f *fakeTimer) advance(t time.Time, d time.Duration) {
	f.lck.Lock()
	defer f.lck.Unlock()

	if f.remaining > d {
		f.remaining -= d

		return
	}

	if f.remaining == 0 {
		return
	}

	f.remaining = 0
	f.sendTick(t)
}

func (f *fakeTimer) sendTick(t time.Time) {
	// similar to a real timer, empty the output channel before writing the new value to the channel (to avoid hanging
	// the current go routine should the timer expire a second time before it was read)
	select {
	case <-f.c:
	default:
	}

	f.c <- t
}
