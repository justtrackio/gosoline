package coffin

import (
	"context"
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
	Wait() error
}

type coffin struct {
	// we MUST represent this as a ptr as tomb.Tomb contains a mutex which
	// we are not allowed to copy!
	tomb *tomb.Tomb
}

func New() Coffin {
	return &coffin{
		tomb: new(tomb.Tomb),
	}
}

func WithContext(parent context.Context) (Coffin, context.Context) {
	tmb, ctx := tomb.WithContext(parent)
	cfn := &coffin{
		tomb: tmb,
	}

	return cfn, ctx
}

func (c *coffin) Alive() bool {
	return c.tomb.Alive()
}

func (c *coffin) Context(parent context.Context) context.Context {
	return c.tomb.Context(parent)
}

func (c *coffin) Dead() <-chan struct{} {
	return c.tomb.Dead()
}

func (c *coffin) Dying() <-chan struct{} {
	return c.tomb.Dying()
}

func (c *coffin) Err() (reason error) {
	return c.tomb.Err()
}

func (c *coffin) Go(f func() error) {
	c.tomb.Go(func() (err error) {
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
	c.tomb.Go(func() (err error) {
		defer func() {
			panicErr := ResolveRecovery(recover())

			if panicErr != nil {
				err = errors.Wrapf(panicErr, msg, args...)
			}
		}()

		err = f()
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

func (c *coffin) Kill(reason error) {
	c.tomb.Kill(reason)
}

func (c *coffin) Killf(f string, a ...interface{}) error {
	return c.tomb.Killf(f, a...)
}

func (c *coffin) Wait() error {
	return c.tomb.Wait()
}
