package ddb

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name LockManager
type LockManager interface {
	RenewLock(ctx context.Context, lockTime time.Duration, resource string, token string) (expiry time.Time, err error)
	ReleaseLock(ctx context.Context, resource string, token string) error
}

type ddbLock struct {
	manager  LockManager
	clock    clock.Clock
	logger   log.Logger
	ctx      context.Context
	resource string
	token    string
	expires  int64
	released conc.SignalOnce
}

func NewDdbLockFromInterfaces(
	manager LockManager,
	clock clock.Clock,
	logger log.Logger,
	ctx context.Context,
	resource string,
	token string,
	expires time.Time,
) *ddbLock {
	return &ddbLock{
		manager:  manager,
		clock:    clock,
		logger:   logger,
		ctx:      ctx,
		resource: resource,
		token:    token,
		expires:  expires.UnixMicro(),
		released: conc.NewSignalOnce(),
	}
}

func (l *ddbLock) Renew(ctx context.Context, lockTime time.Duration) error {
	if l == nil {
		return conc.ErrLockNotOwned
	}

	expiry, err := l.manager.RenewLock(ctx, lockTime, l.resource, l.token)

	if err == nil {
		atomic.StoreInt64(&l.expires, expiry.UnixMicro())
	}

	return err
}

func (l *ddbLock) Release() error {
	if l == nil {
		return conc.ErrLockNotOwned
	}

	// stop the debug thread if needed
	l.released.Signal()

	deadline := time.UnixMicro(atomic.LoadInt64(&l.expires))
	remainingLockTime := deadline.Sub(l.clock.Now())

	if remainingLockTime <= 0 {
		return conc.ErrLockNotOwned
	}

	// we should always release the lock, even when our parent gets canceled.
	// if we don't manage to do this until it expires anyway, there is no further point in trying.
	// make sure we have enough time to release the lock:
	delayedCtx, stop := exec.WithDelayedCancelContext(l.ctx, remainingLockTime)
	defer stop()

	// and that we don't spend more time than needed on it:
	ctx, cancel := context.WithDeadline(delayedCtx, deadline)
	defer cancel()

	err := l.manager.ReleaseLock(ctx, l.resource, l.token)
	if exec.IsRequestCanceled(err) {
		// map the cancel to no error at all: we should only get this error if the lock expired,
		// so we have "released" the lock in some other way as well, and we don't need to bother
		// our caller with this.
		return nil
	}

	return err
}

func (l *ddbLock) runWatcher() {
	t := l.clock.NewTimer(l.expiresIn())
	defer t.Stop()

	for {
		expiresIn := l.expiresIn()

		if expiresIn <= 0 {
			break
		}

		t.Reset(expiresIn)

		select {
		case <-t.Chan():
			continue
		case <-l.released.Channel():
			return
		}
	}

	l.logger.WithFields(log.Fields{
		"ddb_lock_token":    l.token,
		"ddb_lock_resource": l.resource,
	}).Warn(l.ctx, "failed to release or renew the lock before the timeout")
}

func (l *ddbLock) expiresIn() time.Duration {
	expires := time.UnixMicro(atomic.LoadInt64(&l.expires))
	now := l.clock.Now()

	return expires.Sub(now)
}
