package clock

import (
	"errors"
	"sync"
	"time"
)

type fakeTicker struct {
	clock     *fakeClock
	c         chan time.Time
	lck       sync.Mutex
	remaining time.Duration
	duration  time.Duration
}

func (f *fakeClock) NewTicker(d time.Duration) Ticker {
	// replicate the panic from time.NewTicker
	if d <= 0 {
		panic(errors.New("non-positive interval for NewTicker"))
	}

	f.lck.Lock()
	defer f.lck.Unlock()

	ticker := &fakeTicker{
		clock: f,
		// important: the channel needs to be buffered with at least one element capacity, otherwise you can't advance
		// the clock and read the channel in the same thread without hanging.
		c:         make(chan time.Time, 1),
		remaining: d,
		duration:  d,
	}
	f.tickers = append(f.tickers, ticker)
	f.blockOnTickers = f.notifyBlockers(f.blockOnTickers, f.waitingTickers())

	return ticker
}

func (f *fakeTicker) Stop() {
	f.lck.Lock()
	defer f.lck.Unlock()

	f.remaining = 0
	// mark ticker as disabled
	f.duration = 0
}

func (f *fakeTicker) Chan() <-chan time.Time {
	return f.c
}

func (f *fakeTicker) Reset(d time.Duration) {
	if d <= 0 {
		panic(errors.New("non-positive interval for Reset"))
	}

	f.lck.Lock()
	f.remaining = d
	f.duration = d
	f.lck.Unlock()

	f.clock.lck.Lock()
	defer f.clock.lck.Unlock()

	f.clock.blockOnTickers = f.clock.notifyBlockers(f.clock.blockOnTickers, f.clock.waitingTickers())
}

func (f *fakeTicker) advance(t time.Time, d time.Duration) {
	f.lck.Lock()
	defer f.lck.Unlock()

	if f.duration == 0 {
		// ticker was stopped
		return
	}

	if f.remaining > d {
		f.remaining -= d
		return
	}

	f.remaining = f.duration

	// similar to a real timer, empty the output channel before writing the new value to the channel (to avoid hanging
	// the current go routine should the ticker expire a second time before it was read)
	select {
	case <-f.c:
	default:
	}

	f.c <- t
}
