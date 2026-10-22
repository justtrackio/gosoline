package grpcserver

import (
	"context"
	"errors"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc/stats"
)

func TestStatsHandlerCountsCompletedRPCWithError(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(t.Context()))
	})

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	writer := statsNamespaceMetricWriter{Writer: metric.NewOtelWriterWithInterfaces(logger, provider.Meter("grpcserver-test"))}
	handler := &statsHandler{
		logger:       logger,
		metricWriter: writer,
		settings:     &Settings{Stats: Stats{Channel: "grpc_stats", LogLevel: "disabled"}},
	}

	ctx := handler.TagRPC(handler.TagConn(t.Context(), nil), nil)
	handler.HandleRPC(ctx, &stats.InHeader{FullMethod: "/helloworld.v1.GreeterService/SayHello"})
	handler.HandleRPC(ctx, &stats.End{
		BeginTime: time.Unix(10, 0),
		EndTime:   time.Unix(10, 0).Add(25 * time.Millisecond),
		Error:     errors.New("handler failed"),
	})

	var exported metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &exported))
	metrics := make(map[string]metricdata.Metrics)
	for _, scope := range exported.ScopeMetrics {
		for _, emitted := range scope.Metrics {
			metrics[emitted.Name] = emitted
		}
	}

	counterMetric, ok := metrics["gosoline.rpc.server.requests"]
	require.True(t, ok)
	require.Equal(t, "{request}", counterMetric.Unit)
	counter, ok := counterMetric.Data.(metricdata.Sum[float64])
	require.True(t, ok)
	require.True(t, counter.IsMonotonic)
	require.Len(t, counter.DataPoints, 1)
	require.Equal(t, metric.Dimensions{
		MetricDimensionRpcService: "helloworld.v1.GreeterService",
		MetricDimensionRpcMethod:  "SayHello",
	}, statsMetricDimensions(counter.DataPoints[0].Attributes.ToSlice()))
	require.Equal(t, 1.0, counter.DataPoints[0].Value)

	durationMetric, ok := metrics["gosoline.rpc.server.request.duration"]
	require.True(t, ok)
	require.Equal(t, "s", durationMetric.Unit)
	duration, ok := durationMetric.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, duration.DataPoints, 1)
	require.Equal(t, uint64(1), duration.DataPoints[0].Count)
	require.Equal(t, metric.Dimensions{
		MetricDimensionRpcService: "helloworld.v1.GreeterService",
		MetricDimensionRpcMethod:  "SayHello",
	}, statsMetricDimensions(duration.DataPoints[0].Attributes.ToSlice()))
	require.InDelta(t, 0.025, duration.DataPoints[0].Sum, 1e-9)
}

func statsMetricDimensions(attributes []attribute.KeyValue) metric.Dimensions {
	dimensions := make(metric.Dimensions, len(attributes))
	for _, keyValue := range attributes {
		dimensions[string(keyValue.Key)] = keyValue.Value.AsString()
	}

	return dimensions
}

type statsNamespaceMetricWriter struct {
	metric.Writer
}

func (w statsNamespaceMetricWriter) Write(ctx context.Context, batch metric.Data) {
	for _, datum := range batch {
		if datum.Namespace == "" {
			datum.Namespace = metricNamespace
		}
	}

	w.Writer.Write(ctx, batch)
}

func (w statsNamespaceMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
}
