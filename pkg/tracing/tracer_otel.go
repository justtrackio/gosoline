package tracing

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	AddTracerProvider(ProviderOtel, NewOtelTracer)
}

const (
	instrumentationName    = "https://github.com/justtrackio/gosoline"
	instrumentationVersion = "v0.8.0"
)

type otelTracer struct {
	logger log.Logger
	tracer trace.Tracer
}

func NewOtelTracer(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
	logger = logger.WithChannel("tracing")

	traceProvider, err := ProvideOtelTraceProvider(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	tracer := traceProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(instrumentationVersion),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

	return NewOtelTracerWithInterfaces(logger, tracer), nil
}

func NewOtelTracerWithInterfaces(logger log.Logger, tracer trace.Tracer) *otelTracer {
	return &otelTracer{
		logger: logger,
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

	// The Trace ID is expected to be compliant with the W3C trace-context specification. If it is not
	// an empty traceID will be used for the new span.
	tID, err := trace.TraceIDFromHex(trc.GetTraceId())
	if err != nil {
		t.logger.Warn(ctx, "could not parse trace id %s", err.Error())
		tID = trace.TraceID{}
	}

	// The Span ID is expected to be compliant with the W3C trace-context specification. If it is not
	// an empty spanID will be used for the new span.
	sID, err := trace.SpanIDFromHex(trc.GetId())
	if err != nil {
		t.logger.Warn(ctx, "could not parse span id %s", err.Error())
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
