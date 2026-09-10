//go:build integration

package otel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/otel"
	"github.com/justtrackio/gosoline/pkg/test/env/otelcol"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/justtrackio/gosoline/pkg/tracing"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestOtelTestSuite(t *testing.T) {
	suite.Run(t, new(OtelTestSuite))
}

type OtelTestSuite struct {
	suite.Suite
	client *otelcol.Client
}

func (s *OtelTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("./config.dist.yml"),
		suite.WithLogLevel("debug"),
		suite.WithClockProvider(clock.NewRealClock()),
		suite.WithSharedEnvironment(),
	}
}

func (s *OtelTestSuite) SetupTest() error {
	s.client = s.Env().Otel("default").Client()

	return nil
}

func (s *OtelTestSuite) TestMetricExport() {
	ctx := s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	otelSettings, err := otel.ReadSettings(config)
	s.NoError(err)

	res, err := otel.BuildResource(config, otelSettings.Resource)
	s.NoError(err)

	exporter, err := otel.BuildMetricExporter(ctx, otelSettings.Exporter)
	s.NoError(err)

	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(time.Second))
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	meter := provider.Meter("github.com/justtrackio/gosoline/pkg/metric")

	// Use the gosoline OTel writer to write metrics through the full pipeline
	writer := metric.NewOtelWriterWithInterfaces(logger, meter)
	writer.Write(ctx, metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: "TestCounter",
			Unit:       metric.UnitCount,
			Value:      42,
			Kind:       metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: "RequestDuration",
			Unit:       metric.UnitMilliseconds,
			Value:      150.5,
			Kind:       metric.KindHistogram.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: "ActiveConnections",
			Unit:       metric.UnitCount,
			Value:      7,
			Kind:       metric.KindGauge.Build(),
			Dimensions: map[string]string{"service": "api"},
		},
	})

	// Flush metrics to the collector
	s.NoError(provider.ForceFlush(ctx))

	metrics, err := waitForCollector(ctx, s.client, s.client.Metrics, func(metrics []otelcol.Metric) bool {
		return findMetric(metrics, "test_counter") != nil &&
			findMetric(metrics, "request_duration") != nil &&
			findMetric(metrics, "active_connections") != nil
	})
	s.Require().NoError(err)

	counterMetric := findMetric(metrics, "test_counter")
	s.NotNil(counterMetric, "test_counter not found in metrics")
	s.Equal("Sum", counterMetric.DataType)
	s.Equal("true", counterMetric.IsMonotonic)

	histMetric := findMetric(metrics, "request_duration")
	s.NotNil(histMetric, "request_duration not found in metrics")
	s.Equal("Histogram", histMetric.DataType)
	// a duration recorded in milliseconds is exported in seconds, the base unit OTEL reports time in
	s.Equal("s", histMetric.Unit)

	gaugeMetric := findMetric(metrics, "active_connections")
	s.NotNil(gaugeMetric, "active_connections not found in metrics")
	s.Equal("Gauge", gaugeMetric.DataType)

	s.NoError(provider.Shutdown(ctx))
}

func (s *OtelTestSuite) TestLogExport() {
	ctx := s.Env().Context()
	config := s.Env().Config()

	otelSettings, err := otel.ReadSettings(config)
	s.NoError(err)

	res, err := otel.BuildResource(config, otelSettings.Resource)
	s.NoError(err)

	exporter, err := otel.BuildLogExporter(ctx, otelSettings.Exporter)
	s.NoError(err)

	processor := sdklog.NewBatchProcessor(exporter)
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(processor),
	)

	// Create the gosoline OTel log handler and emit logs
	handler := log.NewHandlerOtel(config, log.PriorityInfo, "otel", provider)

	err = handler.Log(ctx, time.Now(), log.PriorityInfo, "user %s logged in", []any{"alice"}, nil, log.Data{
		Channel:       "auth",
		ContextFields: map[string]any{"request_id": "req-123"},
		Fields:        map[string]any{"user_id": "42"},
	})
	s.NoError(err)

	err = handler.Log(ctx, time.Now(), log.PriorityError, "database connection failed", nil, nil, log.Data{
		Channel: "db",
	})
	s.NoError(err)

	// Flush logs to the collector
	s.NoError(provider.ForceFlush(ctx))

	records, err := waitForCollector(ctx, s.client, s.client.LogRecords, func(records []otelcol.LogRecord) bool {
		return findLogRecord(records, "user alice logged in") != nil &&
			findLogRecord(records, "database connection failed") != nil
	})
	s.Require().NoError(err)

	infoLog := findLogRecord(records, "user alice logged in")
	s.NotNil(infoLog, "info log not found")
	s.Equal("info", infoLog.SeverityText)
	s.Equal("auth", infoLog.Attributes["channel"])
	s.Equal("req-123", infoLog.Attributes["request_id"])
	s.Equal("42", infoLog.Attributes["user_id"])

	errorLog := findLogRecord(records, "database connection failed")
	s.NotNil(errorLog, "error log not found")
	s.Equal("error", errorLog.SeverityText)
	s.Equal("db", errorLog.Attributes["channel"])

	s.NoError(provider.Shutdown(ctx))
}

