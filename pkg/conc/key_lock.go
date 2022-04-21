package conc

import "sync"

type UnlockFunc func()

type KeyLock interface {
	// Lock based on the given key, returns an UnlockFunc when called it unlocks the underlying key and also
	// releases the resources.
	Lock(key any) UnlockFunc
}

type keyLock struct {
	lck     *sync.Mutex
	mutexes sync.Map
}

func NewKeyLock() *keyLock {
	return &keyLock{
		lck: &sync.Mutex{},
	}
}

func (l *keyLock) Lock(key any) UnlockFunc {
	l.lck.Lock()
	lockItem, _ := l.mutexes.LoadOrStore(key, &sync.Mutex{})
	l.lck.Unlock()

	mu := lockItem.(*sync.Mutex)
	mu.Lock()

	return func() {
		l.lck.Lock()

		mu.Unlock()
		l.mutexes.Delete(key)

		l.lck.Unlock()
	}
}
