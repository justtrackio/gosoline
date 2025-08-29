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
	RenewLock(ctx context.Context, lockTime time.Duration, resource string, token string) error
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
	expires int64,
) *ddbLock {
	return &ddbLock{
		manager:  manager,
		clock:    clock,
		logger:   logger,
		ctx:      ctx,
		resource: resource,
		token:    token,
		expires:  expires,
		released: conc.NewSignalOnce(),
	}
}

func (l *ddbLock) Renew(ctx context.Context, lockTime time.Duration) error {
	if l == nil {
		return conc.ErrNotOwned
	}

	err := l.manager.RenewLock(ctx, lockTime, l.resource, l.token)

	if err == nil {
		atomic.StoreInt64(&l.expires, l.clock.Now().Add(lockTime).Unix())
	}

	return err
}

func (l *ddbLock) Release() error {
	if l == nil {
		return conc.ErrNotOwned
	}

	// stop the debug thread if needed
	l.released.Signal()

	deadline := time.Unix(atomic.LoadInt64(&l.expires), 0)
	remainingLockTime := deadline.Sub(l.clock.Now())

	if remainingLockTime <= 0 {
		return conc.ErrNotOwned
	}

	done := make(chan struct{})
	defer close(done)

	// we should always release the lock, even when our parent gets cancelled.
	// if we don't manage to do this until it expires anyway, there is no further point in trying.
	ctx, cancel := exec.WithManualCancelContext(l.ctx)
	go func() {
		timer := l.clock.NewTimer(remainingLockTime)
		defer timer.Stop()

		select {
		case <-done:
			return
		case <-timer.Chan():
			cancel()
		}
	}()

	return l.manager.ReleaseLock(ctx, l.resource, l.token)
}

func (l *ddbLock) runWatcher() {
	for {
		expires := atomic.LoadInt64(&l.expires)
		now := l.clock.Now()

		if expires < now.Unix() {
			break
		}

		t := time.NewTimer(time.Unix(expires, 0).Sub(now))

		select {
		case <-t.C:
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
