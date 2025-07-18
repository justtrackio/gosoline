package tracing

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate go run github.com/vektra/mockery/v2 --name Tracer
type Tracer interface {
	StartSpan(name string) (context.Context, Span)
	StartSpanFromContext(ctx context.Context, name string) (context.Context, Span)
	StartSubSpan(ctx context.Context, name string) (context.Context, Span)
}

type TracerSettings struct {
	Provider string `cfg:"provider"  default:"local" validate:"required"`
}

type (
	tracerKey       struct{}
	instrumentorKey struct{}
)

func ProvideTracer(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	return appctx.Provide(ctx, tracerKey{}, func() (Tracer, error) {
		return newTracer(ctx, config, logger)
	})
}

func ProvideInstrumentor(ctx context.Context, config cfg.Config, logger log.Logger) (Instrumentor, error) {
	return appctx.Provide(ctx, instrumentorKey{}, func() (Instrumentor, error) {
		return newInstrumentor(ctx, config, logger)
	})
}

func newTracer(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	var provider TracerProvider
	var ok bool

	settings := &TracerSettings{}
	if err := config.UnmarshalKey("tracing", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracer settings: %w", err)
	}

	if provider, ok = tracerProviders[settings.Provider]; !ok {
		return nil, fmt.Errorf(
			"no tracing provider found for name %s, available providers: %s",
			settings.Provider,
			strings.Join(funk.Keys(tracerProviders), ", "),
		)
	}

	return provider(ctx, config, logger)
}

func newInstrumentor(ctx context.Context, config cfg.Config, logger log.Logger) (Instrumentor, error) {
	settings := &TracerSettings{}
	if err := config.UnmarshalKey("tracing", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracer settings: %w", err)
	}

	provider, ok := instrumentorProviders[settings.Provider]
	if !ok {
		return nil, fmt.Errorf(
			"no tracing providers found for name %s, available providers: %s",
			settings.Provider,
			strings.Join(funk.Keys(instrumentorProviders), ", "),
		)
	}

	return provider(ctx, config, logger)
}
