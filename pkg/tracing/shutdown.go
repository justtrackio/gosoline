package tracing

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/kernel"
)

type tracingShutdownKey struct{}

// NewShutdownHandler returns a ShutdownHandler that retrieves the tracing provider's
// shutdown function from the appctx container and invokes it.
func NewShutdownHandler() kernel.ShutdownHandler {
	return shutdownHandler{}
}

type shutdownHandler struct{}

var _ kernel.ShutdownHandler = shutdownHandler{}

// Shutdown retrieves the registered tracing provider shutdown function from the appctx
// container. If no provider was registered, it is a no-op.
func (shutdownHandler) Shutdown(ctx context.Context) error {
	shutdownFn, err := appctx.Get[func(context.Context) error](ctx, tracingShutdownKey{})
	if err != nil {
		return nil
	}

	return shutdownFn(ctx)
}

// ProvideShutdownForTest stores a shutdown function in the container for testing.
// Intended for test use only.
func ProvideShutdownForTest(ctx context.Context, fn func(context.Context) error) {
	appctx.Provide(ctx, tracingShutdownKey{}, func() (func(context.Context) error, error) { //nolint:errcheck // test helper, factory cannot fail
		return fn, nil
	})
}
