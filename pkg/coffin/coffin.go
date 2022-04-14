package coffin

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/conc"
)

// A StartingCoffin is used to construct a Coffin to track the execution of
// go routines. It is passed to your code in a callback and is only safe to
// use for the duration of said callback (except for the Running method).
// Thus, you are free to spawn new go routines only during the callback, any
// attempt later on will lead to a panic. This is needed to ensure we can
// implement proper semantics for the dying and dead channels (we can't reopen
// them if a new go routine is spawned inside a dead coffin).
type StartingCoffin interface {
	// Go runs f in a new goroutine and tracks its termination.
	//
	// If f returns a non-nil error, Kill is called with that error as the
	// death reason parameter.
	//
	// It is f's responsibility to monitor the Coffin and return
	// appropriately once it is in a dying state. To access the Coffin,
	// you can use the Coffin method which returns you the running Coffin
	// as soon as the callback starting the Coffin returned (as there is
	// no reasonable way to implement the methods returning the dying or
	// dead channels anyway while the Coffin is still starting).
	//
	// Calling Go after the callback which got the StartingCoffin passed
	// returned causes a runtime panic.
	Go(f func() error)
	// Gof is like Go, but wraps the returned error with the given name and args
	Gof(f func() error, name string, args ...interface{})
	// GoWithContext is like Go, but passes the given context to f
	GoWithContext(ctx context.Context, f func(ctx context.Context) error)
	// GoWithContextf is like Gof, but passes the given context to f
	GoWithContextf(ctx context.Context, f func(ctx context.Context) error, msg string, args ...interface{})
	// Running returns the RunningCoffin started by this StartingCoffin. It blocks
	// until the callback which got the StartingCoffin passed returned.
	Running() RunningCoffin
}

// A RunningCoffin monitors the execution of tracked go routines.
type RunningCoffin interface {
	// Kill puts the Coffin in the dying state for the given reason and
	// closes the Dying channel.
	//
	// Although Kill may be called multiple times, only the first
	// non-nil error is recorded as the death reason.
	//
	// A tracked go routine returning an error has the same effect as calling
	// Kill with that error as the reason.
	Kill(reason error)
	// Dying returns the channel that can be used to wait until Kill is called.
	Dying() <-chan struct{}
	// Dead returns the channel that can be used to wait until all goroutines
	// have finished running.
	Dead() <-chan struct{}
}

// A Coffin extends a RunningCoffin with the ability to wait for all go routines
// to finish running. The interface is needed to ensure that a go routine can't
// (easily) wait for itself to exit (by calling Running().Wait()).
type Coffin interface {
	RunningCoffin
	// Wait blocks until all goroutines have finished running, and then returns
	// the reason for their death.
	Wait() error
}

type startingCoffin struct {
	startedLck conc.PoisonedLock
	started    sync.WaitGroup
	cfn        *coffin
}

type coffin struct {
	lck    sync.Mutex
	alive  int
	dying  conc.SignalOnce
	dead   conc.SignalOnce
	reason error
	cancel context.CancelFunc // nil -> no associated context or already called, non-nil -> needs to be called when moving to dying
}

func newCoffin(f func(cfn *startingCoffin)) Coffin {
	cfn := &startingCoffin{
		startedLck: conc.NewPoisonedLock(),
		cfn: &coffin{
			dying: conc.NewSignalOnce(),
			dead:  conc.NewSignalOnce(),
		},
	}
	cfn.started.Add(1)
	defer cfn.finishSpawning()

	f(cfn)

	return cfn.cfn
}

func New(f func(cfn StartingCoffin)) Coffin {
	return newCoffin(func(cfn *startingCoffin) {
		f(cfn)
	})
}

