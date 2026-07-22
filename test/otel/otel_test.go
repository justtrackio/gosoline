//go:build integration

package otel_test

import (
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

	// Allow collector to process and output
	time.Sleep(500 * time.Millisecond)

	// Verify metrics arrived at the collector with correct OTel naming
	found, err := s.client.ContainsMetric("test_counter")
	s.NoError(err)
	s.True(found, "expected metric 'test_counter' in collector output")

	found, err = s.client.ContainsMetric("request_duration")
	s.NoError(err)
	s.True(found, "expected metric 'request_duration' in collector output")

	found, err = s.client.ContainsMetric("active_connections")
	s.NoError(err)
	s.True(found, "expected metric 'active_connections' in collector output")

	// Verify metric types
	metrics, err := s.client.Metrics()
	s.NoError(err)

	counterMetric := findMetric(metrics, "test_counter")
	s.NotNil(counterMetric, "test_counter not found in metrics")
	s.Equal("Sum", counterMetric.DataType)
	s.Equal("true", counterMetric.IsMonotonic)

	histMetric := findMetric(metrics, "request_duration")
	s.NotNil(histMetric, "request_duration not found in metrics")
	s.Equal("Histogram", histMetric.DataType)
	s.Equal("ms", histMetric.Unit)

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

	// Allow collector to process and output
	time.Sleep(500 * time.Millisecond)

	// Verify log records arrived at the collector
	found, err := s.client.ContainsLogRecord("user alice logged in")
	s.NoError(err)
	s.True(found, "expected log 'user alice logged in' in collector output")

	found, err = s.client.ContainsLogRecord("database connection failed")
	s.NoError(err)
	s.True(found, "expected log 'database connection failed' in collector output")

	// Verify log record details
	records, err := s.client.LogRecords()
	s.NoError(err)

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

	// Allow collector to process and output
	time.Sleep(500 * time.Millisecond)

	// Verify spans arrived at the collector
	found, err := s.client.ContainsSpan("otel-integration-parent")
	s.NoError(err)
	s.True(found, "expected span 'otel-integration-parent' in collector output")

	found, err = s.client.ContainsSpan("otel-integration-child")
	s.NoError(err)
	s.True(found, "expected span 'otel-integration-child' in collector output")

	// Verify parent-child relationship via shared trace ID
	spans, err := s.client.Spans()
	s.NoError(err)

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

func TestOtel(t *testing.T) {
	suite.Run(t, new(OtelTestSuite))
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
