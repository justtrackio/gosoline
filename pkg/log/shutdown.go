package log

import (
	"context"
	"sync"
)

func setShutdownFn(ctx context.Context, fn func(context.Context) error) {
	c, ok := ctx.Value(logShutdownKey{}).(*shutdownContainer)
	if !ok || c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.fn = fn
}

type logShutdownKey struct{}

type shutdownContainer struct {
	mu sync.Mutex
	fn func(context.Context) error
}

// WithShutdownContainer returns a new context with an empty shutdown container installed.
// This must be called before NewHandlersFromConfig so that handler factories can register
// their shutdown functions.
func WithShutdownContainer(ctx context.Context) context.Context {
	return context.WithValue(ctx, logShutdownKey{}, &shutdownContainer{})
}

// ShutdownHandler drains the registered log backend shutdown function. It is implemented
// by the value returned from NewShutdownHandler and consumed by the kernel, which runs it
// after emitting the final exit-code log line.
type ShutdownHandler interface {
	Shutdown(ctx context.Context) error
}

// NewShutdownHandler returns a ShutdownHandler that retrieves the log provider's
// shutdown function from the context and invokes it.
func NewShutdownHandler() ShutdownHandler {
	return shutdownHandler{}
}

type shutdownHandler struct{}

// Shutdown retrieves the registered log provider shutdown function from the context.
// If no provider was registered, it is a no-op.
func (shutdownHandler) Shutdown(ctx context.Context) error {
	c, ok := ctx.Value(logShutdownKey{}).(*shutdownContainer)
	if !ok || c == nil {
		return nil
	}

	c.mu.Lock()
	fn := c.fn
	c.mu.Unlock()

	if fn == nil {
		return nil
	}

	return fn(ctx)
}
