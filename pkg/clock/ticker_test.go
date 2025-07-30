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

func TestRealTicker_Buffering(t *testing.T) {
	// This test confirms that a tick is buffered for a short period
	// instead of being dropped immediately.
	t.Run("should buffer one tick for a briefly slow consumer", func(t *testing.T) {
		// ARRANGE: Create a ticker with a 100ms interval.
		const tickDuration = 100 * time.Millisecond
		ticker := clock.NewRealTicker(tickDuration)
		defer ticker.Stop()

		// Allow the first tick to be generated and sent to the buffer.
		time.Sleep(tickDuration + (tickDuration / 2))

		// ACT & ASSERT: We expect to read the buffered tick immediately.
		select {
		case <-ticker.Chan():
			// Test passed: we successfully read the buffered tick.
		case <-time.After(50 * time.Millisecond):
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
		startTime := time.Now()

		// Allow the first tick (Tick 1) to be generated and fill the buffer.
		time.Sleep(tickDuration + (tickDuration / 2))

		// ACT: Wait long enough for a second tick (Tick 2) to be generated.
		// Since the buffer is full with Tick 1, Tick 2 should be dropped.
		time.Sleep(tickDuration)

		// ASSERT:
		// 1. The channel should still only contain one item.
		if len(ticker.Chan()) != 1 {
			t.Fatalf("Expected buffer to contain 1 tick, but found %d", len(ticker.Chan()))
		}

		// 2. The tick we read should be Tick 1, not Tick 2.
		// We verify this by checking its timestamp.
		receivedTick := <-ticker.Chan()
		timeSinceStart := receivedTick.Sub(startTime)

		// The timestamp should be from the first tick interval (~100ms), not the second (~200ms).
		if timeSinceStart > tickDuration*3/2 {
			t.Errorf("Wrong tick was kept in buffer. Expected timestamp ~%v, got %v", tickDuration, timeSinceStart)
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

// TestRealTicker_Stop ensures that after Stop() is called and returns, no more ticks are sent.
func TestRealTicker_Stop(t *testing.T) {
	// GIVEN a fast ticker
	ticker := clock.NewRealTicker(1 * time.Millisecond)
	// Consume one tick to ensure it's running
	<-ticker.Chan()

	// WHEN we stop the ticker
	ticker.Stop()

	// THEN no more ticks should be received.
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
	// GIVEN a ticker
	ticker := clock.NewRealTicker(1 * time.Hour)

	// WHEN we reset it to a shorter, testable duration
	resetTime := time.Now()
	ticker.Reset(100 * time.Millisecond)

	// THEN the next tick should arrive approximately after the new duration has passed.
	select {
	case tickTime := <-ticker.Chan():
		elapsed := tickTime.Sub(resetTime)
		// We expect the tick to arrive *after* the reset duration.
		// We allow for a generous upper bound to account for test scheduler latency.
		assert.True(t, elapsed >= 100*time.Millisecond, "tick arrived too early. elapsed: %v", elapsed)
		assert.True(t, elapsed < 400*time.Millisecond, "tick arrived too late. elapsed: %v", elapsed)
	case <-time.After(500 * time.Millisecond):
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
	case <-time.After(150 * time.Millisecond):
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
