package conc

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/hashicorp/go-multierror"
	"time"
)

// you failed to acquire a lock before the operation timed out
var OwnedLockError = errors.New("lock owned")

// you tried to release a lock which you (no longer) own. Make sure
// you are not releasing a lock twice and are releasing a lock in a timely manner.
var NotOwnedError = errors.New("the lock was not (no longer) owned by you")

//go:generate mockery -name DistributedLockProvider
type DistributedLockProvider interface {
	// Acquire a lock for a duration (given e.g. in a constructor). Aborts the operation if the
	// context is canceled before the lock can be acquired.
	Acquire(ctx context.Context, resource string) (DistributedLock, error)
}

//go:generate mockery -name DistributedLock
type DistributedLock interface {
	// Extend your lease of the lock to at least the given duration
	// (so if your lock has 3 seconds remaining and you give a
	// duration of 5 seconds, your lock is now locked at least until
	// now + 5 seconds).
	// Aborts the operation if the context gets canceled before
	// the operation finishes.
	// Might fail with NotOwnedError if you are no longer the
	// owner of the lock.
	Renew(ctx context.Context, lockTime time.Duration) error
	// Release a lock. Might fail with NotOwnedError if you are
	// releasing a lock too late.
	Release() error
}

type DistributedLockSettings struct {
	Backoff         cloud.BackoffSettings
	DefaultLockTime time.Duration
	Domain          string
}

type ReleaseFunc func() error

// Start a new go routine which renews our distributed lock for us for lockTime every time lockTime/2 time passes.
// Returns a function which has to be used to release the lock eventually and kill the go routine keeping the lock alive.
// If the context gets canceled, the lock will also no longer be held alive, but it will not be released until you call
// Release on the lock or the ReleaseFunc returned by this function.
func HoldDistributedLock(ctx context.Context, lock DistributedLock, lockTime time.Duration) ReleaseFunc {
	ch := make(chan struct{})
	errCh := make(chan error)

	go func() {
		ticker := time.NewTimer(lockTime / 2)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// do nothing, we won't release the lock from someone who might still need to do some cleanup

				return
			case <-ch:
				// do nothing, the caller of the release func releases the lock

				return
			case <-ticker.C:
				err := lock.Renew(ctx, lockTime)

				if err != nil {
					if !cloud.IsRequestCanceled(err) {
						// we really can't handle the error here, so we move it to a channel we can read from later

						errCh <- err
					}

					// we could retry again and again, but would risk only spinning forever, so we
					// stop this here and let our caller deal with this eventually

					return
				}

				ticker.Reset(lockTime / 2)
			}
		}
	}()

	return func() error {
		close(ch)

		// release the lock and combine any errors we might have gotten during the process

		releaseErr := lock.Release()
		var renewErr error

		select {
		case err := <-errCh:
			renewErr = err
		default:
		}

		if releaseErr != nil && renewErr != nil {
			return multierror.Append(renewErr, releaseErr)
		}

		if releaseErr != nil {
			return releaseErr
		}

		return renewErr
	}
}
