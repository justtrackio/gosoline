package conc

import (
	"fmt"
	"sync"
)

// ErrAlreadyPoisoned is returned if you try to lock a lock which was already poisoned
var ErrAlreadyPoisoned = fmt.Errorf("lock was already poisoned")

// A PoisonedLock is similar to a sync.Mutex, but once you Poison it, any attempt to Lock it will fail. Thus, you can
// implement something which is available for some time and at some point no longer is available (because it was closed
// or released and is not automatically reopened, etc.)
//
//go:generate mockery --name PoisonedLock
type PoisonedLock interface {
	// MustLock is like TryLock, but panics if an error is returned by TryLock
	MustLock()
	// TryLock will acquire the lock if it has not yet been poisoned. Otherwise, an error is returned.
	TryLock() error
	// Unlock will release the lock again. You need to hold the lock before calling Unlock.
	Unlock()
	// Poison will acquire the lock, check if is not poisoned, and the poison the lock (so you can only poison a lock once).
	// After a lock has been poisoned, you can not lock it again. Instead, ErrAlreadyPoisoned will be returned by TryLock.
	Poison() error
	// PoisonIf will acquire the lock and run the supplied function. If the function returns true (regardless of any error),
	// the lock is poisoned, otherwise it is only unlocked.
	PoisonIf(func() (bool, error)) error
}

type poisonedLock struct {
	lck      sync.Mutex
	poisoned bool
}

// NewPoisonedLock creates a new lock which can be poisoned. It is initially unlocked and not poisoned.
func NewPoisonedLock() PoisonedLock {
	return &poisonedLock{
		lck:      sync.Mutex{},
		poisoned: false,
	}
}

func (p *poisonedLock) TryLock() error {
	p.lck.Lock()

	if p.poisoned {
		p.lck.Unlock()

		return ErrAlreadyPoisoned
	}

	return nil
}

func (p *poisonedLock) MustLock() {
	err := p.TryLock()
	if err != nil {
		panic(err)
	}
}

func (p *poisonedLock) Unlock() {
	p.lck.Unlock()
}

func (p *poisonedLock) Poison() error {
	p.lck.Lock()

	return p.poisonLocked()
}

func (p *poisonedLock) poisonLocked() error {
	defer p.lck.Unlock()

	if p.poisoned {
		return ErrAlreadyPoisoned
	}

	p.poisoned = true

	return nil
}

func (p *poisonedLock) PoisonIf(f func() (bool, error)) error {
	if err := p.TryLock(); err != nil {
		return err
	}
	defer p.Unlock()

	shouldPoison, err := f()
	if shouldPoison {
		p.poisoned = true
	}

	return err
}
