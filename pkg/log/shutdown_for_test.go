package log

import "context"

// ProvideShutdownForTest sets the shutdown function in the context's container.
// If no container exists, it installs one first. Intended for test use only.
func ProvideShutdownForTest(ctx context.Context, fn func(context.Context) error) context.Context {
	if _, ok := ctx.Value(logShutdownKey{}).(*shutdownContainer); !ok {
		ctx = WithShutdownContainer(ctx)
	}

	setShutdownFn(ctx, fn)

	return ctx
}
