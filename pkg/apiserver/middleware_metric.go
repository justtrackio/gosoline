package apiserver

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	MetricApiRequestCount        = "ApiRequestCount"
	MetricApiRequestResponseTime = "ApiRequestResponseTime"
)

func NewMetricMiddleware() (gin.HandlerFunc, func(definitions []Definition)) {
	// writer without any defaults until we initialize some defaults and overwrite it
	writer := metric.NewWriter()

	return func(ginCtx *gin.Context) {
			metricMiddleware(ginCtx, writer)
		}, func(definitions []Definition) {
			defaults := getMetricMiddlewareDefaults(definitions...)
			writer = metric.NewWriter(defaults...)
		}
}

func metricMiddleware(ginCtx *gin.Context, writer metric.Writer) {
	start := time.Now()
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
		MetricName: MetricApiRequestResponseTime,
		Dimensions: metric.Dimensions{
			"path": path,
		},
		Unit:  metric.UnitMillisecondsAverage,
		Value: requestTimeMillisecond,
	})

	writer.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricApiRequestCount,
		Dimensions: metric.Dimensions{
			"path": path,
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})

	status := ginCtx.Writer.Status() / 100
	statusMetric := fmt.Sprintf("ApiStatus%dXX", status)

	writer.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: statusMetric,
		Dimensions: metric.Dimensions{
			"path": path,
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})
}

func getMetricMiddlewareDefaults(definitions ...Definition) metric.Data {
	return funk.Map(definitions, func(definition Definition) *metric.Datum {
		return &metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricApiRequestCount,
			Dimensions: metric.Dimensions{
				"path": definition.getAbsolutePath(),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		}
	})
}
