package tracing

import (
	"context"
	"errors"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type (
	tracingShutdownKey struct{}
)

var _ kernel.ShutdownHandler = &ShutdownHandler{}

// ProvideShutdownHandler returns the tracing provider shutdown handler from the appctx container.
func ProvideShutdownHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*ShutdownHandler, error) {
	return appctx.Provide(ctx, tracingShutdownKey{}, func() (*ShutdownHandler, error) {
		return &ShutdownHandler{}, nil
	})
}

type ShutdownHandler struct {
	providers []*sdktrace.TracerProvider
}

// AddProvider registers a tracer provider to shut down when the application stops.
func (h *ShutdownHandler) AddProvider(provider *sdktrace.TracerProvider) {
	h.providers = append(h.providers, provider)
}

// Shutdown shuts down all registered tracing providers. If no provider was registered, it is a no-op.
func (h *ShutdownHandler) Shutdown(ctx context.Context) error {
	var err error

	for _, p := range h.providers {
		if sErr := p.Shutdown(ctx); sErr != nil {
			err = errors.Join(err, sErr)
		}
	}

	return err
}
