package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/clock"
	cloudwatchMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/mock"
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

	capture := &metricMiddlewareGateCapture{}
	writer, prometheusRegistry := newMetricMiddlewareGateWriter(t, provider, capture)
	definitions := []Definition{
		{
			group:        &Definitions{},
			httpMethod:   http.MethodGet,
			relativePath: "/widgets/:id",
		},
		{
			group:        &Definitions{},
			httpMethod:   http.MethodGet,
			relativePath: "/auth",
		},
		{
			group:        &Definitions{},
			httpMethod:   http.MethodGet,
			relativePath: "/chaos",
		},
		{
			group:        &Definitions{},
			httpMethod:   http.MethodGet,
			relativePath: "/idle",
		},
		{
			group:        &Definitions{},
			httpMethod:   http.MethodGet,
			relativePath: "/server-error",
		},
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
		}{
			response: response,
			err:      err,
		}
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

	observations := capture.snapshot()
	assertMetricMiddlewareGateDatumKinds(t, observations)
	metrics := metricMiddlewareMetricsByName(t, exported)
	assertMetricMiddlewareGateCounterExports(t, metrics, observations, prometheusRegistry)
	assertMetricMiddlewareGateRequestDuration(t, metrics)
}

type metricMiddlewareGateCapture struct {
	mutex                  sync.Mutex
	counters               []metric.Datum
	responseTotals         []metric.Datum
	rejectedTotals         []metric.Datum
	cloudwatchRejectedData []types.MetricDatum
}

type metricMiddlewareGateSamples struct {
	counters               []metric.Datum
	responseTotals         []metric.Datum
	rejectedTotals         []metric.Datum
	cloudwatchRejectedData []types.MetricDatum
}

func (c *metricMiddlewareGateCapture) observeMetrics(batch metric.Data) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, datum := range batch {
		switch datum.MetricName {
		case "requests", "routed.requests", "responses":
			c.counters = append(c.counters, *datum)
		case MetricHttpRequestsRejected:
			if reflect.DeepEqual(datum.Kind, metric.KindTotal) {
				c.rejectedTotals = append(c.rejectedTotals, *datum)
			} else {
				c.counters = append(c.counters, *datum)
			}
		case "routed.responses":
			// The intentional CloudWatch-only total projection is not an OTEL counter.
			if reflect.DeepEqual(datum.Kind, metric.KindTotal) {
				c.responseTotals = append(c.responseTotals, *datum)
			} else {
				c.counters = append(c.counters, *datum)
			}
		}
	}
}

func (c *metricMiddlewareGateCapture) observeCloudWatch(input *cloudwatch.PutMetricDataInput) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, datum := range input.MetricData {
		if aws.ToString(datum.MetricName) == "HttpServerRejectedRequests" {
			c.cloudwatchRejectedData = append(c.cloudwatchRejectedData, datum)
		}
	}
}

func (c *metricMiddlewareGateCapture) snapshot() metricMiddlewareGateSamples {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return metricMiddlewareGateSamples{
		counters:               append([]metric.Datum(nil), c.counters...),
		responseTotals:         append([]metric.Datum(nil), c.responseTotals...),
		rejectedTotals:         append([]metric.Datum(nil), c.rejectedTotals...),
		cloudwatchRejectedData: append([]types.MetricDatum(nil), c.cloudwatchRejectedData...),
	}
}

func newMetricMiddlewareGateWriter(
	t *testing.T,
	provider *sdkmetric.MeterProvider,
	capture *metricMiddlewareGateCapture,
) (metric.Writer, *prometheus.Registry) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	otelWriter := metric.NewOtelWriterWithInterfaces(logger, provider.Meter("httpserver-test"))
	prometheusRegistry := prometheus.NewRegistry()
	prometheusWriter := metric.NewPrometheusWriterWithInterfaces(logger, prometheusRegistry, 1000, time.Second)
	cloudwatchClient := cloudwatchMocks.NewClient(t)
	cloudwatchClient.EXPECT().PutMetricData(matcher.Context, mock.Anything).
		Run(func(_ context.Context, input *cloudwatch.PutMetricDataInput, _ ...func(*cloudwatch.Options)) {
			require.Equal(t, "test", aws.ToString(input.Namespace))
			capture.observeCloudWatch(input)
		}).
		Return(nil, nil).
		Maybe()
	cloudwatchWriter := metric.NewCloudwatchWriterWithInterfaces(
		logger,
		clock.NewFakeClockAt(time.Unix(1549283566, 0)),
		cloudwatchClient,
		"test",
		time.Second,
	)

	return namespaceMetricWriter{
		Writer:  metricWriterFanout{cloudwatchWriter, otelWriter, prometheusWriter},
		observe: capture.observeMetrics,
	}, prometheusRegistry
}

