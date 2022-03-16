package clock_test

import (
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
)

func TestNewFakeClock(t *testing.T) {
	c := clock.NewFakeClock()
	now := c.Now()
	assert.NotZero(t, now)
	time.Sleep(time.Millisecond)
	assert.Equal(t, now, c.Now())

	//
	c2 := clock.NewFakeClock()
	assert.Equal(t, c.Now(), c2.Now())
}

func TestNewFakeClockAt(t *testing.T) {
	now := time.Now()
	c := clock.NewFakeClockAt(now)
	time.Sleep(time.Millisecond)
	assert.Equal(t, now, c.Now())
}

func TestFakeClock_Since(t *testing.T) {
	c := clock.NewFakeClock()
	start := c.Now()
	c.Advance(time.Hour)
	assert.Equal(t, time.Hour, c.Since(start))
}

func TestFakeClock_AdvanceSleep(t *testing.T) {
	i := 0
	c := clock.NewFakeClock()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		c.Sleep(time.Minute)
		i++
		wg.Done()
	}()

	c.BlockUntil(1)
	assert.Equal(t, 0, i)

	c.Advance(time.Second)
	assert.Equal(t, 0, i)

	c.Advance(time.Second * 59)
	wg.Wait()
	assert.Equal(t, 1, i)
}

func TestFakeClockAfter(t *testing.T) {
	c := clock.NewFakeClock()

	assertCanRead(t, c.After(-1), "should be able to read immediately from a negative time")
	assertCanRead(t, c.After(0), "should be able to read immediately from a zero time")

	ms := c.After(time.Millisecond)
	sec := c.After(time.Second)
	min := c.After(time.Minute)
	h := c.After(time.Hour)

	c.Advance(time.Millisecond)
	assertCanRead(t, ms, "after advancing 1ms we should be able to read c.After(1ms)")
	assertCanNotRead(t, sec, min, h)

	c.Advance(time.Second)
	assertCanRead(t, sec, "after advancing 1s we should be able to read c.After(1s)")
	assertCanNotRead(t, min, h)

	c.Advance(time.Second)
	assertCanNotRead(t, min, h)

	c.Advance(time.Minute)
	assertCanRead(t, min, "after advancing 1m we should be able to read c.After(1m)")
	assertCanNotRead(t, h)

	c.Advance(time.Hour)
	assertCanRead(t, h, "after advancing 1h we should be able to read c.After(1h)")
	assertCanNotRead(t, h)
}

func assertCanRead(t *testing.T, c <-chan time.Time, msg string) {
	select {
	case <-c:
	default:
		assert.Fail(t, msg)
	}
}

func assertCanNotRead(t *testing.T, cs ...<-chan time.Time) {
	for _, c := range cs {
		select {
		case <-c:
			assert.Fail(t, "read time from channel which did not expect this yet")
		default:
		}
	}
}

func TestFakeClock_BlockUntil(t *testing.T) {
	c := clock.NewFakeClock()
	cs := make([]<-chan time.Time, 3)

	for _, waitForBlocked := range []bool{false, true} {
		ch := make(chan struct{})
		go func() {
			close(ch)
			c.BlockUntil(len(cs))
			c.Advance(time.Second)
		}()

		// BlockUntilTimers has two paths - either we already have enough channels waiting or we need to wait for more cs.
		// Thus, we at least once want to wait for the go routine to have a chance to run (although this is not a guarantee
		// that it also entered BlockUntilTimers, but there is not much we can do about that)
		if waitForBlocked {
			<-ch
		}

		for i := range cs {
			cs[i] = c.After(time.Second)
		}

		for _, c := range cs {
			<-c
		}
	}
}

func TestFakeClock_BlockUntilTimers(t *testing.T) {
	c := clock.NewFakeClock()
	timers := make([]clock.Timer, 3)

	for _, waitForBlocked := range []bool{false, true} {
		ch := make(chan struct{})
		go func() {
			close(ch)
			c.BlockUntilTimers(len(timers))
			c.Advance(time.Second)
		}()

		// BlockUntilTimers has two paths - either we already have enough timers waiting or we need to wait for more timers.
		// Thus, we at least once want to wait for the go routine to have a chance to run (although this is not a guarantee
		// that it also entered BlockUntilTimers, but there is not much we can do about that)
		if waitForBlocked {
			<-ch
		}

		for i := range timers {
			if timers[i] == nil {
				timers[i] = c.NewTimer(time.Second)
			} else {
				timers[i].Reset(time.Second)
			}
		}

		for _, timer := range timers {
			<-timer.Chan()
		}
	}
}

func TestFakeClock_BlockUntilTickers(t *testing.T) {
	c := clock.NewFakeClock()
	tickers := make([]clock.Ticker, 3)

	for _, waitForBlocked := range []bool{false, true} {
		ch := make(chan struct{})
		go func() {
			close(ch)
			c.BlockUntilTickers(len(tickers))
			c.Advance(time.Second)
		}()

		// BlockUntilTickers has two paths - either we already have enough tickers waiting or we need to wait for more tickers.
		// Thus, we at least once want to wait for the go routine to have a chance to run (although this is not a guarantee
		// that it also entered BlockUntilTickers, but there is not much we can do about that)
		if waitForBlocked {
			<-ch
		}

		for i := range tickers {
			if tickers[i] == nil {
				tickers[i] = c.NewTicker(time.Second)
			} else {
				tickers[i].Reset(time.Second)
			}
		}

		for _, ticker := range tickers {
			<-ticker.Chan()
		}
	}
}
