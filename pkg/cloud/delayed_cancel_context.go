package cloud

import (
	"context"
	"time"
)

type StoppableContext interface {
	context.Context
	// Stop the go routine waiting for the context being done (and then delaying
	// the cancel) and free its resources. Do not use the context afterwards, it
	// will never be canceled (if you don't cancel it yourself)
	Stop()
}

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

func WithDelayedCancelContext(parentCtx context.Context, delay time.Duration) StoppableContext {
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
