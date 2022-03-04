package exec

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

type StopFunc func()

type stoppableContext struct {
	parentCtx context.Context
	done      chan struct{}
	err       atomic.Value
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
		err:       atomic.Value{},
		done:      make(chan struct{}),
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
			ctx.err.Store(err)
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
	err := c.err.Load()
	if err == nil {
		return nil
	}

	return err.(error)
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
// the context will not experience a delayed cancel anymore.
func WithDelayedCancelContext(parentCtx context.Context, delay time.Duration) (context.Context, StopFunc) {
	return newStoppableContext(parentCtx, nil, func(ctx *stoppableContext) (bool, error) {
		select {
		case <-ctx.stopped:
			return false, nil
		case <-parentCtx.Done():
			clock.Provider.Sleep(delay)
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
	err  atomic.Value
}

func (c *manualCancelContext) Done() <-chan struct{} {
	return c.done
}

func (c *manualCancelContext) Err() error {
	err := c.err.Load()
	if err == nil {
		return nil
	}

	return err.(error)
}

// WithManualCancelContext is similar to context.WithCancel, but it only cancels the returned context once the cancel
// function has been called. Cancellation of the parent context is not automatically propagated to the child context.
func WithManualCancelContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	ctx := &manualCancelContext{
		Context: parentCtx,
		done:    make(chan struct{}),
		err:     atomic.Value{},
	}
	once := &sync.Once{}
	cancel := func() {
		once.Do(func() {
			ctx.err.Store(context.Canceled)
			close(ctx.done)
		})
	}

	return ctx, cancel
}