func assertMetricMiddlewareGateDatumKinds(t *testing.T, observations metricMiddlewareGateSamples) {
	t.Helper()

	for _, datum := range observations.counters {
		require.Equal(t, metric.PriorityHigh, datum.Priority)
		require.Equal(t, metric.UnitCount, datum.Unit)
		require.Equal(t, metric.KindCounter.Build(), datum.Kind)
	}

	require.Len(t, observations.responseTotals, 6)
	expectedTotalClasses := map[string]bool{
		"/widgets/:id:2xx":  true,
		"/widgets/:id:5xx":  true,
		"/auth:4xx":         true,
		"/chaos:5xx":        true,
		"/health:2xx":       true,
		"/server-error:5xx": true,
	}
	actualTotalClasses := make(map[string]bool, len(observations.responseTotals))
	for _, datum := range observations.responseTotals {
		require.Equal(t, metricNamespace, datum.Namespace)
		require.Equal(t, "routed.responses", datum.MetricName)
		require.Equal(t, metric.PriorityHigh, datum.Priority)
		require.Equal(t, metric.UnitCount, datum.Unit)
		require.Equal(t, 1.0, datum.Value)
		require.True(t, reflect.DeepEqual(datum.Kind, metric.KindTotal))
		require.Len(t, datum.Dimensions, 5)
		require.Equal(t, "api", datum.Dimensions[dimensionServerName])
		require.Equal(t, http.MethodGet, datum.Dimensions[dimensionMethod])
		require.Equal(t, metric.DimensionDefault, datum.Dimensions[dimensionStatusCode])
		actualTotalClasses[datum.Dimensions[dimensionRoute]+":"+datum.Dimensions[dimensionStatusClass]] = true
	}
	require.Equal(t, expectedTotalClasses, actualTotalClasses)

	require.Len(t, observations.rejectedTotals, 2)
	rejectedProjectionValues := make(map[float64]bool, len(observations.rejectedTotals))
	for _, datum := range observations.rejectedTotals {
		require.Equal(t, metricNamespace, datum.Namespace)
		require.Equal(t, MetricHttpRequestsRejected, datum.MetricName)
		require.Equal(t, metric.PriorityHigh, datum.Priority)
		require.Equal(t, metric.UnitCount, datum.Unit)
		require.True(t, reflect.DeepEqual(datum.Kind, metric.KindTotal))
		require.Equal(t, metric.Dimensions{dimensionServerName: "api"}, datum.Dimensions)
		rejectedProjectionValues[datum.Value] = true
	}
	require.Equal(t, map[float64]bool{0: true, 1: true}, rejectedProjectionValues)

	for _, datum := range observations.counters {
		require.Equal(t, metric.PriorityHigh, datum.Priority)
		require.Equal(t, metric.UnitCount, datum.Unit)
		require.Equal(t, metric.KindCounter.Build(), datum.Kind)
	}
}

func metricMiddlewareMetricsByName(t *testing.T, exported metricdata.ResourceMetrics) map[string]metricdata.Metrics {
	t.Helper()

	metrics := make(map[string]metricdata.Metrics)
	for _, scope := range exported.ScopeMetrics {
		for _, emitted := range scope.Metrics {
			metrics[emitted.Name] = emitted
		}
	}

	return metrics
}

func assertMetricMiddlewareGateCounterExports(
	t *testing.T,
	metrics map[string]metricdata.Metrics,
	observations metricMiddlewareGateSamples,
	prometheusRegistry *prometheus.Registry,
) {
	t.Helper()

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
	assertCloudWatchRejectedProjection(t, observations.cloudwatchRejectedData)
	assertPrometheusRawRejectedRequests(t, prometheusRegistry)
}

func assertMetricMiddlewareGateRequestDuration(t *testing.T, metrics map[string]metricdata.Metrics) {
	t.Helper()

	duration, ok := metrics["gosoline.http.server.request.duration"]
	require.True(t, ok)
	require.Equal(t, "s", duration.Unit)
	durationHistogram, ok := duration.Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, durationHistogram.DataPoints, 6)
	statuses := make(map[string]bool, 6)
	for _, point := range durationHistogram.DataPoints {
		require.Equal(t, []float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10}, point.Bounds)
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

