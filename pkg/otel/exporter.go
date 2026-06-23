package otel

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // registers the gzip compressor for OTLP gRPC exporters
)

// parseTLSMinVersion converts a version string like "1.2" or "1.3" to the corresponding tls constant.
func parseTLSMinVersion(version string) (uint16, error) {
	switch version {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported otel tls min_version %q (supported: 1.0, 1.1, 1.2, 1.3)", version)
	}
}

// buildTLSConfig builds a *tls.Config from the settings, loading a CA pool and, when both a
// client certificate and key are configured, a client certificate for mutual TLS (mTLS).
func buildTLSConfig(settings TLSSettings) (*tls.Config, error) {
	minVersion, err := parseTLSMinVersion(settings.MinVersion)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:         minVersion,
		ServerName:         settings.ServerName,
		InsecureSkipVerify: settings.InsecureSkipVerify, //nolint:gosec // opt-in via explicit config
	}

	if settings.CaFile != "" {
		caPem, err := os.ReadFile(settings.CaFile)
		if err != nil {
			return nil, fmt.Errorf("could not read otel tls ca file %q: %w", settings.CaFile, err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPem) {
			return nil, fmt.Errorf("could not append otel tls ca file %q to cert pool", settings.CaFile)
		}

		tlsConfig.RootCAs = pool
	}

	if settings.CertFile != "" && settings.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(settings.CertFile, settings.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("could not load otel mTLS client key pair: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// BuildTraceExporter creates an OTLP span exporter for the configured protocol.
func BuildTraceExporter(ctx context.Context, settings ExporterSettings) (sdktrace.SpanExporter, error) {
	switch settings.Protocol {
	case ProtocolGrpc:
		return buildTraceGrpc(ctx, settings)
	case ProtocolHttp:
		return buildTraceHttp(ctx, settings)
	default:
		return nil, unsupportedProtocolError(settings.Protocol)
	}
}

func buildTraceGrpc(ctx context.Context, settings ExporterSettings) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(settings.Address()),
		otlptracegrpc.WithTimeout(settings.Timeout),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig(settings.retryConfig())),
	}
	if len(settings.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(settings.Headers))
	}
	if settings.gzipEnabled() {
		opts = append(opts, otlptracegrpc.WithCompressor("gzip"))
	}
	if settings.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else {
		tlsConfig, err := buildTLSConfig(settings.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlptracegrpc.New(ctx, opts...)
}

func buildTraceHttp(ctx context.Context, settings ExporterSettings) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(settings.Address()),
		otlptracehttp.WithTimeout(settings.Timeout),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig(settings.retryConfig())),
	}
	if urlPath := settings.TracesPath(); urlPath != "" {
		opts = append(opts, otlptracehttp.WithURLPath(urlPath))
	}
	if len(settings.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(settings.Headers))
	}
	if settings.gzipEnabled() {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	}
	if settings.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	} else {
		tlsConfig, err := buildTLSConfig(settings.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfig))
	}

	return otlptracehttp.New(ctx, opts...)
}

// BuildMetricExporter creates an OTLP metric exporter for the configured protocol.
func BuildMetricExporter(ctx context.Context, settings ExporterSettings) (sdkmetric.Exporter, error) {
	switch settings.Protocol {
	case ProtocolGrpc:
		return buildMetricGrpc(ctx, settings)
	case ProtocolHttp:
		return buildMetricHttp(ctx, settings)
	default:
		return nil, unsupportedProtocolError(settings.Protocol)
	}
}

func buildMetricGrpc(ctx context.Context, settings ExporterSettings) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(settings.Address()),
		otlpmetricgrpc.WithTimeout(settings.Timeout),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig(settings.retryConfig())),
	}
	if len(settings.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(settings.Headers))
	}
	if settings.gzipEnabled() {
		opts = append(opts, otlpmetricgrpc.WithCompressor("gzip"))
	}
	if settings.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	} else {
		tlsConfig, err := buildTLSConfig(settings.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

func buildMetricHttp(ctx context.Context, settings ExporterSettings) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(settings.Address()),
		otlpmetrichttp.WithTimeout(settings.Timeout),
		otlpmetrichttp.WithRetry(otlpmetrichttp.RetryConfig(settings.retryConfig())),
	}
	if urlPath := settings.MetricsPath(); urlPath != "" {
		opts = append(opts, otlpmetrichttp.WithURLPath(urlPath))
	}
	if len(settings.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(settings.Headers))
	}
	if settings.gzipEnabled() {
		opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
	}
	if settings.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	} else {
		tlsConfig, err := buildTLSConfig(settings.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithTLSClientConfig(tlsConfig))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

// BuildLogExporter creates an OTLP log exporter for the configured protocol.
func BuildLogExporter(ctx context.Context, settings ExporterSettings) (sdklog.Exporter, error) {
	switch settings.Protocol {
	case ProtocolGrpc:
		return buildLogGrpc(ctx, settings)
	case ProtocolHttp:
		return buildLogHttp(ctx, settings)
	default:
		return nil, unsupportedProtocolError(settings.Protocol)
	}
}

func buildLogGrpc(ctx context.Context, settings ExporterSettings) (sdklog.Exporter, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(settings.Address()),
		otlploggrpc.WithTimeout(settings.Timeout),
		otlploggrpc.WithRetry(otlploggrpc.RetryConfig(settings.retryConfig())),
	}
	if len(settings.Headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(settings.Headers))
	}
	if settings.gzipEnabled() {
		opts = append(opts, otlploggrpc.WithCompressor("gzip"))
	}
	if settings.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	} else {
		tlsConfig, err := buildTLSConfig(settings.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlploggrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlploggrpc.New(ctx, opts...)
}

func buildLogHttp(ctx context.Context, settings ExporterSettings) (sdklog.Exporter, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(settings.Address()),
		otlploghttp.WithTimeout(settings.Timeout),
		otlploghttp.WithRetry(otlploghttp.RetryConfig(settings.retryConfig())),
	}
	if urlPath := settings.LogsPath(); urlPath != "" {
		opts = append(opts, otlploghttp.WithURLPath(urlPath))
	}
	if len(settings.Headers) > 0 {
		opts = append(opts, otlploghttp.WithHeaders(settings.Headers))
	}
	if settings.gzipEnabled() {
		opts = append(opts, otlploghttp.WithCompression(otlploghttp.GzipCompression))
	}
	if settings.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	} else {
		tlsConfig, err := buildTLSConfig(settings.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlploghttp.WithTLSClientConfig(tlsConfig))
	}

	return otlploghttp.New(ctx, opts...)
}

func unsupportedProtocolError(protocol string) error {
	return fmt.Errorf("unsupported otel exporter protocol %q (supported: grpc, http)", protocol)
}
