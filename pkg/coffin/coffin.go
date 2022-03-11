package coffin

import (
	"context"
	"fmt"
	"sync"
)

// A Coffin manages the execution of multiple go routines. A Coffin has
// three different states:
//
//                        +-----------+
//                        |  SPAWNING |-----------------
//                        +-----------+                |
//                              |                      |
//                              |  Wait, Dying, Dead   |
//                              |                      |
//                              v                      |
//                        +-----------+                |
//                        |  WAITING  |                |  Kill, go routine
//                        +-----------+                |  returned an error
//                              |                      |
//                              |  Kill, go routine    |
//                              |  returned an error   |
//                              |                      |
//                              v                      |
//                        +-----------+                |
//                        |   DYING   |<----------------
//                        +-----------+
//                              |
//                              |  Last go routine exited
//                              |
//                              v
//                        +-----------+
//                        |    DEAD   |
//                        +-----------+
//
// A Coffin initially starts in the SPAWNING state. In this state you can spawn
// as many go routines as you want and as long as they don't return an error,
// they can finish running freely. Calling any of Wait, Kill, or Dead will
// put a coffin in the WAITING state. In this state you can still spawn new
// tasks, but the Coffin will move to the DEAD state as soon as the last go
// routine finishes running. As soon as a Coffin is in the DEAD state, you
// can't spawn new go routines (those calls will be ignored).
//
// A go routine which returns an error or panics is equivalent to a call to
// Kill and will put the Coffin in the AWAITED state (if it is still in the
// ALIVE state) or the DEAD state (if the last go routine returned said error
// or panicked).
type Coffin interface {
	// Go runs f in a new goroutine and tracks its termination.
	//
	// If f returns a non-nil error, Kill is called with that error as the
	// death reason parameter.
	//
	// It is f's responsibility to monitor the coffin and return
	// appropriately once it is in a dying state.
	//
	// As long as the Coffin is not in the DEAD state, you can call
	// Go again to spawn additional go routines. Once the Coffin
	// reaches the DEAD state, calls to Go cause a runtime panic.
	//
	// Go returns true if it managed to start the go routine successfully
	// and false if the Coffin was already in the DEAD state.
	//
	// If all tracked go routines return and the Coffin is not in the
	// SPAWNING state (i.e., Wait, Dying, and Dead have at least once been
	// called), it moves to the DEAD state. Thus, it is only safe to call
	// Go outside a tracked go routine if you can guarantee that nobody
	// calls Kill, Dying, Dead, Wait, or returns an error from a tracked
	// go routine.
	Go(f func() error)
	// Gof is like Go, but wraps the returned error with the given name and args
	Gof(f func() error, name string, args ...interface{})
	// GoWithContext is like Go, but passes the given context to f
	GoWithContext(ctx context.Context, f func(ctx context.Context) error)
	// GoWithContextf is like Gof, but passes the given context to f
	GoWithContextf(ctx context.Context, f func(ctx context.Context) error, msg string, args ...interface{})
	// Kill puts the coffin in the DYING state for the given reason and
	// closes the Dying channel.
	//
	// Although Kill may be called multiple times, only the first
	// non-nil error is recorded as the death reason.
	//
	// A tracked go routine returning an error has the same effect as calling
	// Kill with that error as the reason.
	Kill(reason error)
	// Dying returns the channel that can be used to wait until Kill is called.
	//
	// If no go routines are running (anymore or never where), Dying immediately
	// moves the Coffin into the DEAD state and returns a closed channel.
	Dying() <-chan struct{}
	// Dead returns the channel that can be used to wait until
	// all goroutines have finished running.
	//
	// If no go routines are running (anymore or never where), Dead immediately
	// moves the Coffin into the DEAD state and returns a closed channel.
	Dead() <-chan struct{}
	// Wait blocks until all goroutines have finished running, and
	// then returns the reason for their death.
	//
	// If no go routines are running (anymore or never where), Wait immediately
	// moves the Coffin into the DEAD state and returns nil.
	Wait() error
	// Spawn allows you to safely add some go routines to a Coffin. If it is safe
	// to call Go on the passed Coffin, this will remain true for the duration of
	// the callback, regardless of the spawned functions.
	//
	//     cfn := New()
	//     cfn.Spawn(func(cfn Coffin) {
	//         cfn.Go(func() error {
	//             return fmt.Errorf("Oh no, I am killing the coffin")
	//         })
	//         // do something else...
	//         cfn.Go(...) // this call will never panic thanks to Spawn,
	//                     // without it, it might (depending on the go scheduler)
	//     })
	Spawn(func(cfn Coffin))
}

