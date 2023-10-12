package tracing

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
)

var _ Tracer = &noopTracer{}

type noopTracer struct{}

func NewNoopTracer() Tracer {
	return new(noopTracer)
}

func (t *noopTracer) StartSubSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, disabledSpan()
}

func (t *noopTracer) StartSpan(name string) (context.Context, Span) {
	return context.Background(), disabledSpan()
}

func (t *noopTracer) StartSpanFromContext(ctx context.Context, name string) (context.Context, Span) {
	return ctx, disabledSpan()
}

func (t *noopTracer) HttpHandler(h http.Handler) http.Handler {
	return h
}

func (t *noopTracer) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(ctx, req)
	}
}
