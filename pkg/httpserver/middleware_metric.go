package httpserver

import (
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	// MetricHttpRequestDuration records how long a request took. Its observation count is the request
	// count, and its dimensions tell routes, methods and status codes apart, so none of those needs a
	// metric of its own.
	MetricHttpRequestDuration  = "request.duration"
	MetricHttpRequestsRejected = "rejected.requests"

	dimensionServerName = "http.server.name"
	dimensionRoute      = "http.route"
	dimensionMethod     = "http.request.method"
	dimensionStatusCode = "http.response.status_code"
)

func NewMetricMiddleware(name string, metricRecorder ServerMetricRecorder) (middleware gin.HandlerFunc, setupHandler func(definitions []Definition)) {
	// writer without any defaults until we initialize some defaults and overwrite it
	writer := metric.NewWriter(metric.NamespaceHttpServer)

	middleware = func(ginCtx *gin.Context) {
		metricMiddleware(name, ginCtx, writer, metricRecorder)
	}

	setupHandler = func(definitions []Definition) {
		defaults := getMetricMiddlewareDefaults(name, definitions...)
		writer = metric.NewWriter(metric.NamespaceHttpServer, defaults...)
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
		// the path was not found, so no need to print anything
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
		dimensionStatusCode: strconv.Itoa(ginCtx.Writer.Status()),
	}

	writer.Write(ginCtx.Request.Context(), metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestDuration,
			Dimensions: routeDimensions,
			Unit:       metric.UnitMillisecondsAverage,
			Value:      requestTimeMillisecond,
			Kind:       metric.KindHistogram.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestDuration,
			Dimensions: metric.Dimensions{
				dimensionServerName: name,
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: requestTimeMillisecond,
			Kind:  metric.KindTotal,
		},
	})

	if wasRequestRejected(ginCtx.Request) {
		writer.Write(ginCtx.Request.Context(), metric.Data{
			{
				Priority:   metric.PriorityHigh,
				MetricName: MetricHttpRequestsRejected,
				Dimensions: metric.Dimensions{
					dimensionServerName: name,
					dimensionRoute:      path,
					dimensionMethod:     method,
				},
				Value: 1.0,
			},
			{
				Priority:   metric.PriorityHigh,
				MetricName: MetricHttpRequestsRejected,
				Dimensions: metric.Dimensions{
					dimensionServerName: name,
				},
				Value: 1.0,
			},
		})
	}
}

// getMetricMiddlewareDefaults reports zero for every route's rejected requests, so a route that never
// rejects a request reads as zero rather than as a gap. The request duration has no default: a
// histogram can not be seeded with an observation that did not happen.
func getMetricMiddlewareDefaults(name string, definitions ...Definition) metric.Data {
	return slices.Concat(
		funk.Map(definitions, func(definition Definition) *metric.Datum {
			return &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: MetricHttpRequestsRejected,
				Dimensions: metric.Dimensions{
					dimensionServerName: name,
					dimensionRoute:      definition.getAbsolutePath(),
					dimensionMethod:     definition.httpMethod,
				},
				Unit:  metric.UnitCount,
				Value: 0.0,
				Kind:  metric.KindCounter.Build(),
			}
		}),
		metric.Data{
			{
				Priority:   metric.PriorityHigh,
				MetricName: MetricHttpRequestsRejected,
				Dimensions: metric.Dimensions{
					dimensionServerName: name,
				},
				Unit:  metric.UnitCount,
				Value: 0.0,
				Kind:  metric.KindTotal,
			},
		},
	)
}
