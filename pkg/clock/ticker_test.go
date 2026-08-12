package clock_test

import (
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/stretchr/testify/assert"
)

func TestRealTicker_Chan(t *testing.T) {
	clock.WithUseUTC(true)
	start := time.Now()
	ticker := clock.NewRealClock().NewTicker(time.Millisecond * 10)
	<-ticker.Chan()
	<-ticker.Chan()
	<-ticker.Chan()
	ticker.Stop()
	end := time.Now()
	assert.GreaterOrEqual(t, int64(end.Sub(start)), int64(time.Millisecond*30), "%v should be at least 30ms", end.Sub(start))
	// wait a bit for all routines to exit
	time.Sleep(time.Millisecond * 10)
}

func TestRealTicker_Reset(t *testing.T) {
	clock.WithUseUTC(true)
	start := time.Now()
	ticker := clock.NewRealClock().NewTicker(time.Millisecond * 300)
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 10)
		resetStart := time.Now()
		ticker.Reset(time.Millisecond * 300)
		resetEnd := time.Now()
		assert.Less(t, int64(resetEnd.Sub(resetStart)), int64(time.Millisecond*100), "a reset should take at most 100ms, took %v", resetEnd.Sub(resetStart))
		select {
		case <-ticker.Chan():
			assert.Fail(t, "unexpected tick received")
		default:
			// nop
		}
	}
	<-ticker.Chan()
	ticker.Stop()
	end := time.Now()
	assert.GreaterOrEqual(t, int64(end.Sub(start)), int64(time.Millisecond*400), "%v should be at least 400ms", end.Sub(start))
	// wait a bit for all routines to exit
	time.Sleep(time.Millisecond * 10)
}

func TestRealTicker_Reset_DuringTick(t *testing.T) {
	clock.WithUseUTC(true)
	ticker := clock.NewRealClock().NewTicker(time.Millisecond * 10)
	time.Sleep(time.Millisecond * 50)
	ticker.Reset(time.Millisecond * 10)
	time.Sleep(time.Millisecond * 50)
	<-ticker.Chan()
	select {
	case <-ticker.Chan():
		assert.Fail(t, "there should not be a tick immediately after a tick")
	default:
		// nop
	}
	ticker.Stop()
	// wait a bit for all routines to exit
	time.Sleep(time.Millisecond * 10)
}

func TestRealTicker_NewTickerWithZero(t *testing.T) {
	assert.PanicsWithError(t, "non-positive interval (0s) for NewTicker", func() {
		c := clock.NewRealClock()
		_ = c.NewTicker(0)
	})

	assert.PanicsWithError(t, "non-positive interval (0s) for Reset", func() {
		c := clock.NewRealClock()
		ticker := c.NewTicker(1)
		ticker.Reset(0)
	})
}

func TestRealTicker_NewTickerWithNegative(t *testing.T) {
	assert.PanicsWithError(t, "non-positive interval (-1ns) for NewTicker", func() {
		c := clock.NewRealClock()
		_ = c.NewTicker(-1)
	})

	assert.PanicsWithError(t, "non-positive interval (-1ns) for Reset", func() {
		c := clock.NewRealClock()
		ticker := c.NewTicker(1)
		ticker.Reset(-1)
	})
}

func TestRealTicker_ConcurrentResetAndStop(t *testing.T) {
	ticker := clock.NewRealTicker(time.Minute)
	cfn := coffin.New()
	for i := 0; i < 100; i++ {
		cfn.Go(func() error {
			for j := 0; j < 10000; j++ {
				ticker.Reset(time.Minute)
			}

			return nil
		})
		cfn.Go(func() error {
			ticker.Stop()

			return nil
		})
	}

	err := cfn.Wait()
	assert.NoError(t, err)
}

// tickWaitTimeout is a deliberately generous upper bound for waiting on something which has to happen. Assertions on
// real time are only stable in one direction: a loaded machine can delay a tick arbitrarily, but it can never make one
// arrive early. Tests therefore wait generously for expected events and never assert an upper bound on how long an
// event took to occur.
const tickWaitTimeout = 5 * time.Second

// waitForBufferedTick waits until the ticker holds a tick in its output buffer and returns the point in time at which
// that tick was observed. The buffered tick was necessarily generated before the returned timestamp, which allows
// tests to tell it apart from any tick generated afterwards without relying on absolute durations.
func waitForBufferedTick(t *testing.T, ticker clock.Ticker) time.Time {
	t.Helper()

	deadline := time.Now().Add(tickWaitTimeout)
	for time.Now().Before(deadline) {
		if len(ticker.Chan()) > 0 {
			return time.Now()
		}

		time.Sleep(time.Millisecond)
	}

	t.Fatal("timed out waiting for a tick to be buffered")

	return time.Time{}
}

