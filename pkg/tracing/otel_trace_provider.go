package tracing

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type OtelSettings struct {
	Exporter      string  `cfg:"exporter" default:"otel_http"`
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

type otelTraceProviderKey struct{}

func ProvideOtelTraceProvider(ctx context.Context, config cfg.Config, logger log.Logger) (trace.TracerProvider, error) {
	return appctx.Provide(ctx, otelTraceProviderKey{}, func() (trace.TracerProvider, error) {
		return newOtelTraceProvider(ctx, config, logger)
	})
}

func newOtelTraceProvider(ctx context.Context, config cfg.Config, logger log.Logger) (trace.TracerProvider, error) {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	settings := &OtelSettings{}
	config.UnmarshalKey("tracing.otel", settings)

	otelExporterFactory, ok := otelTraceExporters[settings.Exporter]
	if !ok {
		return nil, fmt.Errorf(
			"no otel exporter found for name %s, available exporters: %s",
			settings.Exporter,
			strings.Join(slices.Collect(maps.Keys(otelTraceExporters)), ", "),
		)
	}

	exporter, err := otelExporterFactory(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(fmt.Sprintf("%s-%s-%s-%s-%s", appId.Project, appId.Environment, appId.Family, appId.Group, appId.Application)),
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

	return otel.GetTracerProvider(), nil
}
