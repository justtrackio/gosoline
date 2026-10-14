package otel_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/otel"
	"github.com/stretchr/testify/assert"
)

func TestExporterSettings_Address(t *testing.T) {
	t.Run("host and port", func(t *testing.T) {
		s := otel.ExporterSettings{Host: "localhost", Port: 4317}
		assert.Equal(t, "localhost:4317", s.Address())
	})

	t.Run("explicit endpoint wins", func(t *testing.T) {
		s := otel.ExporterSettings{Host: "localhost", Port: 4317, Endpoint: "collector.obs:4317"}
		assert.Equal(t, "collector.obs:4317", s.Address())
	})

	t.Run("ipv6 host is bracketed", func(t *testing.T) {
		s := otel.ExporterSettings{Host: "::1", Port: 4317}
		assert.Equal(t, "[::1]:4317", s.Address())
	})
}

func TestExporterSettings_SignalPaths(t *testing.T) {
	t.Run("per-signal path takes precedence", func(t *testing.T) {
		s := otel.ExporterSettings{
			UrlPath:        "/shared",
			TracesUrlPath:  "/otel/v1/traces",
			MetricsUrlPath: "/otel/v1/metrics",
			LogsUrlPath:    "/otel/v1/logs",
		}
		assert.Equal(t, "/otel/v1/traces", s.TracesPath())
		assert.Equal(t, "/otel/v1/metrics", s.MetricsPath())
		assert.Equal(t, "/otel/v1/logs", s.LogsPath())
	})

	t.Run("falls back to shared url_path", func(t *testing.T) {
		s := otel.ExporterSettings{UrlPath: "/collector"}
		assert.Equal(t, "/collector", s.TracesPath())
		assert.Equal(t, "/collector", s.MetricsPath())
		assert.Equal(t, "/collector", s.LogsPath())
	})

	t.Run("empty when nothing set", func(t *testing.T) {
		s := otel.ExporterSettings{}
		assert.Equal(t, "", s.TracesPath())
		assert.Equal(t, "", s.MetricsPath())
		assert.Equal(t, "", s.LogsPath())
	})
}
