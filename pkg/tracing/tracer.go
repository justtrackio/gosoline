package tracing

import (
	"context"
	"net/http"
)

//go:generate mockery -name=Tracer
type Tracer interface {
	HttpHandler(h http.Handler) http.Handler
	StartSpan(name string) (context.Context, Span)
	StartSpanFromContext(ctx context.Context, name string) (context.Context, Span)
	StartSpanFromTraceAble(obj TraceAble, name string) (context.Context, Span)
	StartSubSpan(ctx context.Context, name string) (context.Context, Span)
}