func TestMetricMiddlewareCloudWatchRouteClassProjectionPreservesRawExporterCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	cloudwatchClient := cloudwatchMocks.NewClient(t)
	var cloudwatchData []types.MetricDatum
	cloudwatchClient.EXPECT().PutMetricData(matcher.Context, mock.Anything).
		Run(func(_ context.Context, input *cloudwatch.PutMetricDataInput, _ ...func(*cloudwatch.Options)) {
			require.Equal(t, "test", aws.ToString(input.Namespace))
			for _, datum := range input.MetricData {
				if aws.ToString(datum.MetricName) == "HttpServerRoutedResponses" {
					cloudwatchData = append(cloudwatchData, datum)
				}
			}
		}).
		Return(nil, nil).
		Times(2)

	cloudwatchWriter := metric.NewCloudwatchWriterWithInterfaces(
		logger,
		clock.NewFakeClockAt(time.Unix(1549283566, 0)),
		cloudwatchClient,
		"test",
		time.Second,
	)

	otelReader := sdkmetric.NewManualReader()
	otelProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(otelReader))
	t.Cleanup(func() {
		require.NoError(t, otelProvider.Shutdown(t.Context()))
	})
	otelWriter := metric.NewOtelWriterWithInterfaces(logger, otelProvider.Meter("httpserver-class-rollup"))

	prometheusRegistry := prometheus.NewRegistry()
	prometheusWriter := metric.NewPrometheusWriterWithInterfaces(logger, prometheusRegistry, 1000, time.Second)
	writer := namespaceMetricWriter{
		Writer: metricWriterFanout{cloudwatchWriter, otelWriter, prometheusWriter},
	}

	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		metricMiddleware("api", ctx, writer, metricRecorderStub{})
	})
	router.GET("/server-error/:status", func(ctx *gin.Context) {
		statusCode, err := strconv.Atoi(ctx.Param("status"))
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)

			return
		}

		ctx.AbortWithStatus(statusCode)
	})

	for _, statusCode := range []int{500, 599} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/server-error/"+strconv.Itoa(statusCode), http.NoBody)
		router.ServeHTTP(recorder, request)

		require.Equal(t, statusCode, recorder.Code)
	}

	assertCloudWatchRouteClassProjection(t, cloudwatchData)
	assertOTELRawRoutedResponseCounters(t, otelReader)
	assertPrometheusRawRoutedResponseCounters(t, prometheusRegistry)
}

func assertCloudWatchRouteClassProjection(t *testing.T, cloudwatchData []types.MetricDatum) {
	t.Helper()

	rawCloudwatchValues := make(map[string]float64, 2)
	var projectedCloudwatchDimensions metric.Dimensions
	var projectedCloudwatchSum float64
	projectedCloudwatchCount := 0
	for _, datum := range cloudwatchData {
		dimensions := make(metric.Dimensions, len(datum.Dimensions))
		for _, dimension := range datum.Dimensions {
			dimensions[aws.ToString(dimension.Name)] = aws.ToString(dimension.Value)
		}

		require.Equal(t, metric.UnitCount, datum.Unit)
		value := aws.ToFloat64(datum.Value)
		require.Equal(t, 1.0, value)
		if statusCode, ok := dimensions[dimensionStatusCode]; ok {
			require.Len(t, dimensions, 5)
			require.Equal(t, "5xx", dimensions[dimensionStatusClass])
			rawCloudwatchValues[statusCode] += value

			continue
		}

		require.Len(t, dimensions, 4)
		require.Equal(t, metric.Dimensions{
			dimensionServerName:  "api",
			dimensionRoute:       "/server-error/:status",
			dimensionMethod:      http.MethodGet,
			dimensionStatusClass: "5xx",
		}, dimensions)
		if projectedCloudwatchCount == 0 {
			projectedCloudwatchDimensions = dimensions
		} else {
			require.Equal(t, projectedCloudwatchDimensions, dimensions)
		}
		projectedCloudwatchCount++
		projectedCloudwatchSum += value
	}
	require.Equal(t, map[string]float64{"500": 1, "599": 1}, rawCloudwatchValues)
	require.Equal(t, 2, projectedCloudwatchCount)
	require.Equal(t, 2.0, projectedCloudwatchSum)
}

