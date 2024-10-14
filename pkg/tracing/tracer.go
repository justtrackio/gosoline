package tracing

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Tracer
type Tracer interface {
	StartSpan(name string) (context.Context, Span)
	StartSpanFromContext(ctx context.Context, name string) (context.Context, Span)
	StartSubSpan(ctx context.Context, name string) (context.Context, Span)
}

type TracerSettings struct {
	Provider string `cfg:"provider"`
	Enabled  bool   `cfg:"enabled" default:"false"`
}

type appCtxKey string

func ProvideTracer(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	return appctx.Provide(ctx, appCtxKey("tracer"), func() (Tracer, error) {
		return newTracer(ctx, config, logger)
	})
}

func ProvideInstrumentor(ctx context.Context, config cfg.Config, logger log.Logger) (Instrumentor, error) {
	return appctx.Provide(ctx, appCtxKey("instrumentor"), func() (Instrumentor, error) {
		return newInstrumentor(ctx, config, logger)
	})
}

func newTracer(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	settings := &TracerSettings{}
	config.UnmarshalKey("tracing", settings)

	if !settings.Enabled {
		return NewNoopTracer(), nil
	}

	if _, ok := tracerProviders[settings.Provider]; !ok {
		return nil, fmt.Errorf("no tracing provider found for name %s", settings.Provider)
	}

	provider := tracerProviders[settings.Provider]

	return provider(ctx, config, logger)
}

func newInstrumentor(ctx context.Context, config cfg.Config, logger log.Logger) (Instrumentor, error) {
	settings := &TracerSettings{}
	config.UnmarshalKey("tracing", settings)

	if !settings.Enabled {
		return NewNoopInstrumentor(), nil
	}

	if _, ok := instrumentorProviders[settings.Provider]; !ok {
		return nil, fmt.Errorf("no tracing provider found for name %s", settings.Provider)
	}

	provider := instrumentorProviders[settings.Provider]

	return provider(ctx, config, logger)
}
