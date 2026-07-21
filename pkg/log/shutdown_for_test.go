package log

import "context"

func setShutdownFn(ctx context.Context, fn func(context.Context) error) {
	c, ok := ctx.Value(logShutdownKey{}).(*shutdownContainer)
	if !ok || c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.fn = fn
}

// ProvideShutdownForTest sets the shutdown function in the context's container.
// If no container exists, it installs one first. Intended for test use only.
func ProvideShutdownForTest(ctx context.Context, fn func(context.Context) error) context.Context {
	if _, ok := ctx.Value(logShutdownKey{}).(*shutdownContainer); !ok {
		ctx = WithShutdownContainer(ctx)
	}

	setShutdownFn(ctx, fn)

	return ctx
}
