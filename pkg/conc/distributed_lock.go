package conc

import (
	"context"
	"errors"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
)

var (
	// ErrLockOwned is returned if an operation failed to acquire a lock which is owned by
	// someone else before a timeout.
	ErrLockOwned = errors.New("lock owned")
	// ErrLockNotOwned is returned if you tried to release a lock that you (no longer) own.
	// Make sure you are not releasing a lock twice and are releasing a lock in a timely manner.
	ErrLockNotOwned = errors.New("the lock was not (no longer) owned by you")
)

//go:generate go run github.com/vektra/mockery/v2 --name DistributedLockProvider
type DistributedLockProvider interface {
	// Acquire a lock for a duration (given e.g. in a constructor). Aborts the
	// operation if the context is canceled before the lock can be acquired or
	// if the maximum number of retries is reached according to the backoff
	// configuration
	Acquire(ctx context.Context, resource string) (DistributedLock, error)
}

//go:generate go run github.com/vektra/mockery/v2 --name DistributedLock
type DistributedLock interface {
	// Renew extends your lease of the lock to at least the given duration (so
	// if your lock has 3 seconds remaining, and you give a duration of 5 seconds,
	// your lock is now locked at least until now + 5 seconds).
	// Aborts the operation if the context gets canceled before the operation
	// finishes.
	// Might fail with ErrLockNotOwned if you are no longer the owner of the lock.
	Renew(ctx context.Context, lockTime time.Duration) error
	// Release a lock. Might fail with ErrLockNotOwned if you ar releasing a lock too late.
	Release() error
}

type DistributedLockSettings struct {
	cfg.AppId
	Backoff         exec.BackoffSettings
	DefaultLockTime time.Duration
	Domain          string
}
