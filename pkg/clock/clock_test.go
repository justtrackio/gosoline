package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/test/assert"
)

func TestRealClock_After(t *testing.T) {
	clock.WithUseUTC(true)
	c := clock.NewRealClock()
	ch := c.After(time.Millisecond)
	now := <-ch
	assert.Equal(t, now.UTC(), now)
}

func TestRealClock_NowYieldsUTC(t *testing.T) {
	clock.WithUseUTC(true)
	c := clock.NewRealClock()
	now := c.Now()
	assert.Equal(t, now.UTC(), now)
}

func TestNewFakeClock(t *testing.T) {
	c := clock.NewFakeClock()
	now := c.Now()
	time.Sleep(time.Millisecond)
	assert.Equal(t, now, c.Now())
}

func TestNewFakeClockAt(t *testing.T) {
	now := time.Now()
	c := clock.NewFakeClockAt(now)
	time.Sleep(time.Millisecond)
	assert.Equal(t, now, c.Now())
}
