package tracing

import (
	"context"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	AddProvider("noop", func(config cfg.Config, logger log.Logger) (Tracer, error) {
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

func (t noopTracer) HttpHandler(h http.Handler) http.Handler {
	return h
}
