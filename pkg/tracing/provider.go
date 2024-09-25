package tracing

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	ProviderXRay = "xray"
	ProviderOtel = "otel"
)

type (
	TracerProvider       func(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error)
	InstrumentorProvider func(ctx context.Context, config cfg.Config, logger log.Logger) (Instrumentor, error)
)

func AddTracerProvider(name string, provider TracerProvider) {
	tracerProviders[name] = provider
}

func AddInstrumentorProvider(name string, provider TracerProvider) {
	tracerProviders[name] = provider
}

var tracerProviders = map[string]TracerProvider{
	ProviderXRay: NewAwsTracer,
	ProviderOtel: NewOtelTracer,
}

var instrumentorProviders = map[string]InstrumentorProvider{
	ProviderXRay: NewAwsInstrumentor,
	ProviderOtel: NewOtelInstrumentor,
}
