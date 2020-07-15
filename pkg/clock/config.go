package clock

import "sync/atomic"

var useUTCEnabled int32 = 0

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
