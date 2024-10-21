package tracing

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

type OtelExporterSettings struct {
	Endpoint    string        `cfg:"endpoint" default:"localhost:4318"`
	UrlPath     string        `cfg:"url_path" default:"/v1/traces"`
	Compression bool          `cfg:"compression" default:"true"`
	Insecure    bool          `cfg:"insecure" default:"false"`
	Timeout     time.Duration `cfg:"timeout" default:"10s"`
	Retry       RetryConfig   `cfg:"retry"`
}

type RetryConfig struct {
	Enabled         bool          `cfg:"enabled" default:"false"`
	InitialInterval time.Duration `cfg:"initial_interval"`
	MaxInterval     time.Duration `cfg:"max_interval"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time"`
}

type OtelExporterFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (*otlptrace.Exporter, error)

func AddTraceExporter(name string, exporter OtelExporterFactory) {
	TraceExporters[name] = exporter
}

var TraceExporters = map[string]OtelExporterFactory{
	"otel_http": NewOtelHttpTracer,
}

func NewOtelHttpTracer(ctx context.Context, config cfg.Config, _ log.Logger) (*otlptrace.Exporter, error) {
	settings := &OtelExporterSettings{}
	config.UnmarshalKey("tracing.otel.http", settings)

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(settings.Endpoint),
		otlptracehttp.WithURLPath(settings.UrlPath),
		otlptracehttp.WithTimeout(settings.Timeout),
	}

	if settings.Compression {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	}

	if settings.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if settings.Retry.Enabled {
		opts = append(opts, otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         settings.Retry.Enabled,
			InitialInterval: settings.Retry.InitialInterval,
			MaxInterval:     settings.Retry.MaxInterval,
			MaxElapsedTime:  settings.Retry.MaxElapsedTime,
		}))
	}

	return otlptracehttp.New(ctx, opts...)
}
