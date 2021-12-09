package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
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
		assert.Fail(t, "there should not be a tick immediatly after a tick")
	default:
		// nop
	}
	ticker.Stop()
	// wait a bit for all routines to exit
	time.Sleep(time.Millisecond * 10)
}

func TestRealTicker_NewTickerWithZero(t *testing.T) {
	assert.PanicsWithError(t, "non-positive interval for NewTicker", func() {
		c := clock.NewRealClock()
		_ = c.NewTicker(0)
	})

	assert.PanicsWithError(t, "non-positive interval for Reset", func() {
		c := clock.NewRealClock()
		ticker := c.NewTicker(1)
		ticker.Reset(0)
	})
}

func TestRealTicker_NewTickerWithNegative(t *testing.T) {
	assert.PanicsWithError(t, "non-positive interval for NewTicker", func() {
		c := clock.NewRealClock()
		_ = c.NewTicker(-1)
	})

	assert.PanicsWithError(t, "non-positive interval for Reset", func() {
		c := clock.NewRealClock()
		ticker := c.NewTicker(1)
		ticker.Reset(-1)
	})
}
