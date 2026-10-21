package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestMetricMiddlewareExportsRequestAndResponseCountersForServerGates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(t.Context()))
	})

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	otelWriter := metric.NewOtelWriterWithInterfaces(logger, provider.Meter("httpserver-test"))
	var observedCounters []metric.Datum
	var observedCountersMutex sync.Mutex
	writer := namespaceMetricWriter{
		Writer: otelWriter,
		observe: func(batch metric.Data) {
			observedCountersMutex.Lock()
			defer observedCountersMutex.Unlock()
			for _, datum := range batch {
				switch datum.MetricName {
				case "requests", "routed.requests", "responses", "routed.responses", MetricHttpRequestsRejected:
					observedCounters = append(observedCounters, *datum)
				}
			}
		},
	}
	definitions := []Definition{
		{group: &Definitions{}, httpMethod: http.MethodGet, relativePath: "/widgets/:id"},
		{group: &Definitions{}, httpMethod: http.MethodGet, relativePath: "/auth"},
		{group: &Definitions{}, httpMethod: http.MethodGet, relativePath: "/chaos"},
		{group: &Definitions{}, httpMethod: http.MethodGet, relativePath: "/idle"},
		{group: &Definitions{}, httpMethod: http.MethodGet, relativePath: "/server-error"},
	}
	writer.Write(t.Context(), getMetricMiddlewareDefaults("api", definitions...))

	router := gin.New()
	// Keep metrics outermost, as in NewWithSettings: request gates short-circuit inside it.
	router.Use(func(c *gin.Context) {
		metricMiddleware("api", c, writer, metricRecorderStub{})
	})
	// NewWithSettings registers health before the overload middleware, outside its gate.
	router.GET("/health", func(c *gin.Context) {})
	router.Use(ConcurrentRequestLimitMiddleware(ConcurrencySettings{
		MaxRequests:        1,
		OverloadStatusCode: http.StatusServiceUnavailable,
	}))

	entered := make(chan struct{})
	release := make(chan struct{})
	var (
		releaseOnce sync.Once
		enteredOnce sync.Once
	)
	defer releaseOnce.Do(func() {
		close(release)
	})

	router.GET("/widgets/:id", func(c *gin.Context) {
		enteredOnce.Do(func() {
			close(entered)
		})
		<-release
		c.Status(http.StatusNoContent)
	})
	router.GET("/auth", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusUnauthorized)
	}, func(c *gin.Context) {})
	router.GET("/chaos", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusServiceUnavailable)
	})
	router.GET("/idle", func(c *gin.Context) {})
	router.GET("/server-error", func(c *gin.Context) {
		c.AbortWithStatus(599)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	firstResponse := make(chan struct {
		response *http.Response
		err      error
	}, 1)
	go func() {
		response, err := client.Get(server.URL + "/widgets/42")
		firstResponse <- struct {
			response *http.Response
			err      error
		}{response: response, err: err}
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first request did not enter its handler")
	}

	rejectedResponse, err := client.Get(server.URL + "/widgets/42")
	require.NoError(t, err)
	require.Equal(t, http.StatusServiceUnavailable, rejectedResponse.StatusCode)
	require.NoError(t, rejectedResponse.Body.Close())

	releaseOnce.Do(func() {
		close(release)
	})

	first := <-firstResponse
	require.NoError(t, first.err)
	require.Equal(t, http.StatusNoContent, first.response.StatusCode)
	require.NoError(t, first.response.Body.Close())

	authResponse, err := client.Get(server.URL + "/auth")
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, authResponse.StatusCode)
	require.NoError(t, authResponse.Body.Close())

	chaosResponse, err := client.Get(server.URL + "/chaos")
	require.NoError(t, err)
	require.Equal(t, http.StatusServiceUnavailable, chaosResponse.StatusCode)
	require.NoError(t, chaosResponse.Body.Close())

	healthResponse, err := client.Get(server.URL + "/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, healthResponse.StatusCode)
	require.NoError(t, healthResponse.Body.Close())
	serverErrorResponse, err := client.Get(server.URL + "/server-error")
	require.NoError(t, err)
	require.Equal(t, 599, serverErrorResponse.StatusCode)
	require.NoError(t, serverErrorResponse.Body.Close())

	unmatchedResponse, err := client.Get(server.URL + "/missing")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, unmatchedResponse.StatusCode)
	require.NoError(t, unmatchedResponse.Body.Close())

	var exported metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &exported))

	observedCountersMutex.Lock()
	counterSamples := append([]metric.Datum(nil), observedCounters...)
	observedCountersMutex.Unlock()
	for _, datum := range counterSamples {
		require.Equal(t, metric.PriorityHigh, datum.Priority)
		require.Equal(t, metric.UnitCount, datum.Unit)
		require.Equal(t, metric.KindCounter.Build(), datum.Kind)
	}

	metrics := make(map[string]metricdata.Metrics)
	for _, scope := range exported.ScopeMetrics {
		for _, emitted := range scope.Metrics {
			metrics[emitted.Name] = emitted
		}
	}
	require.Len(t, metrics, 6)

	assertExportedCounter(t, metrics["gosoline.http.server.requests"], map[string]float64{"": 6}, false)
	assertExportedCounter(t, metrics["gosoline.http.server.routed.requests"], map[string]float64{
		"/widgets/:id":  2,
		"/auth":         1,
		"/chaos":        1,
		"/idle":         0,
		"/health":       1,
		"/server-error": 1,
	}, true)
	assertExportedCounter(t, metrics["gosoline.http.server.rejected.requests"], map[string]float64{
		"/widgets/:id":  1,
		"/auth":         0,
		"/chaos":        0,
		"/idle":         0,
		"/server-error": 0,
	}, true)
	assertExportedResponseCounter(t, metrics["gosoline.http.server.responses"], map[string]float64{
		"204:2xx": 1,
		"503:5xx": 2,
		"401:4xx": 1,
		"200:2xx": 1,
		"599:5xx": 1,
	}, false)
	assertExportedResponseCounter(t, metrics["gosoline.http.server.routed.responses"], map[string]float64{
		"/widgets/:id:GET:204:2xx":  1,
		"/widgets/:id:GET:503:5xx":  1,
		"/auth:GET:401:4xx":         1,
		"/chaos:GET:503:5xx":        1,
		"/health:GET:200:2xx":       1,
		"/server-error:GET:599:5xx": 1,
	}, true)

	duration, ok := metrics["gosoline.http.server.request.duration"]
	require.True(t, ok)
	require.Equal(t, "s", duration.Unit)
	durationHistogram, ok := duration.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, durationHistogram.DataPoints, 6)
	statuses := make(map[string]bool, 6)
	for _, point := range durationHistogram.DataPoints {
		dimensions := metricDimensions(point.Attributes.ToSlice())
		route := dimensions[dimensionRoute]
		status := dimensions[dimensionStatusCode]
		require.Equal(t, uint64(1), point.Count)
		require.Greater(t, point.Sum, 0.0)
		require.Equal(t, http.MethodGet, dimensions[dimensionMethod])
		require.Equal(t, "api", dimensions[dimensionServerName])
		switch route {
		case "/widgets/:id":
			require.Contains(t, []string{"204", "503"}, status)
		case "/auth":
			require.Equal(t, "401", status)
		case "/health":
			require.Equal(t, "200", status)
		case "/server-error":
			require.Equal(t, "599", status)
		default:
			require.Equal(t, "/chaos", route)
			require.Equal(t, "503", status)
		}
		require.Len(t, dimensions, 4)
		statuses[route+":"+status] = true
	}
	require.Equal(t, map[string]bool{
		"/widgets/:id:204":  true,
		"/widgets/:id:503":  true,
		"/auth:401":         true,
		"/chaos:503":        true,
		"/health:200":       true,
		"/server-error:599": true,
	}, statuses)
}