// WithContext returns a new Coffin that is killed when the provided parent
// context is canceled, and it passes a copy of parent with a replaced Done
// channel that is closed when either the Coffin is dying or the parent is canceled.
//
// If the context is canceled, the Coffin is killed with the error from the context.
// Thus, you will normally get a context.Canceled error from a Coffin you stop like this.
func WithContext(parent context.Context, f func(cfn StartingCoffin, cfnCtx context.Context)) Coffin {
	return newCoffin(func(cfn *startingCoffin) {
		ctx, cancel := context.WithCancel(parent)
		cfn.cfn.cancel = cancel

		if done := parent.Done(); done != nil {
			cfn.Go(func() error {
				select {
				case <-cfn.cfn.dying.Channel():
				case <-done:
					cfn.cfn.Kill(parent.Err())
				}

				return nil
			})
		}

		f(cfn, ctx)
	})
}

func (c *coffin) Dying() <-chan struct{} {
	return c.dying.Channel()
}

func (c *coffin) Dead() <-chan struct{} {
	return c.dead.Channel()
}

func (c *coffin) Wait() error {
	<-c.Dead()

	return c.reason
}

func (c *startingCoffin) Go(f func() error) {
	c.start(func() {
		defer c.done()
		defer func() {
			panicErr := ResolveRecovery(recover())

			if panicErr != nil {
				c.cfn.Kill(panicErr)
			}
		}()

		if err := f(); err != nil {
			c.cfn.Kill(err)
		}
	})
}

func (c *startingCoffin) Gof(f func() error, msg string, args ...interface{}) {
	c.start(func() {
		defer c.done()
		defer func() {
			panicErr := ResolveRecovery(recover())

			if panicErr != nil {
				c.cfn.Kill(fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), panicErr))
			}
		}()

		if err := f(); err != nil {
			c.cfn.Kill(fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), err))
		}
	})
}

func (c *startingCoffin) GoWithContext(ctx context.Context, f func(ctx context.Context) error) {
	c.Go(func() error {
		return f(ctx)
	})
}

func (c *startingCoffin) GoWithContextf(ctx context.Context, f func(ctx context.Context) error, msg string, args ...interface{}) {
	c.Gof(func() error {
		return f(ctx)
	}, msg, args...)
}

func (c *startingCoffin) Running() RunningCoffin {
	c.started.Wait()

	return c.cfn
}

func (c *coffin) Kill(reason error) {
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.reason != nil {
		// something already killed us
		return
	}

	if c.alive == 0 {
		// we are either already dead (so no need to set a reason now) or no go routine has been started yet
		// so Go was never called and there is no use in setting a death reason
		c.moveToDead()
	} else {
		// we have something running, move to DYING and set the death reason
		c.reason = reason
		c.moveToDying()
	}
}

func (c *startingCoffin) finishSpawning() {
	defer c.started.Done()

	if err := c.startedLck.Poison(); err != nil {
		panic(fmt.Errorf("starting coffin was already poisoned (internal error): %w", err))
	}

	c.cfn.lck.Lock()
	defer c.cfn.lck.Unlock()

	if c.cfn.alive == 0 {
		// nothing was started, we are already dead
		c.cfn.moveToDead()
	}
}

func (c *coffin) moveToDying() {
	c.dying.Signal()

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}

func (c *coffin) moveToDead() {
	c.moveToDying()
	c.dead.Signal()
}

func (c *startingCoffin) start(f func()) {
	if err := c.startedLck.TryLock(); err != nil {
		panic(fmt.Errorf("can not start another go routine after the coffin is no longer starting: %w", err))
	}
	defer c.startedLck.Unlock()

	c.cfn.goStart()
	go f()
}

func (c *coffin) goStart() {
	c.lck.Lock()
	defer c.lck.Unlock()

	// we are either still SPAWNING or at least not DEAD
	c.alive++
}

func (c *startingCoffin) done() {
	// wait until the coffin finished starting
	c.started.Wait()

	c.cfn.lck.Lock()
	defer c.cfn.lck.Unlock()

	c.cfn.alive--
	if c.cfn.alive == 0 {
		c.cfn.moveToDead()
	}
}
