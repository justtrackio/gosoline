package clock_test

import (
	"fmt"
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

func TestRealTickerConcurrentResetAndStop(t *testing.T) {
	ticker := clock.NewRealTicker(time.Minute)
	cfn := coffin.New(t.Context())
	for i := 0; i < 100; i++ {
		cfn.Go(fmt.Sprintf("reset task %d", i), func() error {
			for j := 0; j < 10000; j++ {
				ticker.Reset(time.Minute)
			}

			return nil
		})
		cfn.Go(fmt.Sprintf("stop task %d", i), func() error {
			ticker.Stop()

			return nil
		})
	}

	err := cfn.Wait()
	assert.NoError(t, err)
}