func assertExportedCounter(t *testing.T, exported metricdata.Metrics, expectedValues map[string]float64, perRoute bool) {
	t.Helper()

	require.Equal(t, "{request}", exported.Unit)
	counter, ok := exported.Data.(metricdata.Sum[float64])
	require.True(t, ok)
	require.True(t, counter.IsMonotonic)
	require.Len(t, counter.DataPoints, len(expectedValues))

	actualValues := make(map[string]float64, len(counter.DataPoints))
	for _, point := range counter.DataPoints {
		dimensions := metricDimensions(point.Attributes.ToSlice())
		require.Equal(t, "api", dimensions[dimensionServerName])
		if perRoute {
			route := dimensions[dimensionRoute]
			require.NotEmpty(t, route)
			require.Equal(t, http.MethodGet, dimensions[dimensionMethod])
			require.Len(t, dimensions, 3)
			actualValues[route] = point.Value
		} else {
			require.Equal(t, metric.Dimensions{dimensionServerName: "api"}, dimensions)
			actualValues[""] = point.Value
		}
	}
	require.Equal(t, expectedValues, actualValues)
}

func assertExportedResponseCounter(t *testing.T, exported metricdata.Metrics, expectedValues map[string]float64, routed bool) {
	t.Helper()

	require.Equal(t, "{response}", exported.Unit)
	counter, ok := exported.Data.(metricdata.Sum[float64])
	require.True(t, ok)
	require.True(t, counter.IsMonotonic)
	require.Len(t, counter.DataPoints, len(expectedValues))

	actualValues := make(map[string]float64, len(counter.DataPoints))
	for _, point := range counter.DataPoints {
		dimensions := metricDimensions(point.Attributes.ToSlice())
		require.Equal(t, "api", dimensions[dimensionServerName])

		var key string
		if routed {
			require.Equal(t, http.MethodGet, dimensions[dimensionMethod])
			statusCode, statusClass := assertResponseStatusDimensions(t, dimensions, 3)
			key = dimensions[dimensionRoute] + ":" + dimensions[dimensionMethod] + ":" + statusCode + ":" + statusClass
		} else {
			statusCode, statusClass := assertResponseStatusDimensions(t, dimensions, 1)
			key = statusCode + ":" + statusClass
		}
		actualValues[key] = point.Value
	}
	require.Equal(t, expectedValues, actualValues)
}

