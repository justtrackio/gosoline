package conc

import (
	"context"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"sync/atomic"
	"time"
)

type ddbLock struct {
	manager  *ddbLockProvider
	ctx      context.Context
	resource string
	token    string
	expires  int64
	released SignalOnce
}

func newDdbLock(manager *ddbLockProvider, ctx context.Context, resource string, token string, expires int64) *ddbLock {
	return &ddbLock{
		manager:  manager,
		ctx:      ctx,
		resource: resource,
		token:    token,
		expires:  expires,
		released: NewSignalOnce(),
	}
}

func (l *ddbLock) Renew(ctx context.Context, lockTime time.Duration) error {
	if l == nil {
		return ErrNotOwned
	}

	err := l.manager.renew(ctx, lockTime, l.resource, l.token)

	if err == nil {
		atomic.SwapInt64(&l.expires, l.manager.clock.Now().Add(lockTime).Unix())
	}

	return err
}

func (l *ddbLock) Release() error {
	if l == nil {
		return ErrNotOwned
	}

	// stop the debug thread if needed
	l.released.Signal()

	ctx := exec.WithDelayedCancelContext(l.ctx, time.Second*3)
	// stop the cancel context eventually to make sure we are not leaking
	// a lot of go routines should our parent context get reused over and over
	defer ctx.Stop()

	return l.manager.release(ctx, l.resource, l.token)
}

func (l *ddbLock) forkWatcher() {
	go func() {
		for {
			expires := atomic.LoadInt64(&l.expires)
			now := l.manager.clock.Now()

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

		l.manager.logger.WithContext(l.ctx).WithFields(log.Fields{
			"ddb_lock_token":    l.token,
			"ddb_lock_resource": l.resource,
		}).Warn("failed to release or renew the lock before the timeout")
	}()
}
