package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
)

func TestFakeClock_NewTimer(t *testing.T) {
	c := clock.NewFakeClock()
	start := c.Now()
	timer := c.NewTimer(time.Millisecond * 10)
	c.Advance(time.Millisecond * 5)
	select {
	case <-timer.Chan():
		assert.Fail(t, "timer should not have fired")
		return
	default:
		break
	}
	c.Advance(time.Millisecond * 5)
	end := <-timer.Chan()
	assert.Equal(t, end.Sub(start), time.Millisecond*10)

	// check if we can reuse a timer properly
	start = c.Now()
	timer.Reset(time.Millisecond * 20)
	c.Advance(time.Millisecond * 20)
	end = <-timer.Chan()
	assert.Equal(t, end.Sub(start), time.Millisecond*20)

	// check if we can stop and reset a timer properly
	timer.Reset(time.Hour)
	select {
	case <-timer.Chan():
		assert.Fail(t, "timer should not have fired")
		return
	default:
		break
	}
	stopped := timer.Stop()
	assert.True(t, stopped)

	// check if we can now use the timer again
	start = c.Now()
	timer.Reset(time.Millisecond * 30)
	c.Advance(time.Millisecond * 30)
	end = <-timer.Chan()
	assert.Equal(t, end.Sub(start), time.Millisecond*30)

	// check if we can advance the time without triggering another message
	c.Advance(time.Millisecond * 30)
	select {
	case <-timer.Chan():
		assert.Fail(t, "timer should not have fired again")
		return
	default:
		break
	}

	// we should be able to stop the timer after it fired (but as it already fired, it will return false)
	stopped = timer.Stop()
	assert.False(t, stopped)
	// and stop it again should not crash
	stopped = timer.Stop()
	assert.False(t, stopped)

	// calling reset twice should not cause any problems
	timer.Reset(time.Hour)
	timer.Reset(time.Minute * 30)

	// we are done, clean up
	timer.Stop()
}

func TestFakeTimer_NewTimerWithZero(t *testing.T) {
	c := clock.NewFakeClock()
	timer := c.NewTimer(0)

	select {
	case now := <-timer.Chan():
		assert.Equal(t, c.Now(), now)
	default:
		assert.Fail(t, "reading from a timer with 0 duration should work")
	}

	timer.Reset(0)
	select {
	case now := <-timer.Chan():
		assert.Equal(t, c.Now(), now)
	default:
		assert.Fail(t, "reading from a timer reset to 0 duration should work")
	}
}

func TestFakeTimer_NewTimerWithNegative(t *testing.T) {
	c := clock.NewFakeClock()
	timer := c.NewTimer(-1)

	select {
	case now := <-timer.Chan():
		assert.Equal(t, c.Now(), now)
	default:
		assert.Fail(t, "reading from a timer with negative duration should work")
	}

	timer.Reset(-1)
	select {
	case now := <-timer.Chan():
		assert.Equal(t, c.Now(), now)
	default:
		assert.Fail(t, "reading from a timer reset to negative duration should work")
	}
}
