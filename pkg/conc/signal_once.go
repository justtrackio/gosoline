package conc

import (
	"sync"
)

//go:generate mockery --name SignalOnce
type SignalOnce interface {
	// Signal causes the channel returned by Channel to be closed.
	// All go routines waiting on that channel thus immediately get a value.
	// Can be called more than once.
	Signal()
	// Channel returns a channel you can read on to wait for Signal to be called.
	Channel() chan struct{}
	// Signaled returns true after Signal has been called at least once.
	Signaled() bool
}

type signalOnce struct {
	c      chan struct{}
	closed sync.Once
}

func NewSignalOnce() SignalOnce {
	return &signalOnce{
		c:      make(chan struct{}),
		closed: sync.Once{},
	}
}

func (c *signalOnce) Signal() {
	c.closed.Do(func() {
		close(c.c)
	})
}

func (c *signalOnce) Channel() chan struct{} {
	return c.c
}

func (c *signalOnce) Signaled() bool {
	select {
	case <-c.c:
		return true
	default:
		return false
	}
}
