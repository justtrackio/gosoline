package coffin

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"
)

type Coffin interface {
	// Alive returns true if the coffin is not in a dying or dead state.
	Alive() bool
	// Context returns a context that is a copy of the provided parent context with
	// a replaced Done channel that is closed when either the coffin is dying or the
	// parent is cancelled.
	//
	// If parent is nil, it defaults to the parent provided via WithContext, or an
	// empty background parent if the coffin wasn't created via WithContext.
	Context(parent context.Context) context.Context
	// Dead returns the channel that can be used to wait until
	// all goroutines have finished running.
	Dead() <-chan struct{}
	// Dying returns the channel that can be used to wait until
	// t.Kill is called.
	Dying() <-chan struct{}
	Err() (reason error)
	// Go runs f in a new goroutine and tracks its termination.
	//
	// If f returns a non-nil error, t.Kill is called with that
	// error as the death reason parameter.
	//
	// It is f's responsibility to monitor the coffin and return
	// appropriately once it is in a dying state.
	//
	// It is safe for the f function to call the Go method again
	// to create additional tracked goroutines. Once all tracked
	// goroutines return, the Dead channel is closed and the
	// Wait method unblocks and returns the death reason.
	//
	// Calling the Go method after all tracked goroutines return
	// causes a runtime panic. For that reason, calling the Go
	// method a second time out of a tracked goroutine is unsafe.
	Go(f func() error)
	// Gof is like Go, but wraps the returned error with the given
	// name and args
	Gof(f func() error, name string, args ...interface{})
	// GoWithContext is like Go, but passes the given context to f
	GoWithContext(ctx context.Context, f func(ctx context.Context) error)
	// GoWithContextf is like Gof, but passes the given context to f
	GoWithContextf(ctx context.Context, f func(ctx context.Context) error, msg string, args ...interface{})
	// Kill puts the coffin in a dying state for the given reason,
	// closes the Dying channel, and sets Alive to false.
	//
	// Although Kill may be called multiple times, only the first
	// non-nil error is recorded as the death reason.
	//
	// If reason is ErrDying, the previous reason isn't replaced
	// even if nil. It's a runtime error to call Kill with ErrDying
	// if t is not in a dying state.
	Kill(reason error)
	// Killf calls the Kill method with an error built providing the received
	// parameters to fmt.Errorf. The generated error is also returned.
	Killf(f string, a ...interface{}) error
	// Wait blocks until all goroutines have finished running, and
	// then returns the reason for their death.
	//
	// If you never spawned a task using one of the Go function, Wait
	// returns nil.
	Wait() error
	// Returns the number of started go routines in this coffin.
	Started() int
	// Returns the number of currently running go routines in this coffin.
	Running() int
	// Returns the number of go routines that have already returned in this coffin.
	Terminated() int
}

type coffin struct {
	// we MUST represent this as a ptr as tomb. Tomb contains a mutex that we are not allowed to copy!
	tomb *tomb.Tomb
	// number of started and stopped go routines (stopped << 32 | started)
	status int64
	// function to stop the go routine keeping the coffin alive until the first Wait call
	markRunning func()
}

const (
	startedShift            = 0
	terminatedShift         = 32
	increaseStartedCount    = 1 << startedShift
	increaseTerminatedCount = 1 << terminatedShift
	startedMask             = (1<<terminatedShift - 1) << startedShift
	terminatedMask          = ^0 ^ (1<<(terminatedShift+startedShift) - 1)
)

func New() Coffin {
	tmb := new(tomb.Tomb)

	return &coffin{
		tomb:        tmb,
		status:      0,
		markRunning: prepareTomb(tmb),
	}
}

// WithContext returns a new coffin that is killed when the provided parent
// context is canceled, and a copy of parent with a replaced Done channel
// that is closed when either the coffin is dying or the parent is canceled.
// The returned context may also be obtained via the coffin's Context method.
//
// If the context is canceled, the coffin is killed with the error from the context.
// Thus, you will normally get a context.Canceled error from a coffin you stop like this.
func WithContext(parent context.Context) (Coffin, context.Context) {
	tmb, ctx := tomb.WithContext(parent)
	cfn := &coffin{
		tomb:        tmb,
		status:      0,
		markRunning: prepareTomb(tmb),
	}

	return cfn, ctx
}

func prepareTomb(tmb *tomb.Tomb) func() {
	once := &sync.Once{}
	ch := make(chan struct{})
	tmb.Go(func() error {
		<-ch

		return nil
	})

	return func() {
		once.Do(func() {
			close(ch)
		})
	}
}

func (c *coffin) Alive() bool {
	c.markRunning()

	return c.tomb.Alive()
}

func (c *coffin) Context(parent context.Context) context.Context {
	return c.tomb.Context(parent)
}

func (c *coffin) Dead() <-chan struct{} {
	c.markRunning()

	return c.tomb.Dead()
}

func (c *coffin) Dying() <-chan struct{} {
	c.markRunning()

	return c.tomb.Dying()
}

func (c *coffin) Err() (reason error) {
	c.markRunning()

	return c.tomb.Err()
}

func (c *coffin) Go(f func() error) {
	atomic.AddInt64(&c.status, increaseStartedCount)
	c.tomb.Go(func() (err error) {
		defer atomic.AddInt64(&c.status, increaseTerminatedCount)
		defer func() {
			panicErr := ResolveRecovery(recover())

			if panicErr != nil {
				err = panicErr
			}
		}()

		return f()
	})
}

func (c *coffin) Gof(f func() error, msg string, args ...interface{}) {
	atomic.AddInt64(&c.status, increaseStartedCount)
	c.tomb.Go(func() (err error) {
		defer atomic.AddInt64(&c.status, increaseTerminatedCount)
		defer func() {
			panicErr := ResolveRecovery(recover())

			if panicErr != nil {
				err = errors.Wrapf(panicErr, msg, args...)
			}
		}()

		err = f()
		if err != nil {
			err = errors.Wrapf(err, msg, args...)
		}
		return
	})
}

func (c *coffin) GoWithContext(ctx context.Context, f func(ctx context.Context) error) {
	c.Go(func() error {
		return f(ctx)
	})
}

func (c *coffin) GoWithContextf(ctx context.Context, f func(ctx context.Context) error, msg string, args ...interface{}) {
	c.Gof(func() error {
		return f(ctx)
	}, msg, args...)
}

// Kill puts the coffin in a dying state for the given reason,
// closes the Dying channel, and sets Alive to false.
//
// Although Kill may be called multiple times, only the first
// non-nil error is recorded as the death reason.
//
// If reason is ErrDying, the previous reason isn't replaced
// even if nil. It's a runtime error to call Kill with ErrDying
// if t is not in a dying state.
func (c *coffin) Kill(reason error) {
	c.markRunning()
	c.tomb.Kill(reason)
}

func (c *coffin) Killf(f string, a ...interface{}) error {
	c.markRunning()

	return c.tomb.Killf(f, a...)
}

func (c *coffin) Wait() error {
	c.markRunning()

	return c.tomb.Wait()
}

func (c *coffin) Started() int {
	return int((atomic.LoadInt64(&c.status) >> startedShift) & startedMask)
}

func (c *coffin) Running() int {
	status := atomic.LoadInt64(&c.status)
	started := (status >> startedShift) & startedMask
	terminated := (status >> terminatedShift) & terminatedMask

	return int(started - terminated)
}

func (c *coffin) Terminated() int {
	return int((atomic.LoadInt64(&c.status) >> terminatedShift) & terminatedMask)
}
