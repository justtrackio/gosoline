package exec

import (
	"context"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

type DelayedCancelContext struct {
	context.Context
	done     chan struct{}
	stopWg   *sync.WaitGroup
	stopOnce sync.Once
	stop     chan struct{}
}

func (c *DelayedCancelContext) Done() <-chan struct{} {
	return c.done
}

// Stop releases the resources from a delayed cancel context. This causes the delayed context to never be canceled (if it
// wasn't canceled already). Calling stop never returns before all resources have been released, so after Stop returns,
// the context will not experience a delayed cancel anymore.
func (c *DelayedCancelContext) Stop() {
	c.stopOnce.Do(func() {
		close(c.stop)
	})

	c.stopWg.Wait()
}

// WithDelayedCancelContext creates a context which propagates the cancellation of the parent context after a fixed delay
// to the returned context. Call DelayedCancelContext.Stop to release resources associated with the returned context once
// you no longer need it.
func WithDelayedCancelContext(parentCtx context.Context, delay time.Duration) *DelayedCancelContext {
	done := make(chan struct{})
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()

		select {
		case <-stop:
		case <-parentCtx.Done():
			clock.Provider.Sleep(delay)
			close(done)
		}
	}()

	return &DelayedCancelContext{
		Context: parentCtx,
		done:    done,
		stopWg:  wg,
		stop:    stop,
	}
}

type manualCancelContext struct {
	context.Context
	done chan struct{}
}

func (c *manualCancelContext) Done() <-chan struct{} {
	return c.done
}

// WithManualCancelContext is similar to context.WithCancel, but it only cancels the returned context once the cancel
// function has been called. Cancellation of the parent context is not automatically propagated to the child context.
func WithManualCancelContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	done := make(chan struct{})
	once := &sync.Once{}
	cancel := func() {
		once.Do(func() {
			close(done)
		})
	}

	return &manualCancelContext{
		Context: parentCtx,
		done:    done,
	}, cancel
}
