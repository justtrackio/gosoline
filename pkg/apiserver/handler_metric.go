package apiserver

import (
	"fmt"
	"github.com/applike/gosoline/pkg/metric"
	"github.com/gin-gonic/gin"
	"time"
)

const (
	MetricApiRequestCount        = "ApiRequestCount"
	MetricApiRequestResponseTime = "ApiRequestResponseTime"
)

func CreateMetricHandler(definition Definition) gin.HandlerFunc {
	defaults := getMetricMiddlewareDefaults(definition)
	writer := metric.NewDaemonWriter(defaults...)

	return func(ginCtx *gin.Context) {
		start := time.Now()

		ginCtx.Next()

		requestTimeNano := time.Since(start)
		requestTimeMillisecond := float64(requestTimeNano) / float64(time.Millisecond)

		writer.WriteOne(&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricApiRequestResponseTime,
			Dimensions: metric.Dimensions{
				"path": definition.getAbsolutePath(),
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: requestTimeMillisecond,
		})

		writer.WriteOne(&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricApiRequestCount,
			Dimensions: metric.Dimensions{
				"path": definition.getAbsolutePath(),
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
				"path": definition.getAbsolutePath(),
			},
			Unit:  metric.UnitCount,
			Value: 1.0,
		})
	}
}

func getMetricMiddlewareDefaults(definition Definition) metric.Data {
	defaults := make(metric.Data, 0)

	metric := &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricApiRequestCount,
		Dimensions: metric.Dimensions{
			"path": definition.getAbsolutePath(),
		},
		Unit:  metric.UnitCount,
		Value: 0.0,
	}
	defaults = append(defaults, metric)

	return defaults
}
