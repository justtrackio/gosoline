package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp/filters"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	instrumentationName    = "https://github.com/justtrackio/gosoline"
	instrumentationVersion = "v0.8.0"
)

var _ Tracer = &otelTracer{}

type OtelSettings struct {
	Exporter      string  `cfg:"exporter"`
	SamplingRatio float64 `cfg:"sampling_ratio" default:"0.05"`
	SpanLimits
}

type SpanLimits struct {
	AttributeValueLengthLimit   int `cfg:"attribute_value_length_limit" default:"-1"`
	AttributeCountLimit         int `cfg:"attribute_count_limit" default:"128"`
	EventCountLimit             int `cfg:"event_count_limit" default:"128"`
	LinkCountLimit              int `cfg:"link_count_limit" default:"128"`
	AttributePerEventCountLimit int `cfg:"attribute_per_event_count_limit" default:"128"`
	AttributePerLinkCountLimit  int `cfg:"attribute_per_link_count_limit" default:"128"`
}

type otelTracer struct {
	cfg.AppId
	tracer trace.Tracer
}

func NewOtelTracer(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	settings := &OtelSettings{}
	config.UnmarshalKey("tracing.otel", settings)

	otelExporterFactory := TraceExporters[settings.Exporter]

	exporter, err := otelExporterFactory(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(fmt.Sprintf("%v-%v-%v-%v", appId.Project, appId.Environment, appId.Family, appId.Application)),
		)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(settings.SamplingRatio))),
		sdktrace.WithRawSpanLimits(sdktrace.SpanLimits{
			AttributeValueLengthLimit:   settings.AttributeValueLengthLimit,
			AttributeCountLimit:         settings.AttributeCountLimit,
			EventCountLimit:             settings.EventCountLimit,
			LinkCountLimit:              settings.LinkCountLimit,
			AttributePerEventCountLimit: settings.AttributePerEventCountLimit,
			AttributePerLinkCountLimit:  settings.AttributePerLinkCountLimit,
		}),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := otel.GetTracerProvider().Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	return NewOtelTracerWithInterfaces(appId, tracer), nil
}

func NewOtelTracerWithInterfaces(appId cfg.AppId, tracer trace.Tracer) *otelTracer {
	return &otelTracer{
		AppId:  appId,
		tracer: tracer,
	}
}

func (t *otelTracer) StartSubSpan(ctx context.Context, name string) (context.Context, Span) {
	ctx, span := t.tracer.Start(ctx, name)

	return newOtelSpan(ctx, span)
}

func (t *otelTracer) StartSpan(name string) (context.Context, Span) {
	return t.StartSubSpan(context.Background(), name)
}

func (t *otelTracer) StartSpanFromContext(ctx context.Context, name string) (context.Context, Span) {
	if parentSpan := GetSpanFromContext(ctx); parentSpan != nil {
		parentTrace := parentSpan.GetTrace()
		return t.spanFromTrace(ctx, parentTrace, name)
	}

	if ctxTrace := GetTraceFromContext(ctx); ctxTrace != nil {
		return t.spanFromTrace(ctx, ctxTrace, name)
	}

	return t.StartSubSpan(ctx, name)
}

func (t *otelTracer) HttpHandler(h http.Handler) http.Handler {
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

func (t *otelTracer) HttpClient(baseClient *http.Client) *http.Client {
	return &http.Client{
		Transport:     otelhttp.NewTransport(baseClient.Transport),
		CheckRedirect: baseClient.CheckRedirect,
		Jar:           baseClient.Jar,
		Timeout:       baseClient.Timeout,
	}
}

// GrpcUnaryServerInterceptor we still need to use the UnaryServerInterceptor because to maintain
// because the Xray is also uses the UnaryServerInterceptor.
//
//nolint:staticcheck
func (t *otelTracer) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return otelgrpc.UnaryServerInterceptor(otelgrpc.WithInterceptorFilter(
		filters.Not(
			filters.HealthCheck(),
		),
	))
}

func (t *otelTracer) spanFromTrace(ctx context.Context, trc *Trace, name string) (context.Context, Span) {
	var tFlags trace.TraceFlags
	if trc.GetSampled() {
		tFlags = trace.FlagsSampled
	}

	tID, _ := trace.TraceIDFromHex(trc.GetTraceId())
	sID, _ := trace.SpanIDFromHex(trc.GetId())

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tID,
		SpanID:     sID,
		TraceFlags: tFlags,
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, spanCtx)

	return t.StartSubSpan(ctx, name)
}
