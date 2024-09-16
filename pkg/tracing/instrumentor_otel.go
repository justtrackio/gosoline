package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc/filters"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type otelInstrumentor struct {
	cfg.AppId
}

func NewOtelInstrumentor(_ context.Context, config cfg.Config, _ log.Logger) (Instrumentor, error) {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	return NewOtelInstrumentorWithAppId(appId), nil
}

func NewOtelInstrumentorWithAppId(appId cfg.AppId) *otelInstrumentor {
	return &otelInstrumentor{
		AppId: appId,
	}
}

func (t *otelInstrumentor) HttpHandler(h http.Handler) http.Handler {
	name := fmt.Sprintf("%v-%v-%v-%v", t.Project, t.Environment, t.Family, t.Application)
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.SpanFromContext(r.Context())

		ctx, _ = newOtelSpan(ctx, span)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})

	return otelhttp.NewHandler(handlerFunc, name)
}

func (t *otelInstrumentor) HttpClient(baseClient *http.Client) *http.Client {
	return &http.Client{
		Transport:     otelhttp.NewTransport(baseClient.Transport),
		CheckRedirect: baseClient.CheckRedirect,
		Jar:           baseClient.Jar,
		Timeout:       baseClient.Timeout,
	}
}

// GrpcUnaryServerInterceptor returns a grpc.UnaryServerInterceptor instead of the recommended stats.Handler because
// we want to be compatible with the Xray instrumentor implementation.
//
//nolint:staticcheck
func (t *otelInstrumentor) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return otelgrpc.UnaryServerInterceptor(
		otelgrpc.WithInterceptorFilter(
			filters.HealthCheck(),
		),
	)
}
