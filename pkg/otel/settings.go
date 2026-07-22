// Package otel provides the shared OpenTelemetry core used by gosoline's
// tracing, metric, and log integrations: a resource builder derived from the
// application identity and OTLP exporter builders (gRPC/HTTP, with TLS/mTLS)
// driven by the native gosoline config system.
package otel

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

// ConfigKey is the root config key for the shared OTEL settings.
const ConfigKey = "otel"

// ProtocolGrpc and ProtocolHttp are the supported OTLP transports.
const (
	ProtocolGrpc = "grpc"
	ProtocolHttp = "http"
)

// Settings holds the shared OTEL configuration reused by all signals.
type Settings struct {
	Resource ResourceSettings `cfg:"resource"`
	Exporter ExporterSettings `cfg:"exporter"`
}

// ResourceSettings configures the OTEL resource attributes derived from the app identity.
type ResourceSettings struct {
	// ServiceNamePattern is expanded via cfg.Identity.Format (placeholders: {app.name}, {app.env}, {app.namespace}, {app.tags.x}).
	ServiceNamePattern string `cfg:"service_name_pattern,nodecode" default:"{app.name}"`
	// ServiceNamespacePattern maps to the service.namespace resource attribute.
	ServiceNamespacePattern string `cfg:"service_namespace_pattern,nodecode" default:"{app.namespace}"`
	// Delimiter is used when joining namespace parts during pattern expansion.
	Delimiter string `cfg:"delimiter" default:"-"`
	// Attributes are additional resource attributes; values may contain identity placeholders.
	Attributes map[string]string `cfg:"attributes"`
}

// ExporterSettings configures a single OTLP exporter shared by all signals (per-signal overrides possible).
type ExporterSettings struct {
	// Protocol selects the OTLP transport: grpc (default) or http.
	Protocol string `cfg:"protocol" default:"grpc"`
	// Host is the collector host; override via env from pod metadata (status.hostIP).
	Host string `cfg:"host" default:"localhost"`
	// Port is the collector port (4317 for gRPC, 4318 for HTTP by convention).
	Port int `cfg:"port" default:"4317"`
	// Endpoint, when set, takes precedence over Host:Port.
	Endpoint string `cfg:"endpoint" default:""`
	// UrlPath overrides the HTTP signal path for all signals; empty uses the SDK defaults.
	// Per-signal paths (TracesUrlPath, MetricsUrlPath, LogsUrlPath) take precedence when set.
	UrlPath string `cfg:"url_path" default:""`
	// TracesUrlPath overrides the HTTP path for traces; falls back to UrlPath, then SDK default (/v1/traces).
	TracesUrlPath string `cfg:"traces_url_path" default:""`
	// MetricsUrlPath overrides the HTTP path for metrics; falls back to UrlPath, then SDK default (/v1/metrics).
	MetricsUrlPath string `cfg:"metrics_url_path" default:""`
	// LogsUrlPath overrides the HTTP path for logs; falls back to UrlPath, then SDK default (/v1/logs).
	LogsUrlPath string `cfg:"logs_url_path" default:""`
	// Insecure disables transport security; set false to enable TLS/mTLS.
	Insecure bool `cfg:"insecure" default:"true"`
	// Compression enables payload compression ("gzip" or "" / "none").
	Compression string `cfg:"compression" default:"gzip"`
	// Timeout bounds a single export attempt.
	Timeout time.Duration `cfg:"timeout" default:"10s"`
	// Headers are static headers attached to every export (auth, tenant, ...).
	Headers map[string]string `cfg:"headers"`
	// TLS configures transport security when Insecure is false.
	TLS TLSSettings `cfg:"tls"`
	// Retry configures the exporter's built-in retry behavior.
	Retry RetrySettings `cfg:"retry"`
}

// TLSSettings configures TLS and, when client cert/key are set, mTLS.
type TLSSettings struct {
	CaFile             string `cfg:"ca_file" default:""`
	CertFile           string `cfg:"cert_file" default:""`
	KeyFile            string `cfg:"key_file" default:""`
	ServerName         string `cfg:"server_name" default:""`
	InsecureSkipVerify bool   `cfg:"insecure_skip_verify" default:"false"`
	// MinVersion is the minimum TLS version to accept (e.g. "1.2", "1.3"). Defaults to TLS 1.3.
	MinVersion string `cfg:"min_version" default:"1.3"`
}

// RetrySettings configures the OTLP exporter retry/backoff.
type RetrySettings struct {
	Enabled         bool          `cfg:"enabled" default:"true"`
	InitialInterval time.Duration `cfg:"initial_interval" default:"5s"`
	MaxInterval     time.Duration `cfg:"max_interval" default:"30s"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time" default:"300s"`
}

// Address returns the exporter endpoint, preferring an explicit Endpoint over Host:Port.
func (e ExporterSettings) Address() string {
	if e.Endpoint != "" {
		return e.Endpoint
	}

	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}

// TracesPath returns the URL path for traces, preferring TracesUrlPath over UrlPath.
func (e ExporterSettings) TracesPath() string {
	if e.TracesUrlPath != "" {
		return e.TracesUrlPath
	}

	return e.UrlPath
}

// MetricsPath returns the URL path for metrics, preferring MetricsUrlPath over UrlPath.
func (e ExporterSettings) MetricsPath() string {
	if e.MetricsUrlPath != "" {
		return e.MetricsUrlPath
	}

	return e.UrlPath
}

// LogsPath returns the URL path for logs, preferring LogsUrlPath over UrlPath.
func (e ExporterSettings) LogsPath() string {
	if e.LogsUrlPath != "" {
		return e.LogsUrlPath
	}

	return e.UrlPath
}

// gzipEnabled reports whether gzip compression is requested.
func (e ExporterSettings) gzipEnabled() bool {
	return e.Compression == "gzip"
}

// retryConfig is field-compatible with the per-signal OTLP RetryConfig types and is converted to
// each via a struct conversion in the exporter builders, avoiding repetition.
type retryConfig struct {
	Enabled         bool
	InitialInterval time.Duration
	MaxInterval     time.Duration
	MaxElapsedTime  time.Duration
}

func (e ExporterSettings) retryConfig() retryConfig {
	return retryConfig{
		Enabled:         e.Retry.Enabled,
		InitialInterval: e.Retry.InitialInterval,
		MaxInterval:     e.Retry.MaxInterval,
		MaxElapsedTime:  e.Retry.MaxElapsedTime,
	}
}

// ReadSettings unmarshals the shared OTEL settings from the config root.
func ReadSettings(config cfg.Config) (*Settings, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey(ConfigKey, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal otel settings: %w", err)
	}

	return settings, nil
}
