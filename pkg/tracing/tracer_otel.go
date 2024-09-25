package tracing

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
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

	return NewOtelTracerWithInterfaces(tracer), nil
}

func NewOtelTracerWithInterfaces(tracer trace.Tracer) *otelTracer {
	return &otelTracer{
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

func (t *otelTracer) spanFromTrace(ctx context.Context, trc *Trace, name string) (context.Context, Span) {
	var tFlags trace.TraceFlags
	if trc.GetSampled() {
		tFlags = trace.FlagsSampled
	}

	tID, err := trace.TraceIDFromHex(trc.GetTraceId())
	if err != nil {
		tID = trace.TraceID{}
	}
	sID, err := trace.SpanIDFromHex(trc.GetId())
	if err != nil {
		sID = trace.SpanID{}
	}

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tID,
		SpanID:     sID,
		TraceFlags: tFlags,
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, spanCtx)

	return t.StartSubSpan(ctx, name)
}