type coffin struct {
	lck     sync.Mutex
	alive   int
	spawned bool          // false -> SPAWNING; true -> AWAITING, DYING, or DEAD
	dying   chan struct{} // nil/open -> SPAWNING or AWAITING; closed -> DYING or DEAD
	dead    chan struct{} // nil/open -> SPAWNING, AWAITING, or DYING; closed -> DEAD
	reason  error
	cancel  context.CancelFunc // nil -> no associated context or already called, non-nil -> needs to be called when DYING
}

func New() Coffin {
	return new(coffin)
}

// WithContext returns a new Coffin that is killed when the provided parent
// context is canceled, and a copy of parent with a replaced Done channel
// that is closed when either the Coffin is dying or the parent is canceled.
//
// If the context is canceled, the Coffin is killed with the error from the context.
// Thus, you will normally get a context.Canceled error from a Coffin you stop like this.
func WithContext(parent context.Context) (Coffin, context.Context) {
	cfn := new(coffin)
	// create the dying channel manually to avoid moving to WAITING or DYING immediately
	cfn.dying = make(chan struct{})
	ctx, cancel := context.WithCancel(parent)
	cfn.cancel = cancel

	if done := parent.Done(); done != nil {
		go func() {
			select {
			case <-cfn.dying:
			case <-done:
				cfn.Kill(parent.Err())
			}
		}()
	}

	return cfn, ctx
}

func (c *coffin) Dying() <-chan struct{} {
	c.finishSpawning()

	return c.dying
}

func (c *coffin) Dead() <-chan struct{} {
	c.finishSpawning()

	return c.dead
}

func (c *coffin) Wait() error {
	<-c.Dead()

	return c.reason
}

func (c *coffin) Go(f func() error) {
	c.goIfNotDead(func() {
		defer c.goDone()
		defer func() {
			panicErr := ResolveRecovery(recover())

			if panicErr != nil {
				c.Kill(panicErr)
			}
		}()

		if err := f(); err != nil {
			c.Kill(err)
		}
	})
}

func (c *coffin) Gof(f func() error, msg string, args ...interface{}) {
	c.goIfNotDead(func() {
		defer c.goDone()
		defer func() {
			panicErr := ResolveRecovery(recover())

			if panicErr != nil {
				c.Kill(fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), panicErr))
			}
		}()

		if err := f(); err != nil {
			c.Kill(fmt.Errorf("%s: %w", fmt.Sprintf(msg, args...), err))
		}
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
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.reason != nil {
		// something already killed us
		return
	}

	c.initLocked()
	if c.alive == 0 {
		// we are either already DEAD (so no need to set a reason now) or where still in SPAWNING with nothing running,
		// so Go was never called and there is no use in setting a death reason
		c.moveToDead()
	} else {
		// we have something running, move to DYING and set the death reason
		c.reason = reason
		c.moveToDying()
	}
}

func (c *coffin) Spawn(spawner func(cfn Coffin)) {
	// emulate a running go routine without actually running one
	c.goStart()
	defer c.goDone()

	spawner(c)
}

func (c *coffin) initLocked() {
	if c.dying == nil {
		c.dying = make(chan struct{})
	}
	if c.dead == nil {
		c.dead = make(chan struct{})
	}
}

func (c *coffin) finishSpawning() {
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.spawned {
		return
	}

	c.spawned = true
	c.initLocked()
	if c.alive != 0 {
		return
	}

	// directly move from SPAWNING to DEAD

	c.moveToDead()
}

func (c *coffin) moveToDying() {
	c.spawned = true
	closeOnce(c.dying)

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}

func (c *coffin) moveToDead() {
	c.moveToDying()
	closeOnce(c.dead)
}

func (c *coffin) goIfNotDead(f func()) {
	c.goStart()
	go f()
}

func (c *coffin) goStart() {
	c.lck.Lock()
	defer c.lck.Unlock()

	if c.spawned {
		select {
		case <-c.dead:
			// we are already DEAD, this is a panic
			panic(withStack(fmt.Errorf("can't call Go in state DEAD")))
		default:
		}
	}

	// we are either still SPAWNING or at least not DEAD
	c.alive++
}

func (c *coffin) goDone() {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.alive--
	if c.alive > 0 || !c.spawned {
		// we are either in SPAWNING or have still some go routines running
		return
	}

	c.initLocked()
	c.moveToDead()
}

func closeOnce(c chan struct{}) {
	select {
	case <-c:
	default:
		close(c)
	}
}
