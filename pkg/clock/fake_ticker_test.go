package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
)

func TestFakeTicker(t *testing.T) {
	c := clock.NewFakeClock()
	ticker := c.NewTicker(time.Millisecond * 10)

	// advance in two steps
	c.Advance(time.Millisecond * 5)
	assertCanNotRead(t, ticker.Chan())
	c.Advance(time.Millisecond * 5)
	assert.Equal(t, c.Now(), <-ticker.Chan())
	assertCanNotRead(t, ticker.Chan())

	// advance in one step
	c.Advance(time.Millisecond * 10)
	assert.Equal(t, c.Now(), <-ticker.Chan())
	assertCanNotRead(t, ticker.Chan())

	// advance two times
	c.Advance(time.Millisecond * 10)
	c.Advance(time.Millisecond * 10)
	assert.Equal(t, c.Now(), <-ticker.Chan())
	assertCanNotRead(t, ticker.Chan())

	// should not trigger after stopping
	ticker.Stop()
	c.Advance(time.Millisecond * 10)
	assertCanNotRead(t, ticker.Chan())

	// reset and retry stuff
	ticker.Reset(time.Millisecond * 15)
	c.Advance(time.Millisecond * 10)
	assertCanNotRead(t, ticker.Chan())
	c.Advance(time.Millisecond * 5)
	assert.Equal(t, c.Now(), <-ticker.Chan())

	// reset after some time has past
	c.Advance(time.Millisecond * 10)
	ticker.Reset(time.Millisecond * 20)
	c.Advance(time.Millisecond * 10)
	assertCanNotRead(t, ticker.Chan())
	c.Advance(time.Millisecond * 10)
	assert.Equal(t, c.Now(), <-ticker.Chan())
}

func TestFakeTicker_NewTickerWithZero(t *testing.T) {
	assert.PanicsWithError(t, "non-positive interval for NewTicker", func() {
		c := clock.NewFakeClock()
		_ = c.NewTicker(0)
	})

	assert.PanicsWithError(t, "non-positive interval for Reset", func() {
		c := clock.NewFakeClock()
		ticker := c.NewTicker(1)
		ticker.Reset(0)
	})
}

func TestFakeTicker_NewTickerWithNegative(t *testing.T) {
	assert.PanicsWithError(t, "non-positive interval for NewTicker", func() {
		c := clock.NewFakeClock()
		_ = c.NewTicker(-1)
	})

	assert.PanicsWithError(t, "non-positive interval for Reset", func() {
		c := clock.NewFakeClock()
		ticker := c.NewTicker(1)
		ticker.Reset(-1)
	})
}
