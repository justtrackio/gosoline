package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/clock"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestClientRequestCountersTrackStartsAndCompletedOutcomes(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(t.Context()))
	})

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	writer := clientNamespaceMetricWriter{Writer: metric.NewOtelWriterWithInterfaces(logger, provider.Meter("http-client-test"))}
	client := NewHttpClientWithInterfaces(logger, clock.Provider, writer, resty.New(), false, 0, NewNopUrlValidator())

	requestEntered := make(chan struct{})
	completeRequest := make(chan struct{})
	var completeRequestOnce sync.Once
	responseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(requestEntered)
		<-completeRequest
		w.WriteHeader(599)
	}))
	defer responseServer.Close()
	defer completeRequestOnce.Do(func() {
		close(completeRequest)
	})

	type result struct {
		response *Response
		err      error
	}
	responseResult := make(chan result, 1)
	go func() {
		response, err := client.Get(t.Context(), client.NewRequest().WithUrl(responseServer.URL))
		responseResult <- result{response: response, err: err}
	}()

	select {
	case <-requestEntered:
	case <-time.After(time.Second):
		t.Fatal("request did not reach the response handler")
	}

	inFlight := collectClientMetrics(t, reader)
	assertClientCounterPoint(t, inFlight["gosoline.http.client.requests"], metric.Dimensions{dimensionMethod: GetRequest}, 1)
	require.NotContains(t, inFlight, "gosoline.http.client.completed.requests")

	completeRequestOnce.Do(func() {
		close(completeRequest)
	})
	select {
	case result := <-responseResult:
		require.NoError(t, result.err)
		require.NotNil(t, result.response)
		require.Equal(t, 599, result.response.StatusCode)
	case <-time.After(time.Second):
		t.Fatal("request did not complete")
	}

	failedServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	failedURL := failedServer.URL
	failedServer.Close()
	_, err := client.Get(t.Context(), client.NewRequest().WithUrl(failedURL))
	require.Error(t, err)
	require.False(t, errors.Is(err, context.Canceled))

	canceledRequestEntered := make(chan struct{})
	canceledServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(canceledRequestEntered)
		<-request.Context().Done()
	}))
	defer canceledServer.Close()
	canceledContext, cancel := context.WithCancel(t.Context())
	defer cancel()
	canceledResult := make(chan error, 1)
	go func() {
		_, requestErr := client.Get(canceledContext, client.NewRequest().WithUrl(canceledServer.URL))
		canceledResult <- requestErr
	}()

	select {
	case <-canceledRequestEntered:
	case <-time.After(time.Second):
		t.Fatal("canceled request did not reach the response handler")
	}
	cancel()
	select {
	case err := <-canceledResult:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("canceled request did not return")
	}

	exported := collectClientMetrics(t, reader)
	assertClientCounterPoint(t, exported["gosoline.http.client.requests"], metric.Dimensions{dimensionMethod: GetRequest}, 3)
	completed := clientCounter(t, exported["gosoline.http.client.completed.requests"])
	require.Len(t, completed.DataPoints, 3)
	completedOutcomes := make(map[string]metric.Dimensions, len(completed.DataPoints))
	for _, point := range completed.DataPoints {
		dimensions := clientMetricDimensions(point.Attributes.ToSlice())
		require.Equal(t, 1.0, point.Value)
		key := dimensions[dimensionStatusCode] + ":" + dimensions[dimensionStatusClass] + ":" + dimensions[metric.DimensionErrorType]
		completedOutcomes[key] = dimensions
	}
	require.Equal(t, map[string]metric.Dimensions{
		"599:5xx:": {
			dimensionMethod:      GetRequest,
			dimensionStatusCode:  "599",
			dimensionStatusClass: "5xx",
		},
		"::request_failed": {
			dimensionMethod:           GetRequest,
			metric.DimensionErrorType: errorTypeRequestFailed,
		},
		"::request_canceled": {
			dimensionMethod:           GetRequest,
			metric.DimensionErrorType: errorTypeRequestCanceled,
		},
	}, completedOutcomes)

	duration, ok := exported["gosoline.http.client.request.duration"]
	require.True(t, ok)
	require.Equal(t, "s", duration.Unit)
	histogram, ok := duration.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 3)
	durationOutcomes := make(map[string]bool, len(histogram.DataPoints))
	for _, point := range histogram.DataPoints {
		dimensions := clientMetricDimensions(point.Attributes.ToSlice())
		require.Equal(t, uint64(1), point.Count)
		require.Equal(t, GetRequest, dimensions[dimensionMethod])
		durationOutcomes[dimensions[dimensionStatusCode]+":"+dimensions[metric.DimensionErrorType]] = true
		require.NotContains(t, dimensions, dimensionStatusClass)
	}
	require.Equal(t, map[string]bool{
		"599:":                         true,
		":" + errorTypeRequestFailed:   true,
		":" + errorTypeRequestCanceled: true,
	}, durationOutcomes)
}

func collectClientMetrics(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()

	var exported metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &exported))

	metrics := make(map[string]metricdata.Metrics)
	for _, scope := range exported.ScopeMetrics {
		for _, emitted := range scope.Metrics {
			metrics[emitted.Name] = emitted
		}
	}

	return metrics
}

func assertClientCounterPoint(t *testing.T, exported metricdata.Metrics, dimensions metric.Dimensions, value float64) {
	t.Helper()

	counter := clientCounter(t, exported)
	require.Equal(t, "{request}", exported.Unit)
	require.True(t, counter.IsMonotonic)
	require.Len(t, counter.DataPoints, 1)
	require.Equal(t, dimensions, clientMetricDimensions(counter.DataPoints[0].Attributes.ToSlice()))
	require.Equal(t, value, counter.DataPoints[0].Value)
}

func clientCounter(t *testing.T, exported metricdata.Metrics) metricdata.Sum[float64] {
	t.Helper()

	counter, ok := exported.Data.(metricdata.Sum[float64])
	require.True(t, ok)

	return counter
}

func clientMetricDimensions(attributes []attribute.KeyValue) metric.Dimensions {
	dimensions := make(metric.Dimensions, len(attributes))
	for _, keyValue := range attributes {
		dimensions[string(keyValue.Key)] = keyValue.Value.AsString()
	}

	return dimensions
}

type clientNamespaceMetricWriter struct {
	metric.Writer
}

func (w clientNamespaceMetricWriter) Write(ctx context.Context, batch metric.Data) {
	for _, datum := range batch {
		if datum.Namespace == "" {
			datum.Namespace = metricNamespace
		}
	}

	w.Writer.Write(ctx, batch)
}

func (w clientNamespaceMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
}
