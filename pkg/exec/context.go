package exec

import (
	"context"
	"time"
)

type DelayedCancelContext struct {
	context.Context
	done chan struct{}
	stop chan struct{}
}

func (c *DelayedCancelContext) Done() <-chan struct{} {
	return c.done
}

func (c *DelayedCancelContext) Stop() {
	close(c.stop)
}

func WithDelayedCancelContext(parentCtx context.Context, delay time.Duration) *DelayedCancelContext {
	done := make(chan struct{})
	stop := make(chan struct{})

	go func() {
		select {
		case <-stop:
		case <-parentCtx.Done():
			time.Sleep(delay)
			close(done)
		}
	}()

	return &DelayedCancelContext{
		Context: parentCtx,
		done:    done,
		stop:    stop,
	}
}
