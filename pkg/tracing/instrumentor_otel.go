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
	"google.golang.org/grpc/stats"
)

func init() {
	AddInstrumentorProvider(ProviderOtel, NewOtelInstrumentor)
}

type otelInstrumentor struct {
	cfg.AppId
}

func NewOtelInstrumentor(ctx context.Context, config cfg.Config, logger log.Logger) (Instrumentor, error) {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	// used to set the global trace provider and text map propagator.
	_, err := ProvideOtelTraceProvider(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	return NewOtelInstrumentorWithAppId(appId), nil
}

func NewOtelInstrumentorWithAppId(appId cfg.AppId) *otelInstrumentor {
	return &otelInstrumentor{
		AppId: appId,
	}
}

func (t *otelInstrumentor) HttpHandler(h http.Handler) http.Handler {
	name := fmt.Sprintf("%s-%s-%s-%s-%s", t.Project, t.Environment, t.Family, t.Group, t.Application)
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)

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

func (t *otelInstrumentor) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return nil
}

func (t *otelInstrumentor) GrpcServerHandler() stats.Handler {
	return otelgrpc.NewServerHandler(
		otelgrpc.WithFilter(filters.HealthCheck()),
	)
}
