package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/stretchr/testify/assert"
)

func TestRealTimer(t *testing.T) {
	for _, isUtc := range []bool{false, true} {
		clock.WithUseUTC(isUtc)
		c := clock.NewRealClock()
		start := c.Now()
		timer := c.NewTimer(time.Millisecond * 10)
		end := <-timer.Chan()
		if isUtc {
			assert.Equal(t, end.UTC(), end)
		}
		assert.GreaterOrEqual(t, end.Sub(start), time.Millisecond*10)

		// check if we can reuse it with 0
		start = c.Now()
		timer.Reset(0)
		end = <-timer.Chan()
		if isUtc {
			assert.Equal(t, end.UTC(), end)
		}
		assert.GreaterOrEqual(t, end.Sub(start), time.Duration(0))

		// check if we can reuse a timer properly
		start = c.Now()
		timer.Reset(time.Millisecond * 20)
		end = <-timer.Chan()
		if isUtc {
			assert.Equal(t, end.UTC(), end)
		}
		assert.GreaterOrEqual(t, end.Sub(start), time.Millisecond*20)

		// check if we can stop and reset a timer properly
		timer.Reset(time.Hour)
		stdTimer := time.NewTimer(time.Millisecond)
		select {
		case <-stdTimer.C:
			// the timer with 1ms should trigger before the timer with 1h, so this is correct
			break
		case <-timer.Chan():
			assert.Fail(t, "timer should not have triggered that fast")
			return
		}
		stopped := timer.Stop()
		assert.True(t, stopped)

		// check if we can now use the timer again
		start = c.Now()
		timer.Reset(time.Millisecond * 30)
		end = <-timer.Chan()
		if isUtc {
			assert.Equal(t, end.UTC(), end)
		}
		assert.GreaterOrEqual(t, end.Sub(start), time.Millisecond*30)

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
}

func TestRealTimer_NewTimerWithZero(t *testing.T) {
	c := clock.NewRealClock()
	timer := c.NewTimer(0)

	select {
	case now := <-timer.Chan():
		assert.GreaterOrEqual(t, c.Now().UnixNano(), now.UnixNano())
	default:
		assert.Fail(t, "reading from a timer with 0 duration should work")
	}

	timer.Reset(0)
	select {
	case now := <-timer.Chan():
		assert.GreaterOrEqual(t, c.Now().UnixNano(), now.UnixNano())
	default:
		assert.Fail(t, "reading from a timer reset to 0 duration should work")
	}
}

func TestRealTimer_NewTimerWithNegative(t *testing.T) {
	c := clock.NewRealClock()
	timer := c.NewTimer(-1)

	select {
	case now := <-timer.Chan():
		assert.GreaterOrEqual(t, c.Now().UnixNano(), now.UnixNano())
	default:
		assert.Fail(t, "reading from a timer with negative duration should work")
	}

	timer.Reset(-1)
	select {
	case now := <-timer.Chan():
		assert.GreaterOrEqual(t, c.Now().UnixNano(), now.UnixNano())
	default:
		assert.Fail(t, "reading from a timer reset to negative duration should work")
	}
}

func TestRealTimerConcurrentResetAndStop(t *testing.T) {
	timer := clock.NewRealTimer(time.Minute)
	cfn := coffin.New()
	for i := 0; i < 100; i++ {
		cfn.Go(func() error {
			for j := 0; j < 10000; j++ {
				timer.Reset(time.Minute)
			}

			return nil
		})
		cfn.Go(func() error {
			timer.Stop()

			return nil
		})
	}

	err := cfn.Wait()
	assert.NoError(t, err)
}