func assertResponseStatusDimensions(t *testing.T, dimensions metric.Dimensions, baseDimensionCount int) (statusCode, statusClass string) {
	t.Helper()

	statusCode, hasStatusCode := dimensions[dimensionStatusCode]
	statusClass, hasStatusClass := dimensions[dimensionStatusClass]
	require.Equal(t, hasStatusCode, hasStatusClass)

	expectedDimensionCount := baseDimensionCount
	if hasStatusCode {
		expectedDimensionCount += 2
	}
	require.Len(t, dimensions, expectedDimensionCount)

	return statusCode, statusClass
}

func metricDimensions(attributes []attribute.KeyValue) metric.Dimensions {
	dimensions := make(metric.Dimensions, len(attributes))
	for _, keyValue := range attributes {
		dimensions[string(keyValue.Key)] = keyValue.Value.AsString()
	}

	return dimensions
}

type namespaceMetricWriter struct {
	metric.Writer
	observe func(metric.Data)
}

func (w namespaceMetricWriter) Write(ctx context.Context, batch metric.Data) {
	for _, datum := range batch {
		datum.Namespace = metricNamespace
	}
	if w.observe != nil {
		w.observe(batch)
	}

	w.Writer.Write(ctx, batch)
}

type metricRecorderStub struct{}

func (metricRecorderStub) TrackRequestStarted(context.Context)   {}
func (metricRecorderStub) TrackRequestCompleted(context.Context) {}
func (metricRecorderStub) TrackConnectionOpened(context.Context) {}
func (metricRecorderStub) TrackConnectionClosed(context.Context) {}
func (metricRecorderStub) Run(context.Context) error             { return nil }
