package tracing

import (
	"context"
	"net/http"

	"google.golang.org/grpc"
)

type noopInstrumentor struct{}

func NewNoopInstrumentor() Instrumentor {
	return &noopInstrumentor{}
}

func (t *noopInstrumentor) HttpHandler(h http.Handler) http.Handler {
	return h
}

func (t *noopInstrumentor) HttpClient(baseClient *http.Client) *http.Client {
	return baseClient
}

func (t *noopInstrumentor) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(ctx, req)
	}
}
