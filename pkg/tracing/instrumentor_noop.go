package tracing

import (
	"context"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func init() {
	AddInstrumentorProvider(ProviderNoop, func(context.Context, cfg.Config, log.Logger) (Instrumentor, error) {
		return NewNoopInstrumentor(), nil
	})
}

type noopInstrumentor struct{}

func NewNoopInstrumentor() Instrumentor {
	return &noopInstrumentor{}
}

func (t noopInstrumentor) HttpHandler(h http.Handler) http.Handler {
	return h
}

func (t noopInstrumentor) HttpClient(baseClient *http.Client) *http.Client {
	return baseClient
}

func (t noopInstrumentor) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		return handler(ctx, req)
	}
}

func (t noopInstrumentor) GrpcServerHandler() stats.Handler {
	return nil
}
