package tracing

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	AddTracerProvider(ProviderNoop, func(context.Context, cfg.Config, log.Logger) (Tracer, error) {
		return NewNoopTracer(), nil
	})
}

type noopTracer struct{}

func NewNoopTracer() Tracer {
	return noopTracer{}
}

func (t noopTracer) StartSubSpan(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, disabledSpan()
}

func (t noopTracer) StartSpan(_ string) (context.Context, Span) {
	return context.Background(), disabledSpan()
}

func (t noopTracer) StartSpanFromContext(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, disabledSpan()
}
