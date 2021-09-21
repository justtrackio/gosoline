package clock_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
)

func TestUnixNano(t *testing.T) {
	c := clock.NewRealClock()

	for _, useUTC := range []bool{false, true} {
		clock.WithUseUTC(useUTC)

		// get the current time, but strip the monotonic clock part from it (that is what Truncate(0) does) because we
		// can't restore that upon parsing it from a timestamp again
		now := c.Now().Truncate(0)
		u1 := clock.ToUnixNano(now)
		u2 := clock.ToUnixNano(now.UTC())
		assert.Equal(t, u1, u2, "unix nano should make no difference for the time zone")

		parsed := clock.FromUnixNano(u1)
		assert.Equalf(t, now, parsed, "parsing a unix timestamp should result in the same value it was generated from, got %s != %s", now.Format(time.RFC3339Nano), parsed.Format(time.RFC3339Nano))
	}
}
