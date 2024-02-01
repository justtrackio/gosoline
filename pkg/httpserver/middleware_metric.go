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

	writer.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricHttpRequestResponseTime,
		Dimensions: metric.Dimensions{
			"Method":     method,
			"Path":       path,
			"ServerName": name,
		},
		Unit:  metric.UnitMillisecondsAverage,
		Value: requestTimeMillisecond,
	})

	writer.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricHttpRequestCount,
		Dimensions: metric.Dimensions{
			"Method":     method,
			"Path":       path,
			"ServerName": name,
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})

	status := ginCtx.Writer.Status() / 100
	statusMetric := fmt.Sprintf("%s%dXX", MetricHttpStatus, status)

	writer.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: statusMetric,
		Dimensions: metric.Dimensions{
			"Method":     method,
			"Path":       path,
			"ServerName": name,
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})
}

func getMetricMiddlewareDefaults(name string, definitions ...Definition) metric.Data {
	return funk.Map(definitions, func(definition Definition) *metric.Datum {
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
	})
}
