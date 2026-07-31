package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	// ExporterOtelHttp pushes spans via OTLP/HTTP (legacy config under tracing.otel.http).
	ExporterOtelHttp = "otel_http"
	// ExporterOtelGrpc pushes spans via OTLP/gRPC using the shared otel.exporter config.
	ExporterOtelGrpc = "otel_grpc"
	// ExporterStdout writes spans to stdout (local/dev).
	ExporterStdout = "stdout"
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
	InitialInterval time.Duration `cfg:"initial_interval" default:"5s"`
	MaxInterval     time.Duration `cfg:"max_interval" default:"30s"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time" default:"300s"`
}

// OtelExporterFactory builds an OTEL span exporter. It returns the SpanExporter interface so
// OTLP (http/grpc) and non-OTLP (stdout) exporters can be registered uniformly.
type OtelExporterFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (sdktrace.SpanExporter, error)

func AddOtelTraceExporter(name string, exporter OtelExporterFactory) {
	otelTraceExporters[name] = exporter
}

var otelTraceExporters = map[string]OtelExporterFactory{
	ExporterOtelHttp: NewOtelHttpTracer,
	ExporterOtelGrpc: NewOtelGrpcTracer,
	ExporterStdout:   NewStdoutTracer,
}

// NewOtelHttpTracer builds an OTLP/HTTP span exporter from the legacy tracing.otel.http config.
func NewOtelHttpTracer(ctx context.Context, config cfg.Config, _ log.Logger) (sdktrace.SpanExporter, error) {
	settings := &OtelExporterSettings{}
	if err := config.UnmarshalKey("tracing.otel.http", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal otel http exporter settings: %w", err)
	}

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

// NewOtelGrpcTracer builds an OTLP/gRPC span exporter from the shared otel.exporter config (forced to gRPC).
func NewOtelGrpcTracer(ctx context.Context, config cfg.Config, _ log.Logger) (sdktrace.SpanExporter, error) {
	settings, err := otel.ReadSettings(config)
	if err != nil {
		return nil, err
	}

	exporterSettings := settings.Exporter
	exporterSettings.Protocol = otel.ProtocolGrpc

	return otel.BuildTraceExporter(ctx, exporterSettings)
}

// NewStdoutTracer writes spans to stdout, useful for local development.
func NewStdoutTracer(_ context.Context, _ cfg.Config, _ log.Logger) (sdktrace.SpanExporter, error) {
	return stdouttrace.New(stdouttrace.WithPrettyPrint())
}
