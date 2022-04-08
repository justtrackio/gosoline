package conc

import "sync/atomic"

type AtomicBoolean struct {
	value int32
}

func (a *AtomicBoolean) Set(value bool) {
	if value {
		atomic.StoreInt32(&a.value, 1)
	} else {
		atomic.StoreInt32(&a.value, 0)
	}
}

func (a *AtomicBoolean) Get() bool {
	return atomic.LoadInt32(&a.value) != 0
}

func (a *AtomicBoolean) Flip() (newValue bool) {
	for {
		old := atomic.LoadInt32(&a.value)
		if old == 0 {
			if atomic.CompareAndSwapInt32(&a.value, 0, 1) {
				return true
			}
		} else {
			if atomic.CompareAndSwapInt32(&a.value, old, 0) {
				return false
			}
		}
	}
}
