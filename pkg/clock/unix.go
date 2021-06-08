package clock

import "time"

// ToUnixNano converts a timestamp to a 64 bit unsigned integer, namely the number of nanoseconds since 1970-01-01T00:00:00Z.
func ToUnixNano(t time.Time) int64 {
	return t.UnixNano()
}

// FromUnixNano is the inverse of ToUnixNano and will return a timestamp in UTC if configured to do so, otherwise it will
// return a timestamp in the local time zone.
func FromUnixNano(nanoSeconds int64) time.Time {
	seconds := nanoSeconds / 1e9
	nanos := nanoSeconds % 1e9

	t := time.Unix(seconds, nanos)
	if shouldUseUTC() {
		t = t.UTC()
	}

	return t
}
