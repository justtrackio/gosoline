package clock

import "sync/atomic"

var Provider = NewRealClock()
var useUTCEnabled int32 = 0

func WithProvider(def Clock) {
	Provider = def
}

func WithUseUTC(useUTC bool) {
	enabled := int32(0)
	if useUTC {
		enabled = 1
	}

	atomic.SwapInt32(&useUTCEnabled, enabled)
}

func shouldUseUTC() bool {
	return atomic.LoadInt32(&useUTCEnabled) == 1
}
