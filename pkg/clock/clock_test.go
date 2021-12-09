package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
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

func TestRealClock_Sleep(t *testing.T) {
	c := clock.NewRealClock()
	start := c.Now()
	c.Sleep(time.Millisecond * 5)
	took := c.Now().Sub(start)
	assert.GreaterOrEqual(t, took, time.Millisecond*5)
}
