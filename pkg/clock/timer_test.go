package clock_test

import (
	"sync"
	"sync/atomic"
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

func TestRealTimer_ConcurrentResetAndStop(t *testing.T) {
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

// TestRealTimer_RaceCondition demonstrates an old data race in the realTimer implementation.
//
// This test should be run with the race detector enabled to help it expose the timing condition.
func TestRealTimer_RaceCondition(t *testing.T) {
	var done atomic.Int32
	const iterations = 10000

	go func() {
		// Loop multiple times to increase the chances of the scheduler
		// creating the conditions for the race.
		for i := 0; i < iterations; i++ {
			t.Logf("running iteration %d", i)
			// 1. Create a timer with a tiny duration. This immediately starts an internal
			//    goroutine that will eventually call sendTick() WITHOUT a lock.
			timer := clock.NewRealTimer(time.Microsecond)

			var wg sync.WaitGroup
			wg.Add(1)

			// 2. Concurrently, call Reset(0). This code path calls sendTick()
			//    WITH a lock held. This creates the race condition.
			go func() {
				defer wg.Done()
				timer.Reset(0)
			}()

			// 3. Wait for the Reset() call to complete.
			wg.Wait()

			// 4. Drain the timer's channel to prevent the internal goroutines
			//    from blocking on subsequent iterations of the loop.
			select {
			case <-timer.Chan():
			case <-time.After(100 * time.Millisecond):
				// This timeout prevents the test from hanging, which might happen
				// if the race condition causes a tick to be lost. The primary
				// goal, however, is to trigger the race detector.
			}

			done.Add(1)
		}
	}()

	assert.Eventually(t, func() bool {
		return done.Load() == iterations
	}, time.Second*30, time.Millisecond*10)
}

func TestRealTimer_ResetRace(t *testing.T) {
	// This test is designed to fail by exposing a race condition in Reset.
	// It may need to be run multiple times or with the -race detector to reliably fail.
	// A timer's goroutine can fire and deliver a stale tick after the timer has been reset.

	// Loop to increase the chances of triggering the race condition.
	for i := 0; i < 100; i++ {
		// 1. Start a timer with a very short duration.
		timer := clock.NewRealTimer(1 * time.Millisecond)

		// 2. Wait a moment to ensure the timer has likely fired.
		// The internal goroutine will receive from the underlying timer's channel
		// and then attempt to acquire the lock. We want to call Reset() in this window.
		time.Sleep(2 * time.Millisecond)

		// 3. Reset the timer to a much longer duration.
		// If the old goroutine was preempted, it's now blocked on the lock that Reset() holds.
		// Reset() will complete, releasing the lock and allowing the old goroutine to proceed.
		timer.Reset(time.Hour)

		// 4. Check for an immediate, stale tick.
		// After Reset returns, the old goroutine might acquire the lock and send its stale tick.
		// The channel should be empty, and this select should block until the new timer fires
		// (which won't happen during the test's short timeout).
		select {
		case tick := <-timer.Chan():
			// A value was received immediately after Reset. This should not happen.
			t.Fatalf("Stale tick received on channel after Reset: %v", tick)
		case <-time.After(20 * time.Millisecond):
			// This is the correct behavior. No value was received after a reasonable time.
			// The test for this iteration passes.
		}

		timer.Stop()
	}
}
