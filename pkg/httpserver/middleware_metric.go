package httpserver

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	MetricHttpRequestCount        = "HttpRequestCount"
	MetricHttpRequestResponseTime = "HttpRequestResponseTime"
	MetricHttpStatus              = "HttpStatus"
)

func NewMetricMiddleware(name string) (middleware gin.HandlerFunc, setupHandler func(definitions []Definition)) {
	// writer without any defaults until we initialize some defaults and overwrite it
	writer := metric.NewWriter()

	middleware = func(ginCtx *gin.Context) {
		metricMiddleware(name, ginCtx, writer)
	}

	setupHandler = func(definitions []Definition) {
		defaults := getMetricMiddlewareDefaults(name, definitions...)
		writer = metric.NewWriter(defaults...)
	}

	return middleware, setupHandler
}

func metricMiddleware(name string, ginCtx *gin.Context, writer metric.Writer) {
	start := time.Now()
	method := ginCtx.Request.Method

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

	status := ginCtx.Writer.Status() / 100
	statusMetric := fmt.Sprintf("%s%dXX", MetricHttpStatus, status)

	writer.Write(createMetricsWithDimensions(metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestResponseTime,
			Unit:       metric.UnitMillisecondsAverage,
			Value:      requestTimeMillisecond,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestCount,
			Unit:       metric.UnitCount,
			Value:      1.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: statusMetric,
			Unit:       metric.UnitCount,
			Value:      1.0,
		},
	}, metric.Dimensions{
		"Method":     method,
		"Path":       path,
		"ServerName": name,
	}, metric.Dimensions{
		"ServerName": name,
	}))
}

// createMetricsWithDimensions is creating a metric.Data set
// which included each provided metric with each provided set of dimensions.
func createMetricsWithDimensions(metrics metric.Data, dimensions ...metric.Dimensions) metric.Data {
	return funk.Flatten(funk.Map(metrics, func(metricDatum *metric.Datum) metric.Data {
		return funk.Map(dimensions, func(dimensions metric.Dimensions) *metric.Datum {
			datum := *metricDatum
			datum.Dimensions = dimensions

			return &datum
		})
	}))
}

func getMetricMiddlewareDefaults(name string, definitions ...Definition) metric.Data {
	return append(funk.Map(definitions, func(definition Definition) *metric.Datum {
		return &metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpRequestCount,
			Dimensions: metric.Dimensions{
				"Method":     definition.httpMethod,
				"Path":       definition.getAbsolutePath(),
				"ServerName": name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		}
	}), &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricHttpRequestCount,
		Dimensions: metric.Dimensions{
			"ServerName": name,
		},
		Unit:  metric.UnitCount,
		Value: 0.0,
	})
}