func TestRealTicker_Buffering(t *testing.T) {
	// This test confirms that a tick is buffered
	// instead of being dropped immediately.
	t.Run("should buffer one tick for a briefly slow consumer", func(t *testing.T) {
		// ARRANGE: Create a ticker with a 100ms interval.
		const tickDuration = 100 * time.Millisecond
		ticker := clock.NewRealTicker(tickDuration)
		defer ticker.Stop()

		// Allow the first tick to be generated and sent to the buffer while we are not reading the channel at all,
		// which is what a briefly slow consumer looks like to the ticker.
		waitForBufferedTick(t, ticker)

		// ACT & ASSERT: The tick generated while we were busy has to still be readable without blocking.
		select {
		case <-ticker.Chan():
			// Test passed: we successfully read the buffered tick.
		default:
			t.Fatal("Test failed: Did not receive the buffered tick. It was likely dropped.")
		}
	})

	// This test confirms that if the buffer is full, the OLD tick is kept
	// and the NEW tick is dropped.
	t.Run("should drop new ticks when buffer is full", func(t *testing.T) {
		// ARRANGE: Create a ticker with a 100ms interval.
		const tickDuration = 100 * time.Millisecond
		ticker := clock.NewRealTicker(tickDuration)
		defer ticker.Stop()

		// Allow the first tick (Tick 1) to be generated and fill the buffer. Every tick generated from now on
		// carries a timestamp after bufferedAt.
		bufferedAt := waitForBufferedTick(t, ticker)

		// ACT: Wait long enough for further ticks to be generated. Since the buffer is full with Tick 1, they
		// should all be dropped.
		time.Sleep(3 * tickDuration)

		// ASSERT:
		// 1. The channel should still only contain one item.
		if l := len(ticker.Chan()); l != 1 {
			t.Fatalf("Expected buffer to contain 1 tick, but found %d", l)
		}

		// 2. The tick we read should be Tick 1, not one of the later ticks. Comparing against bufferedAt rather
		// than an absolute duration keeps this stable under load: if the machine is busy and delays Tick 1, then
		// bufferedAt is delayed by exactly the same amount.
		receivedTick := <-ticker.Chan()
		if receivedTick.After(bufferedAt) {
			t.Errorf("Wrong tick was kept in buffer. Expected the tick buffered before %v, got one from %v", bufferedAt, receivedTick)
		}
	})
}

// TestRealTicker_Reset_DoesNotSendStaleTick verifies that after resetting a Ticker,
// a stale tick from the previous, shorter period is not immediately sent on the channel.
// This test is designed to be deterministic by ensuring the output channel buffer is full
// (or has a tick pending) at the time of the reset, which would leave a stale tick if the bug were present.
func TestRealTicker_Reset_DoesNotSendStaleTick(t *testing.T) {
	// GIVEN a Ticker with a very short interval and a buffered output channel of 1.
	ticker := clock.NewRealTicker(1 * time.Millisecond)

	// WHEN we let the ticker run long enough for it to produce a tick and fill
	// the output buffer, without consuming it. This ensures a tick is pending.
	time.Sleep(50 * time.Millisecond)

	// AND we reset the ticker to a very long duration.
	// If the bug exists, the stale tick will remain in the output channel.
	// If the bug is fixed, the channel will be drained during the reset.
	ticker.Reset(1 * time.Hour)

	// THEN we should NOT receive a tick immediately.
	// We check for a tick for a duration much longer than the original period
	// but much shorter than the new period. If a stale tick was left in the
	// output buffer during the reset, this select will pick it up and fail the test.
	select {
	case <-ticker.Chan():
		// This is the failure case. We received a tick when we shouldn't have.
		t.Fatal("received a stale tick from the ticker after it was reset")
	case <-time.After(100 * time.Millisecond):
		// This is the success case. No tick was received, which is the correct behavior.
		// The test can now end successfully.
	}
}

