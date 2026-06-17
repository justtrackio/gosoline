package tracing

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/otel"
	otelglobal "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	PropagatorTraceContext = "tracecontext"
	PropagatorBaggage      = "baggage"
)

type OtelSettings struct {
	Exporter      string   `cfg:"exporter" default:"otel_http"`
	SamplingRatio float64  `cfg:"sampling_ratio" default:"0.05"`
	Propagators   []string `cfg:"propagators" default:"tracecontext,baggage"`
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

type otelTraceProviderKey struct{}

func ProvideOtelTraceProvider(ctx context.Context, config cfg.Config, logger log.Logger) (trace.TracerProvider, error) {
	return appctx.Provide(ctx, otelTraceProviderKey{}, func() (trace.TracerProvider, error) {
		return newOtelTraceProvider(ctx, config, logger)
	})
}

func newOtelTraceProvider(ctx context.Context, config cfg.Config, logger log.Logger) (trace.TracerProvider, error) {
	settings := &OtelSettings{}
	if err := config.UnmarshalKey("tracing.otel", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal otel tracing settings: %w", err)
	}

	res, err := otel.ProvideResource(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build otel resource: %w", err)
	}

	otelExporterFactory, ok := otelTraceExporters[settings.Exporter]
	if !ok {
		return nil, fmt.Errorf(
			"no otel exporter found for name %s, available exporters: %s",
			settings.Exporter,
			strings.Join(funk.Keys(otelTraceExporters), ", "),
		)
	}

	exporter, err := otelExporterFactory(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	propagator, err := buildPropagator(settings.Propagators)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
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

	otel.Register(otel.PriorityTraces, tracerProvider)
	otelglobal.SetTracerProvider(tracerProvider)
	otelglobal.SetTextMapPropagator(propagator)

	return otelglobal.GetTracerProvider(), nil
}

// buildPropagator assembles a composite text map propagator from the configured propagator names.
func buildPropagator(names []string) (propagation.TextMapPropagator, error) {
	if len(names) == 0 {
		names = []string{PropagatorTraceContext, PropagatorBaggage}
	}

	propagators := make([]propagation.TextMapPropagator, 0, len(names))
	for _, name := range names {
		switch name {
		case PropagatorTraceContext:
			propagators = append(propagators, propagation.TraceContext{})
		case PropagatorBaggage:
			propagators = append(propagators, propagation.Baggage{})
		default:
			return nil, fmt.Errorf("unknown trace propagator %q (supported: tracecontext, baggage)", name)
		}
	}

	return propagation.NewCompositeTextMapPropagator(propagators...), nil
}
