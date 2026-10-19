package httpserver

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	// MetricHttpRequestDuration records an HTTP server request's latency with response status.
	// Separate counters report request totals without a response-status dimension.
	MetricHttpRequestDuration  = "request.duration"
	MetricHttpRequestsRejected = "rejected.requests"

	metricRequests      = "requests"
	metricRouteRequests = "routed.requests"

	dimensionServerName = "name"
	dimensionRoute      = "route"
	dimensionMethod     = "request.method"
	dimensionStatusCode = "response.status_code"
)

func NewMetricMiddleware(name string, metricRecorder ServerMetricRecorder) (middleware gin.HandlerFunc, setupHandler func(definitions []Definition)) {
	// writer without any defaults until we initialize some defaults and overwrite it
	writer := metric.NewWriter(metricNamespace)

	middleware = func(ginCtx *gin.Context) {
		metricMiddleware(name, ginCtx, writer, metricRecorder)
	}

	setupHandler = func(definitions []Definition) {
		defaults := getMetricMiddlewareDefaults(name, definitions...)
		writer = metric.NewWriter(metricNamespace, defaults...)
	}

	return middleware, setupHandler
}

func metricMiddleware(name string, ginCtx *gin.Context, writer metric.Writer, metricRecorder ServerMetricRecorder) {
	start := time.Now()
	method := ginCtx.Request.Method

	metricRecorder.TrackRequestStarted(ginCtx.Request.Context())
	defer metricRecorder.TrackRequestCompleted(ginCtx.Request.Context())

	path := ginCtx.FullPath()
	if path == "" {
		// The path was not found, so no request metrics are recorded.
		return
	}

	path = trimRightPath(path)
	path = removeDuplicates(path)

	ginCtx.Next()

	requestTimeNano := time.Since(start)
	requestTimeMillisecond := float64(requestTimeNano) / float64(time.Millisecond)
	routeDimensions := metric.Dimensions{
		dimensionServerName: name,
		dimensionRoute:      path,
		dimensionMethod:     method,
	}

	data := make(metric.Data, 0, 4)
	data = append(data,
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestDuration,
			Dimensions: metric.Dimensions{
				dimensionServerName: name,
				dimensionRoute:      path,
				dimensionMethod:     method,
				dimensionStatusCode: strconv.Itoa(ginCtx.Writer.Status()),
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: requestTimeMillisecond,
			Kind:  metric.KindHistogram.Build(),
		},
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricRequests,
			Dimensions: metric.Dimensions{
				dimensionServerName: name,
			},
			Unit:  metric.UnitCount,
			Value: 1.0,
			Kind:  metric.KindCounter.Build(),
		},
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricRouteRequests,
			Dimensions: routeDimensions,
			Unit:       metric.UnitCount,
			Value:      1.0,
			Kind:       metric.KindCounter.Build(),
		},
	)

	if wasRequestRejected(ginCtx.Request) {
		data = append(data, &metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestsRejected,
			Dimensions: routeDimensions,
			Unit:       metric.UnitCount,
			Value:      1.0,
			Kind:       metric.KindCounter.Build(),
		})
	}

	writer.Write(ginCtx.Request.Context(), data)
}

// getMetricMiddlewareDefaults reports zero for request and rejected-request counters.
func getMetricMiddlewareDefaults(name string, definitions ...Definition) metric.Data {
	defaults := make(metric.Data, 0, 1+2*len(definitions))
	defaults = append(defaults, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricRequests,
		Dimensions: metric.Dimensions{
			dimensionServerName: name,
		},
		Unit:  metric.UnitCount,
		Value: 0.0,
		Kind:  metric.KindCounter.Build(),
	})

	for _, definition := range definitions {
		routeDimensions := metric.Dimensions{
			dimensionServerName: name,
			dimensionRoute:      definition.getAbsolutePath(),
			dimensionMethod:     definition.httpMethod,
		}
		defaults = append(defaults,
			&metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: metricRouteRequests,
				Dimensions: routeDimensions,
				Unit:       metric.UnitCount,
				Value:      0.0,
				Kind:       metric.KindCounter.Build(),
			},
			&metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: MetricHttpRequestsRejected,
				Dimensions: routeDimensions,
				Unit:       metric.UnitCount,
				Value:      0.0,
				Kind:       metric.KindCounter.Build(),
			},
		)
	}

	return defaults
}
