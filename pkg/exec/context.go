package exec

import (
	"context"
	"time"
)

type delayedCancelContext struct {
	context.Context
	done chan struct{}
	stop chan struct{}
}

func (c *delayedCancelContext) Done() <-chan struct{} {
	return c.done
}

func (c *delayedCancelContext) Stop() {
	close(c.stop)
}

func WithDelayedCancelContext(parentCtx context.Context, delay time.Duration) *delayedCancelContext {
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

	return &delayedCancelContext{
		Context: parentCtx,
		done:    done,
		stop:    stop,
	}
}
