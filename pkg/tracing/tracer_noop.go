package tracing

import (
	"context"
)

var _ Tracer = &noopTracer{}

type noopTracer struct{}

func NewNoopTracer() Tracer {
	return new(noopTracer)
}

func (t *noopTracer) StartSubSpan(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, disabledSpan()
}

func (t *noopTracer) StartSpan(string) (context.Context, Span) {
	return context.Background(), disabledSpan()
}

func (t *noopTracer) StartSpanFromContext(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, disabledSpan()
}
