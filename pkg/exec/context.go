package exec

import (
	"context"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

type StopFunc func()

type syncErr struct {
	errLck sync.RWMutex
	err    error
}

func (e *syncErr) load() error {
	e.errLck.RLock()
	defer e.errLck.RUnlock()

	return e.err
}

func (e *syncErr) store(err error) {
	e.errLck.Lock()
	defer e.errLck.Unlock()

	e.err = err
}

type stoppableContext struct {
	parentCtx context.Context
	done      chan struct{}
	err       *syncErr
	stopWg    *sync.WaitGroup
	stopOnce  sync.Once
	stopped   chan struct{}
	deadline  *time.Time
}

func newStoppableContext(parentCtx context.Context, deadline *time.Time, handler func(ctx *stoppableContext) (bool, error)) (context.Context, StopFunc) {
	if parentCtx == nil {
		panic("cannot create context from nil parent")
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx := &stoppableContext{
		parentCtx: parentCtx,
		done:      make(chan struct{}),
		err:       &syncErr{},
		stopWg:    wg,
		stopped:   make(chan struct{}),
		deadline:  deadline,
	}
	if parentDeadline, ok := parentCtx.Deadline(); ok {
		if ctx.deadline == nil || ctx.deadline.After(parentDeadline) {
			ctx.deadline = &parentDeadline
		}
	}

	go func() {
		defer ctx.stopWg.Done()

		shouldClose, err := handler(ctx)
		if err != nil {
			ctx.err.store(err)
		}
		if shouldClose {
			close(ctx.done)
		}
	}()

	return ctx, ctx.stop
}

func (c *stoppableContext) Done() <-chan struct{} {
	return c.done
}

func (c *stoppableContext) Err() error {
	return c.err.load()
}

func (c *stoppableContext) Deadline() (time.Time, bool) {
	if c.deadline != nil {
		return *c.deadline, true
	}

	return time.Time{}, false
}

func (c *stoppableContext) Value(key interface{}) interface{} {
	return c.parentCtx.Value(key)
}

// stop releases the resources from a delayed cancel context. This causes the delayed context to never be canceled (if it
// wasn't canceled already). Calling stop never returns before all resources have been released, so after Stop returns,
// the context will not experience a delayed cancel anymore.
func (c *stoppableContext) stop() {
	c.stopOnce.Do(func() {
		close(c.stopped)
	})

	c.stopWg.Wait()
}

// WithDelayedCancelContext creates a context which propagates the cancellation of the parent context after a fixed delay
// to the returned context. Call the returned StopFunc function to release resources associated with the returned context once
// you no longer need it. Calling stop never returns before all resources have been released, so after Stop returns,
// the context will not experience a delayed cancel anymore (however, if the parent context was already canceled the moment
// you called stop, the child context will immediately get canceled).
func WithDelayedCancelContext(parentCtx context.Context, delay time.Duration) (context.Context, StopFunc) {
	return newStoppableContext(parentCtx, nil, func(ctx *stoppableContext) (bool, error) {
		select {
		case <-ctx.stopped:
			return false, nil
		case <-parentCtx.Done():
			// we used to just call sleep here. but this is not good if you have the following sequence:
			// - parent gets canceled
			// - we sleep to propagate the cancel
			// - we are stopped - so now we have a caller blocking on the sleep call in the end because they are waiting for the waitgroup
			timer := clock.Provider.NewTimer(delay)
			select {
			case <-ctx.stopped:
				timer.Stop()
			case <-timer.Chan():
				// nop
			}

			return true, parentCtx.Err()
		}
	})
}

// WithStoppableDeadlineContext is similar to context.WithDeadline. However, while context.WithDeadline cancels the context when
// you call the returned context.CancelFunc, WithStoppableDeadlineContext does not cancel the context if it is not yet canceled
// once you stop it.
func WithStoppableDeadlineContext(parentCtx context.Context, deadline time.Time) (context.Context, StopFunc) {
	return newStoppableContext(parentCtx, &deadline, func(ctx *stoppableContext) (bool, error) {
		c := clock.Provider
		waitTime := -c.Since(deadline)
		timer := c.NewTimer(waitTime)
		defer timer.Stop()

		select {
		case <-ctx.stopped:
			return false, nil
		case <-parentCtx.Done():
			return true, parentCtx.Err()
		case <-timer.Chan():
			return true, context.DeadlineExceeded
		}
	})
}

type manualCancelContext struct {
	context.Context
	done chan struct{}
	err  *syncErr
}

func (c *manualCancelContext) Done() <-chan struct{} {
	return c.done
}

func (c *manualCancelContext) Err() error {
	return c.err.load()
}

// WithManualCancelContext is similar to context.WithCancel, but it only cancels the returned context once the cancel
// function has been called. Cancellation of the parent context is not automatically propagated to the child context.
func WithManualCancelContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	ctx := &manualCancelContext{
		Context: parentCtx,
		done:    make(chan struct{}),
		err:     &syncErr{},
	}
	once := &sync.Once{}
	cancel := func() {
		once.Do(func() {
			ctx.err.store(context.Canceled)
			close(ctx.done)
		})
	}

	return ctx, cancel
}