func (s *OtelTestSuite) TestTraceExport() {
	ctx := s.Env().Context()
	config := s.Env().Config()
	logger := s.Env().Logger()

	otelSettings, err := otel.ReadSettings(config)
	s.NoError(err)

	res, err := otel.BuildResource(config, otelSettings.Resource)
	s.NoError(err)

	exporterSettings := otelSettings.Exporter
	exporterSettings.Protocol = otel.ProtocolGrpc
	spanExporter, err := otel.BuildTraceExporter(ctx, exporterSettings)
	s.NoError(err)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(spanExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Use the gosoline OTel tracer to create spans
	tracer := tracing.NewOtelTracerWithInterfaces(logger, tp.Tracer("test"))

	spanCtx, parentSpan := tracer.StartSpan("otel-integration-parent")
	_, childSpan := tracer.StartSubSpan(spanCtx, "otel-integration-child")
	childSpan.Finish()
	parentSpan.Finish()

	// Flush spans to the collector
	s.NoError(tp.ForceFlush(ctx))

	spans, err := waitForCollector(ctx, s.client, s.client.Spans, func(spans []otelcol.Span) bool {
		return findSpan(spans, "otel-integration-parent") != nil &&
			findSpan(spans, "otel-integration-child") != nil
	})
	s.Require().NoError(err)

	parentOtel := findSpan(spans, "otel-integration-parent")
	childOtel := findSpan(spans, "otel-integration-child")
	s.NotNil(parentOtel, "parent span not found")
	s.NotNil(childOtel, "child span not found")
	s.Equal(parentOtel.TraceID, childOtel.TraceID, "parent and child should share the same trace ID")
	s.Equal(parentOtel.SpanID, childOtel.ParentID, "child's parent ID should match parent's span ID")
	s.NotEmpty(parentOtel.TraceID)
	s.NotEmpty(parentOtel.SpanID)

	s.NoError(tp.Shutdown(ctx))
}

func waitForCollector[T any](ctx context.Context, client *otelcol.Client, read func() ([]T, error), ready func([]T) bool) ([]T, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		telemetry, err := read()
		if err != nil {
			return nil, fmt.Errorf("poll collector telemetry: %w", err)
		}
		if ready(telemetry) {
			return telemetry, nil
		}

		select {
		case <-ctx.Done():
			output, err := client.Logs()
			if err != nil {
				return nil, fmt.Errorf("poll collector telemetry: %w; failed to read final collector output: %v", ctx.Err(), err)
			}

			return nil, fmt.Errorf("poll collector telemetry: %w\nfinal collector output:\n%s", ctx.Err(), output)
		case <-ticker.C:
		}
	}
}

func findMetric(metrics []otelcol.Metric, name string) *otelcol.Metric {
	for i := range metrics {
		if metrics[i].Name == name {
			return &metrics[i]
		}
	}

	return nil
}

func findLogRecord(records []otelcol.LogRecord, body string) *otelcol.LogRecord {
	for i := range records {
		if records[i].Body == body {
			return &records[i]
		}
	}

	return nil
}

func findSpan(spans []otelcol.Span, name string) *otelcol.Span {
	for i := range spans {
		if spans[i].Name == name {
			return &spans[i]
		}
	}

	return nil
}
