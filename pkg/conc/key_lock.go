package conc

import (
	"sync"
)

type UnlockFunc func()

type KeyLock interface {
	// Lock based on the given key, returns an UnlockFunc when called it unlocks the underlying key and
	// releases the resources.
	Lock(key any) UnlockFunc
}

type keyLock struct {
	lck     sync.Mutex
	entries map[any]*entry
}

type entry struct {
	sync.Mutex
	refCount int
}

func NewKeyLock() KeyLock {
	return &keyLock{
		lck:     sync.Mutex{},
		entries: map[any]*entry{},
	}
}

func (l *keyLock) Lock(key any) UnlockFunc {
	l.lck.Lock()
	lockEntry, ok := l.entries[key]
	if !ok {
		lockEntry = &entry{}
		l.entries[key] = lockEntry
	}
	lockEntry.refCount++
	l.lck.Unlock()

	lockEntry.Lock()

	return func() {
		l.lck.Lock()

		lockEntry.refCount--
		lockEntry.Unlock()
		if lockEntry.refCount == 0 {
			delete(l.entries, key)
		}

		l.lck.Unlock()
	}
}
