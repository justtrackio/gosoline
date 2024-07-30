package conc

import (
	"context"
	"errors"
	"time"
)

// you failed to acquire a lock before the operation timed out
var ErrOwnedLock = errors.New("lock owned")

// you tried to release a lock that you (no longer) own. Make sure you are not releasing a lock twice and are releasing a lock in a timely manner.
var ErrNotOwned = errors.New("the lock was not (no longer) owned by you")

//go:generate mockery --name DistributedLockProvider
type DistributedLockProvider interface {
	// Acquire a lock for a duration (given e.g. in a constructor). Aborts the operation if the
	// context is canceled before the lock can be acquired.
	Acquire(ctx context.Context, resource string) (DistributedLock, error)
}

//go:generate mockery --name DistributedLock
type DistributedLock interface {
	// Extend your lease of the lock to at least the given duration
	// (so if your lock has 3 seconds remaining and you give a
	// duration of 5 seconds, your lock is now locked at least until
	// now + 5 seconds).
	// Aborts the operation if the context gets canceled before
	// the operation finishes.
	// Might fail with ErrNotOwned if you are no longer the
	// owner of the lock.
	Renew(ctx context.Context, lockTime time.Duration) error
	// Release a lock. Might fail with ErrNotOwned if you are
	// releasing a lock too late.
	Release() error
}
