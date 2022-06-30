package conc

import (
	"sync"
)

// Lazy provides a thread-safe way of creating a resource on-demand, allowing you to provide needed data with a parameter
type Lazy[T any, ARG any] interface {
	Get(arg ARG) (T, error)
}

type lazy[T any, ARG any] struct {
	init func(arg ARG) (T, error)
	val  T
	set  bool
	lck  sync.RWMutex
}

// NewLazy creates a new, empty Lazy with the given init function.
func NewLazy[T any, ARG any](init func(arg ARG) (T, error)) Lazy[T, ARG] {
	return &lazy[T, ARG]{
		init: init,
	}
}

func (l *lazy[T, ARG]) Get(arg ARG) (T, error) {
	l.lck.RLock()
	if l.set {
		l.lck.RUnlock()

		return l.val, nil
	}

	l.lck.RUnlock()
	l.lck.Lock()
	defer l.lck.Unlock()

	// need to check again because it can be changed between releasing the read lock and getting the write lock
	if l.set {
		return l.val, nil
	}

	var err error
	if l.val, err = l.init(arg); err != nil {
		var empty T

		return empty, err
	}

	// remember that we successfully initialized the value
	l.set = true

	// remove reference, allowing it to be GCed
	l.init = nil

	return l.val, nil
}
