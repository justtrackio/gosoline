package clock_test

import (
	"context"
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

func TestRealClock_SleepWithContext(t *testing.T) {
	c := clock.NewRealClock()
	start := c.Now()
	c.SleepWithContext(context.Background(), time.Millisecond*5)
	took := c.Now().Sub(start)
	assert.GreaterOrEqual(t, took, time.Millisecond*5)
}

func TestRealClock_SleepWithContext_ContextCancelled(t *testing.T) {
	c := clock.NewRealClock()
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(time.Millisecond)
		cancel()
	}()

	start := c.Now()
	c.SleepWithContext(ctx, time.Hour)
	took := c.Now().Sub(start)
	assert.Less(t, took, time.Hour)
}

func TestRealClock_SleepWithContext_AlreadyCancelledContext(t *testing.T) {
	c := clock.NewRealClock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := c.Now()
	c.SleepWithContext(ctx, time.Hour)
	took := c.Now().Sub(start)
	assert.Less(t, took, time.Hour)
}
