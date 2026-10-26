package httpserver

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	// MetricHttpRequestDuration records an HTTP server request's latency with response status.
	// Statusless counters report request totals; response counters report status by route and server.
	MetricHttpRequestDuration  = "request.duration"
	MetricHttpRequestsRejected = "rejected.requests"

	metricRequests       = "requests"
	metricRouteRequests  = "routed.requests"
	metricResponses      = "responses"
	metricRouteResponses = "routed.responses"

	dimensionServerName  = "name"
	dimensionRoute       = "route"
	dimensionMethod      = "request.method"
	dimensionStatusCode  = "response.status_code"
	dimensionStatusClass = "response.status_class"
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

	responseStatus := ginCtx.Writer.Status()
	responseStatusCode := strconv.Itoa(responseStatus)
	responseStatusClass := httpStatusClass(responseStatus)
	requestTimeNano := time.Since(start)
	requestTimeMillisecond := float64(requestTimeNano) / float64(time.Millisecond)
	serverDimensions := metric.Dimensions{
		dimensionServerName: name,
	}

	routeDimensions := metric.Dimensions{
		dimensionServerName: name,
		dimensionRoute:      path,
		dimensionMethod:     method,
	}
	responseDimensions := metric.Dimensions{
		dimensionServerName:  name,
		dimensionStatusCode:  responseStatusCode,
		dimensionStatusClass: responseStatusClass,
	}
	routedResponseDimensions := metric.Dimensions{
		dimensionServerName:  name,
		dimensionRoute:       path,
		dimensionMethod:      method,
		dimensionStatusCode:  responseStatusCode,
		dimensionStatusClass: responseStatusClass,
	}

	routedResponseTotalDimensions := metric.Dimensions{
		dimensionServerName:  name,
		dimensionRoute:       path,
		dimensionMethod:      method,
		dimensionStatusCode:  metric.DimensionDefault,
		dimensionStatusClass: responseStatusClass,
	}

	data := make(metric.Data, 0, 8)
	data = append(data,
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestDuration,
			Dimensions: metric.Dimensions{
				dimensionServerName: name,
				dimensionRoute:      path,
				dimensionMethod:     method,
				dimensionStatusCode: responseStatusCode,
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: requestTimeMillisecond,
			Kind:  metric.KindHistogram.WithHttpDurationBuckets().Build(),
		},
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricResponses,
			Dimensions: responseDimensions,
			Unit:       metric.UnitCount,
			Value:      1.0,
			Kind:       metric.KindCounter.Build(),
		},
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricRouteResponses,
			Dimensions: routedResponseDimensions,
			Unit:       metric.UnitCount,
			Value:      1.0,
			Kind:       metric.KindCounter.Build(),
		},
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricRouteResponses,
			Dimensions: routedResponseTotalDimensions,
			Unit:       metric.UnitCount,
			Value:      1.0,
			Kind:       metric.KindTotal,
		},
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: metricRequests,
			Dimensions: serverDimensions,
			Unit:       metric.UnitCount,
			Value:      1.0,
			Kind:       metric.KindCounter.Build(),
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
		data = append(data,
			&metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: MetricHttpRequestsRejected,
				Dimensions: routeDimensions,
				Unit:       metric.UnitCount,
				Value:      1.0,
				Kind:       metric.KindCounter.Build(),
			},
			&metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: MetricHttpRequestsRejected,
				Dimensions: serverDimensions,
				Unit:       metric.UnitCount,
				Value:      1.0,
				Kind:       metric.KindTotal,
			},
		)
	}

	writer.Write(ginCtx.Request.Context(), data)
}

// getMetricMiddlewareDefaults reports zero for request and rejected-request counters.
func getMetricMiddlewareDefaults(name string, definitions ...Definition) metric.Data {
	serverDimensions := metric.Dimensions{
		dimensionServerName: name,
	}
	defaults := make(metric.Data, 0, 2+2*len(definitions))
	defaults = append(defaults, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricRequests,
		Dimensions: serverDimensions,
		Unit:       metric.UnitCount,
		Value:      0.0,
		Kind:       metric.KindCounter.Build(),
	}, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricHttpRequestsRejected,
		Dimensions: serverDimensions,
		Unit:       metric.UnitCount,
		Value:      0.0,
		Kind:       metric.KindTotal,
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

func httpStatusClass(statusCode int) string {
	switch {
	case statusCode >= 100 && statusCode <= 199:
		return "1xx"
	case statusCode >= 200 && statusCode <= 299:
		return "2xx"
	case statusCode >= 300 && statusCode <= 399:
		return "3xx"
	case statusCode >= 400 && statusCode <= 499:
		return "4xx"
	case statusCode >= 500 && statusCode <= 599:
		return "5xx"
	default:
		return metric.DimensionDefault
	}
}
