package clock_test

import (
	"github.com/applike/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testableTicker interface {
	IsStopped() bool
}

func TestRealTicker_Tick(t *testing.T) {
	clock.WithUseUTC(true)
	start := time.Now()
	ticker := clock.NewRealTicker(time.Millisecond * 10)
	<-ticker.Tick()
	<-ticker.Tick()
	<-ticker.Tick()
	ticker.Stop()
	end := time.Now()
	assert.GreaterOrEqual(t, int64(end.Sub(start)), int64(time.Millisecond*30), "%v should be at least 30ms", end.Sub(start))
	// wait a bit for all routines to exit
	time.Sleep(time.Millisecond * 10)
	assert.True(t, ticker.(testableTicker).IsStopped())
}

func TestRealTicker_Reset(t *testing.T) {
	clock.WithUseUTC(true)
	start := time.Now()
	ticker := clock.NewRealTicker(time.Millisecond * 300)
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 10)
		resetStart := time.Now()
		ticker.Reset()
		resetEnd := time.Now()
		assert.Less(t, int64(resetEnd.Sub(resetStart)), int64(time.Millisecond*100), "a reset should take at most 100ms, took %v", resetEnd.Sub(resetStart))
		select {
		case <-ticker.Tick():
			assert.Fail(t, "unexpected tick received")
		default:
			// nop
		}
	}
	<-ticker.Tick()
	ticker.Stop()
	end := time.Now()
	assert.GreaterOrEqual(t, int64(end.Sub(start)), int64(time.Millisecond*400), "%v should be at least 400ms", end.Sub(start))
	// wait a bit for all routines to exit
	time.Sleep(time.Millisecond * 10)
	assert.True(t, ticker.(testableTicker).IsStopped())
}

func TestRealTicker_Reset_DuringTick(t *testing.T) {
	clock.WithUseUTC(true)
	ticker := clock.NewRealTicker(time.Millisecond * 10)
	time.Sleep(time.Millisecond * 50)
	ticker.Reset()
	time.Sleep(time.Millisecond * 50)
	<-ticker.Tick()
	select {
	case <-ticker.Tick():
		assert.Fail(t, "there should not be a tick immediatly after a tick")
	default:
		// nop
	}
	ticker.Stop()
	// wait a bit for all routines to exit
	time.Sleep(time.Millisecond * 10)
	assert.True(t, ticker.(testableTicker).IsStopped())
}

func TestRealTicker_Stop_DuringTick(t *testing.T) {
	clock.WithUseUTC(true)
	ticker := clock.NewRealTicker(time.Millisecond * 10)
	time.Sleep(time.Millisecond * 50)
	ticker.Stop()
	time.Sleep(time.Millisecond * 50)
	select {
	case <-ticker.Tick():
		assert.Fail(t, "expected tick to be eaten by ticker again")
	default:
		// nop
	}
	assert.True(t, ticker.(testableTicker).IsStopped())
}
