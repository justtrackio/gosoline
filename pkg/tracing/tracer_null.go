package tracing

import (
	"context"
	"net/http"
)

type noopTracer struct{}

func NewNoopTracer() Tracer {
	return new(noopTracer)
}

func (t *noopTracer) StartSubSpan(ctx context.Context, name string) (context.Context, Span) {
	return context.Background(), disabledSpan()
}

func (t *noopTracer) StartSpan(name string) (context.Context, Span) {
	return context.Background(), disabledSpan()
}

func (t *noopTracer) StartSpanFromContext(ctx context.Context, name string) (context.Context, Span) {
	return context.Background(), disabledSpan()
}

func (t *noopTracer) StartSpanFromTraceAble(obj TraceAble, name string) (context.Context, Span) {
	return context.Background(), disabledSpan()
}

func (t *noopTracer) HttpHandler(h http.Handler) http.Handler {
	return h
}
