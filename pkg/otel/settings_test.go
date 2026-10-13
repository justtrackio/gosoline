package otel_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestExporterHeadersArePrivateInConfigDebug(t *testing.T) {
	headers := map[string]string{
		"Authorization": "synthetic-exporter-token",
		"X-Tenant":      "synthetic-tenant-header",
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"otel": map[string]any{
			"exporter": map[string]any{
				"headers": headers,
				"host":    "visible-collector",
				"tls": map[string]any{
					"key_file": "/visible/client-key.pem",
				},
			},
		},
	})
	settings, err := otel.ReadSettings(config)
	require.NoError(t, err)
	require.Equal(t, headers, settings.Exporter.Headers)

	var output bytes.Buffer
	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, &output)
	logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
	require.NoError(t, cfg.DebugConfig(t.Context(), config, logger))
	require.NotContains(t, output.String(), "synthetic-exporter-token")
	require.NotContains(t, output.String(), "synthetic-tenant-header")
	require.Contains(t, output.String(), "visible-collector")
	require.Contains(t, output.String(), "/visible/client-key.pem")

	stored, err := config.GetStringMapString("otel.exporter.headers")
	require.NoError(t, err)
	require.Equal(t, headers, stored)
}