func assertCloudWatchRejectedProjection(t *testing.T, cloudwatchData []types.MetricDatum) {
	t.Helper()

	rawValues := make(map[string][]float64, 5)
	var projectedValues []float64
	for _, datum := range cloudwatchData {
		require.Equal(t, "HttpServerRejectedRequests", aws.ToString(datum.MetricName))
		require.Equal(t, metric.UnitCount, datum.Unit)
		value := aws.ToFloat64(datum.Value)
		dimensions := make(metric.Dimensions, len(datum.Dimensions))
		for _, dimension := range datum.Dimensions {
			dimensions[aws.ToString(dimension.Name)] = aws.ToString(dimension.Value)
		}

		if route, ok := dimensions[dimensionRoute]; ok {
			require.Len(t, dimensions, 3)
			require.Equal(t, "api", dimensions[dimensionServerName])
			require.Equal(t, http.MethodGet, dimensions[dimensionMethod])
			rawValues[route] = append(rawValues[route], value)

			continue
		}

		require.Equal(t, metric.Dimensions{dimensionServerName: "api"}, dimensions)
		projectedValues = append(projectedValues, value)
	}

	require.Equal(t, map[string][]float64{
		"/widgets/:id":  {0, 1},
		"/auth":         {0},
		"/chaos":        {0},
		"/idle":         {0},
		"/server-error": {0},
	}, rawValues)
	require.ElementsMatch(t, []float64{0, 1}, projectedValues)
}

func assertPrometheusRawRejectedRequests(t *testing.T, registry *prometheus.Registry) {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)
	var rejectedRequests *dto.MetricFamily
	for _, family := range families {
		if family.GetName() == "gosoline_http_server_rejected_requests_total" {
			rejectedRequests = family

			break
		}
	}
	require.NotNil(t, rejectedRequests)
	require.Equal(t, dto.MetricType_COUNTER, rejectedRequests.GetType())
	require.Len(t, rejectedRequests.Metric, 5)

	values := make(map[string]float64, 5)
	var sum float64
	for _, sample := range rejectedRequests.Metric {
		labels := make(map[string]string, len(sample.Label))
		for _, label := range sample.Label {
			labels[label.GetName()] = label.GetValue()
		}
		require.Len(t, labels, 3)
		require.Equal(t, "api", labels["name"])
		require.Equal(t, http.MethodGet, labels["request_method"])
		value := sample.GetCounter().GetValue()
		values[labels["route"]] = value
		sum += value
	}
	require.Equal(t, map[string]float64{
		"/widgets/:id":  1,
		"/auth":         0,
		"/chaos":        0,
		"/idle":         0,
		"/server-error": 0,
	}, values)
	require.Equal(t, 1.0, sum)
}

func assertOTELRawRoutedResponseCounters(t *testing.T, reader *sdkmetric.ManualReader) {
	t.Helper()

	var exported metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &exported))
	otelValues := make(map[string]float64, 2)
	var otelSum float64
	for _, scope := range exported.ScopeMetrics {
		for _, exported := range scope.Metrics {
			if exported.Name != "gosoline.http.server.routed.responses" {
				continue
			}

			counter, ok := exported.Data.(metricdata.Sum[float64])
			require.True(t, ok)
			require.True(t, counter.IsMonotonic)
			require.Len(t, counter.DataPoints, 2)
			for _, point := range counter.DataPoints {
				dimensions := metricDimensions(point.Attributes.ToSlice())
				require.Len(t, dimensions, 5)
				require.Equal(t, "5xx", dimensions[dimensionStatusClass])
				otelValues[dimensions[dimensionStatusCode]] += point.Value
				otelSum += point.Value
			}
		}
	}
	require.Equal(t, map[string]float64{"500": 1, "599": 1}, otelValues)
	require.Equal(t, 2.0, otelSum)
}

func assertPrometheusRawRoutedResponseCounters(t *testing.T, registry *prometheus.Registry) {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)
	var routedResponses *dto.MetricFamily
	for _, family := range families {
		if family.GetName() == "gosoline_http_server_routed_responses_total" {
			routedResponses = family

			break
		}
	}
	require.NotNil(t, routedResponses)
	require.Equal(t, dto.MetricType_COUNTER, routedResponses.GetType())
	require.Len(t, routedResponses.Metric, 2)
	values := make(map[string]float64, 2)
	var sum float64
	for _, sample := range routedResponses.Metric {
		labels := make(map[string]string, len(sample.Label))
		for _, label := range sample.Label {
			labels[label.GetName()] = label.GetValue()
		}
		require.Equal(t, "5xx", labels["response_status_class"])
		value := sample.GetCounter().GetValue()
		values[labels["response_status_code"]] += value
		sum += value
	}
	require.Equal(t, map[string]float64{"500": 1, "599": 1}, values)
	require.Equal(t, 2.0, sum)
}

type metricWriterFanout []metric.Writer

func (w metricWriterFanout) GetPriority() int {
	return metric.PriorityLow
}

func (w metricWriterFanout) Write(ctx context.Context, data metric.Data) {
	for _, writer := range w {
		writer.Write(ctx, data)
	}
}

func (w metricWriterFanout) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
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
