package clock

import (
	"sync/atomic"
	"time"
)

type Ticker interface {
	Stop()
	Reset()
	Tick() <-chan time.Time
}

type TickerFactory func(duration time.Duration) Ticker

type realTicker struct {
	ticks            chan time.Time
	confirm          chan struct{}
	commands         chan int
	runningProcesses *int32
}

const (
	cmdReset = iota
	cmdStop
)

// Create a new thread-safe ticker with the given tick duration.
// You can call Ticker.Reset to remove any pending tick (if any)
// and reset the next tick to only occur duration after the call
// to Ticker.Reset.
//
// Example:
//  ticker := NewRealTicker(time.Minute)
//  time.Sleep(time.Second * 30)
//  ticker.Reset()
//  tick := <- ticker.Tick()
//  // now at least 90 seconds have passed
//  time.Sleep(time.Second * 90)
//  ticker.Reset()
//  tick <- ticker.Tick()
//  // now at least an additional 150 seconds have passed
func NewRealTicker(duration time.Duration) Ticker {
	t := &realTicker{
		ticks:            make(chan time.Time),
		confirm:          make(chan struct{}),
		commands:         make(chan int),
		runningProcesses: new(int32),
	}

	go t.run(duration)

	return t
}

// can the ticker ever produce another tick?
func (t *realTicker) IsStopped() bool {
	return atomic.LoadInt32(t.runningProcesses) == 0
}

func (t *realTicker) run(duration time.Duration) {
	atomic.AddInt32(t.runningProcesses, 1)
	defer atomic.AddInt32(t.runningProcesses, -1)

	ticker := time.NewTicker(duration)

	for {
		select {
		case tick := <-ticker.C:
			if shouldUseUTC() {
				tick = tick.UTC()
			}
			stopped := t.writeTick(tick, duration, &ticker)
			if stopped {
				return
			}
		case cmd := <-t.commands:
			switch cmd {
			case cmdReset:
				ticker.Stop()
				ticker = time.NewTicker(duration)

			case cmdStop:
				ticker.Stop()
				return
			}
		}
	}
}

func (t *realTicker) writeTick(tick time.Time, duration time.Duration, ticker **time.Ticker) bool {
	go func() {
		atomic.AddInt32(t.runningProcesses, 1)
		defer atomic.AddInt32(t.runningProcesses, -1)

		t.ticks <- tick
		t.confirm <- struct{}{}
	}()

	select {
	case <-t.confirm:
		return false
	case cmd := <-t.commands:
		switch cmd {
		case cmdReset:
			// stop the ticker for now, we don't need it until we dealt with the tick
			(*ticker).Stop()
			// we might be in the process of writing a tick, but are told to reset before anyone managed to read the tick (maybe)
			// in that case, try to read the tick or the confirmation to potentially undo our work
			select {
			case <-t.ticks:
				// the tick was written, consume the confirm
				<-t.confirm
			case <-t.confirm:
				// someone else read the tick already, but we consumed the confirmation, so all is good
			}

			// we are in a clean state again, create a fresh ticker to start anew
			*ticker = time.NewTicker(duration)

		case cmdStop:
			// stop the ticker for now, we don't need it until we dealt with the tick
			(*ticker).Stop()
			// clean the tick if needed
			select {
			case <-t.ticks:
				<-t.confirm
			case <-t.confirm:
			}

			return true

		}

		return false
	}
}

func (t *realTicker) Reset() {
	t.commands <- cmdReset
}

func (t *realTicker) Stop() {
	t.commands <- cmdStop
}

func (t *realTicker) Tick() <-chan time.Time {
	return t.ticks
}

type FakeTicker struct {
	ch chan time.Time
}

func NewFakeTicker() *FakeTicker {
	return &FakeTicker{
		ch: make(chan time.Time),
	}
}

func (f *FakeTicker) Stop() {
}

func (f *FakeTicker) Tick() <-chan time.Time {
	return f.ch
}

func (f *FakeTicker) Reset() {
}

func (f *FakeTicker) Trigger(time time.Time) {
	f.ch <- time
}
