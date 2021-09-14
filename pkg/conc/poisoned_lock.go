package conc

import (
	"errors"
	"sync"
)

var ErrAlreadyPoisoned = errors.New("lock was already poisoned")

//go:generate mockery --name PoisonedLock
type PoisonedLock interface {
	Lock()
	TryLock() error
	Unlock()
	Poison()
}

type poisonedLock struct {
	lck      sync.Mutex
	poisoned bool
}

func (p *poisonedLock) TryLock() error {
	p.lck.Lock()

	if p.poisoned {
		p.lck.Unlock()

		return ErrAlreadyPoisoned
	}

	return nil
}

func (p *poisonedLock) Lock() {
	err := p.TryLock()
	if err != nil {
		panic(err)
	}
}

func (p *poisonedLock) Unlock() {
	p.lck.Unlock()
}

func (p *poisonedLock) Poison() {
	p.lck.Lock()
	defer p.lck.Unlock()

	p.poisoned = true
}

func NewPoisonedLock() PoisonedLock {
	return &poisonedLock{
		lck:      sync.Mutex{},
		poisoned: false,
	}
}
