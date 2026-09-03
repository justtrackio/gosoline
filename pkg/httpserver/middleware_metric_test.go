package httpserver

import (
	"context"
	stdHttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricMiddlewareWritesOnlyRouteSeries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := middlewareMetricWriter{batches: make(chan metric.Data, 2)}
	router := gin.New()
	router.GET("/widgets/:id", func(ginCtx *gin.Context) {
		metricMiddleware("api", ginCtx, writer, noopServerMetricRecorder{})
	})

	request := httptest.NewRequest(stdHttp.MethodGet, "/widgets/1", nil)
	request = request.WithContext(context.WithValue(request.Context(), rejectedRequestKey{}, true))
	router.ServeHTTP(httptest.NewRecorder(), request)

	duration := <-writer.batches
	rejected := <-writer.batches

	require.Len(t, duration, 1)
	assert.Equal(t, MetricHttpRequestDuration, duration[0].MetricName)
	assert.Equal(t, metric.Dimensions{
		dimensionServerName: "api",
		dimensionRoute:      "/widgets/:id",
		dimensionMethod:     stdHttp.MethodGet,
		dimensionStatusCode: "200",
	}, duration[0].Dimensions)

	require.Len(t, rejected, 1)
	assert.Equal(t, MetricHttpRequestsRejected, rejected[0].MetricName)
	assert.Equal(t, metric.Dimensions{
		dimensionServerName: "api",
		dimensionRoute:      "/widgets/:id",
		dimensionMethod:     stdHttp.MethodGet,
	}, rejected[0].Dimensions)
}

func TestMetricMiddlewareDefaultsExcludeServerAggregate(t *testing.T) {
	assert.Empty(t, getMetricMiddlewareDefaults("api"))
}

type middlewareMetricWriter struct {
	batches chan metric.Data
}

func (w middlewareMetricWriter) GetPriority() int {
	return metric.PriorityLow
}

func (w middlewareMetricWriter) Write(_ context.Context, data metric.Data) {
	w.batches <- data
}

func (w middlewareMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
}

type noopServerMetricRecorder struct{}

func (noopServerMetricRecorder) TrackRequestStarted(context.Context)   {}
func (noopServerMetricRecorder) TrackRequestCompleted(context.Context) {}
func (noopServerMetricRecorder) TrackConnectionOpened(context.Context) {}
func (noopServerMetricRecorder) TrackConnectionClosed(context.Context) {}
func (noopServerMetricRecorder) Run(context.Context) error             { return nil }
