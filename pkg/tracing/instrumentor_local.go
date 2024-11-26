package tracing

import (
	"context"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func init() {
	AddInstrumentorProvider(ProviderLocal, func(context.Context, cfg.Config, log.Logger) (Instrumentor, error) {
		return NewLocalInstrumentor(), nil
	})
}

var _ Instrumentor = &localInstrumentor{}

type localInstrumentor struct {
	traceIdSource uuid.Uuid
}

func NewLocalInstrumentor() Instrumentor {
	return &localInstrumentor{
		traceIdSource: uuid.New(),
	}
}

func (t localInstrumentor) HttpHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if trace, err := StringToTrace(r.Header.Get(xray.TraceIDHeaderKey)); err == nil {
			ctx = ContextWithTrace(ctx, trace)
		} else {
			ctx = t.ensureLocalTrace(ctx)
		}

		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

func (t localInstrumentor) HttpClient(baseClient *http.Client) *http.Client {
	return baseClient
}

func (t localInstrumentor) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		ctx = t.ensureLocalTrace(ctx)

		return handler(ctx, req)
	}
}

func (t localInstrumentor) GrpcServerHandler() stats.Handler {
	return nil
}

func (t localInstrumentor) ensureLocalTrace(ctx context.Context) context.Context {
	if trace := GetTraceFromContext(ctx); trace != nil {
		return ctx
	}

	return contextWithLocalTraceId(ctx, t.traceIdSource.NewV4())
}
