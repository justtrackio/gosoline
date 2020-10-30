package clock

import (
	"time"
)

type Ticker interface {
	Stop()
	Reset()
	Tick() <-chan time.Time
}

type TickerFactory func(duration time.Duration) Ticker

type realTicker struct {
	ticker   *time.Ticker
	duration time.Duration
}

func NewRealTicker(duration time.Duration) Ticker {
	t := &realTicker{
		ticker:   time.NewTicker(duration),
		duration: duration,
	}

	return t
}

func (t *realTicker) Reset() {
	t.ticker.Reset(t.duration)
}

func (t *realTicker) Stop() {
	t.ticker.Stop()
}

func (t *realTicker) Tick() <-chan time.Time {
	return t.ticker.C
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