// TestRealTicker_Stop ensures that after Stop() is called and returns, no more ticks are sent. This includes a tick
// which was already generated before Stop was called, but not consumed yet.
func TestRealTicker_Stop(t *testing.T) {
	// GIVEN a fast ticker
	ticker := clock.NewRealTicker(1 * time.Millisecond)
	// Consume one tick to ensure it's running
	<-ticker.Chan()

	// AND a tick which has been generated but not consumed. Waiting for it explicitly makes this deterministic:
	// otherwise whether a tick is pending at the time of the Stop below is decided by the scheduler.
	waitForBufferedTick(t, ticker)

	// WHEN we stop the ticker
	ticker.Stop()

	// THEN no more ticks should be received, the pending tick has to be discarded as well.
	select {
	case tick, ok := <-ticker.Chan():
		if ok {
			t.Fatalf("received a tick after Stop() was called: %v", tick)
		} else {
			t.Fatal("channel was closed unexpectedly")
		}
	case <-time.After(100 * time.Millisecond):
		// Success: No tick received after a reasonable wait time.
	}
}

// TestRealTicker_Reset_Timing ensures the first tick after a Reset arrives after the new duration.
func TestRealTicker_Reset_Timing(t *testing.T) {
	// GIVEN a ticker which would not produce a tick during this test
	ticker := clock.NewRealTicker(1 * time.Hour)
	defer ticker.Stop()

	// WHEN we reset it to a shorter, testable duration
	const resetDuration = 100 * time.Millisecond
	resetTime := time.Now()
	ticker.Reset(resetDuration)

	// THEN the next tick should arrive after the new duration has passed. Receiving a tick at all already proves the
	// reset took effect, as the original duration would not have produced one.
	select {
	case tickTime := <-ticker.Chan():
		elapsed := tickTime.Sub(resetTime)
		// Only the lower bound is asserted. It describes an actual property of the ticker, namely that no stale
		// tick of the previous period is delivered. An upper bound on the other hand only describes how busy the
		// machine running the test is.
		assert.GreaterOrEqual(t, elapsed, resetDuration, "tick arrived too early. elapsed: %v", elapsed)
	case <-time.After(tickWaitTimeout):
		t.Fatal("timed out waiting for tick after reset")
	}
}

// TestRealTicker_ResetAfterStop verifies that Reset() correctly restarts a ticker
// that has been previously stopped.
func TestRealTicker_ResetAfterStop(t *testing.T) {
	// ARRANGE: Create a ticker and immediately stop it.
	ticker := clock.NewRealTicker(1 * time.Hour)
	ticker.Stop()

	// ACT: Reset the ticker to a short, testable duration. This should restart it.
	ticker.Reset(50 * time.Millisecond)
	defer ticker.Stop()

	// ASSERT: We should receive a tick from the restarted ticker.
	select {
	case <-ticker.Chan():
		// Success: a tick was received, so the ticker was restarted.
	case <-time.After(tickWaitTimeout):
		t.Fatal("timed out waiting for a tick; Reset() did not restart the stopped ticker")
	}
}

// TestRealTicker_StopIdempotency ensures that calling Stop() multiple times
// on the same ticker is a safe operation and does not cause a panic or deadlock.
func TestRealTicker_StopIdempotency(t *testing.T) {
	// ARRANGE: Create a running ticker.
	ticker := clock.NewRealTicker(10 * time.Millisecond)

	// ACT & ASSERT: Call Stop() multiple times. The test passes if it completes
	// without panicking or deadlocking.
	assert.NotPanics(t, func() {
		ticker.Stop()
		ticker.Stop()
	}, "calling Stop() multiple times should not cause a panic")
}

// TestRealTicker_ConcurrentResetAndChan stress-tests for race conditions between
// Reset() calls and reads from the ticker's channel.
func TestRealTicker_ConcurrentResetAndChan(t *testing.T) {
	// GIVEN a ticker and a WaitGroup to manage goroutines
	ticker := clock.NewRealTicker(1 * time.Microsecond)
	var wg sync.WaitGroup
	// Use a quit channel to signal goroutines to stop
	quit := make(chan struct{})

	wg.Add(2)

	// WHEN one goroutine continuously resets the ticker
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			default:
				ticker.Reset(1 * time.Microsecond)
			}
		}
	}()

	// AND another goroutine continuously consumes from the ticker's channel
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				// Before returning, drain the channel to prevent the other
				// goroutine from blocking on a send if we exit first.
				for len(ticker.Chan()) > 0 {
					<-ticker.Chan()
				}

				return
			case <-ticker.Chan():
				// Just consume the tick
			}
		}
	}()

	// THEN the test runs for a short duration without panicking.
	// A panic would indicate a race condition (e.g., "send on closed channel").
	time.Sleep(100 * time.Millisecond)
	close(quit)
	wg.Wait()
	ticker.Stop()
}
