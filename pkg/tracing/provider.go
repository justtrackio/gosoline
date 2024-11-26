package tracing

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	TracerProvider       func(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error)
	InstrumentorProvider func(ctx context.Context, config cfg.Config, logger log.Logger) (Instrumentor, error)
)

func AddTracerProvider(name string, provider TracerProvider) {
	tracerProviders[name] = provider
}

func AddInstrumentorProvider(name string, provider InstrumentorProvider) {
	instrumentorProviders[name] = provider
}

var tracerProviders = map[string]TracerProvider{}

var instrumentorProviders = map[string]InstrumentorProvider{}
