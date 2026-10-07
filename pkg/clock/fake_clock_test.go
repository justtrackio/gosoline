package clock_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
)

func TestNewFakeClock(t *testing.T) {
	c := clock.NewFakeClock()
	now := c.Now()
	assert.NotZero(t, now)
	time.Sleep(time.Millisecond)
	assert.Equal(t, now, c.Now())

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

func TestFakeClock_SleepWithContext(t *testing.T) {
	i := 0
	c := clock.NewFakeClock()

	var wg sync.WaitGroup
	wg.Go(func() {
		c.SleepWithContext(t.Context(), time.Minute)
		i++
	})

	c.BlockUntil(1)
	assert.Equal(t, 0, i)

	c.Advance(time.Second)
	assert.Equal(t, 0, i)

	c.Advance(time.Second * 59)
	wg.Wait()
	assert.Equal(t, 1, i)
}

func TestFakeClock_SleepWithContext_ContextCancelled(t *testing.T) {
	i := 0
	c := clock.NewFakeClock()
	ctx, cancel := context.WithCancel(t.Context())

	var wg sync.WaitGroup
	wg.Go(func() {
		c.SleepWithContext(ctx, time.Minute)
		i++
	})

	c.BlockUntil(1)
	assert.Equal(t, 0, i)

	cancel()
	wg.Wait()
	assert.Equal(t, 1, i)
}

func TestFakeClock_SleepWithContext_CanceledSleeperUnregistered(t *testing.T) {
	c := clock.NewFakeClock()
	ctxA, cancelA := context.WithCancel(t.Context())
	ctxB, cancelB := context.WithCancel(t.Context())

	aDone := make(chan struct{})
	bDone := make(chan struct{})
	go func() {
		c.SleepWithContext(ctxA, time.Minute)
		close(aDone)
	}()
	go func() {
		c.SleepWithContext(ctxB, time.Minute)
		close(bDone)
	}()

	c.BlockUntil(2)
	cancelA()
	<-aDone

	// A's canceled sleeper must no longer be counted as waiting, so BlockUntil(2)
	// must not return while only B is still sleeping
	unblocked := make(chan struct{})
	go func() {
		c.BlockUntil(2)
		close(unblocked)
	}()
	select {
	case <-unblocked:
		t.Fatal("BlockUntil(2) returned although only one sleeper is waiting")
	case <-time.After(50 * time.Millisecond):
	}

	// B's sleeper must still be registered and fire when the clock advances
	c.Advance(time.Minute)
	<-bDone
	cancelB()
}

func TestFakeClockAfter(t *testing.T) {
	c := clock.NewFakeClock()

	assertCanRead(t, c.After(-1), "should be able to read immediately from a negative time")
	assertCanRead(t, c.After(0), "should be able to read immediately from a zero time")

	ms := c.After(time.Millisecond)
	sec := c.After(time.Second)
	minute := c.After(time.Minute)
	h := c.After(time.Hour)

	c.Advance(time.Millisecond)
	assertCanRead(t, ms, "after advancing 1ms we should be able to read c.After(1ms)")
	assertCanNotRead(t, sec, minute, h)

	c.Advance(time.Second)
	assertCanRead(t, sec, "after advancing 1s we should be able to read c.After(1s)")
	assertCanNotRead(t, minute, h)

	c.Advance(time.Second)
	assertCanNotRead(t, minute, h)

	c.Advance(time.Minute)
	assertCanRead(t, minute, "after advancing 1m we should be able to read c.After(1m)")
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
	blockUntilSomethingTest(clock.FakeClock.BlockUntilTimers, clock.FakeClock.NewTimer)
}

func TestFakeClock_BlockUntilTickers(t *testing.T) {
	blockUntilSomethingTest(clock.FakeClock.BlockUntilTickers, clock.FakeClock.NewTicker)
}

type timerOrTicker interface {
	comparable
	Reset(duration time.Duration)
	Chan() <-chan time.Time
}

func blockUntilSomethingTest[T timerOrTicker](
	blockUntilSomething func(fakeClock clock.FakeClock, count int),
	mkEntry func(fakeClock clock.FakeClock, duration time.Duration) T,
) {
	c := clock.NewFakeClock()
	entries := make([]T, 3)

	for _, waitForBlocked := range []bool{false, true} {
		ch := make(chan struct{})
		go func() {
			close(ch)
			blockUntilSomething(c, len(entries))
			c.Advance(time.Second)
		}()

		// BlockUntilTimers/BlockUntilTickers have two paths - either we already have enough timers/tickers waiting
		// or we need to wait for more timers/tickers. Thus, we at least once want to wait for the go routine to
		// have a chance to run (although this is not a guarantee that it also entered BlockUntilTimers/BlockUntilTickers,
		// but there is not much we can do about that)
		if waitForBlocked {
			<-ch
		}

		for i := range entries {
			if entries[i] == mdl.Empty[T]() {
				entries[i] = mkEntry(c, time.Second)
			} else {
				entries[i].Reset(time.Second)
			}
		}

		for _, entry := range entries {
			<-entry.Chan()
		}
	}
}
