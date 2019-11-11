package apiserver

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"time"
)

const (
	MetricApiRequestCount   = "ApiRequestCount"
	MetricApiRequestLatency = "ApiRequestLatency"
)

func CreateMetricHandler(definition Definition) gin.HandlerFunc {
	defaults := getMetricMiddlewareDefaults(definition)
	writer := mon.NewMetricDaemonWriter(defaults...)

	return func(ginCtx *gin.Context) {
		start := time.Now()

		ginCtx.Next()

		requestTimeNano := time.Since(start)
		requestTimeMillisecond := float64(requestTimeNano) / float64(time.Millisecond)
		status := ginCtx.Writer.Status() / 100
		statusMetric := fmt.Sprintf("ApiStatus%dXX", status)

		writer.WriteOne(&mon.MetricDatum{
			Priority:   mon.PriorityLow,
			MetricName: MetricApiRequestCount,
			Dimensions: mon.MetricDimensions{
				"path": definition.getAbsolutePath(),
			},
			Unit:  mon.UnitCount,
			Value: 1.0,
		})

		writer.WriteOne(&mon.MetricDatum{
			Priority:   mon.PriorityLow,
			MetricName: MetricApiRequestLatency,
			Dimensions: mon.MetricDimensions{
				"path": definition.getAbsolutePath(),
			},
			Unit:  mon.UnitMilliseconds,
			Value: requestTimeMillisecond,
		})

		writer.WriteOne(&mon.MetricDatum{
			Priority:   mon.PriorityLow,
			MetricName: statusMetric,
			Dimensions: mon.MetricDimensions{
				"path": definition.getAbsolutePath(),
			},
			Unit:  mon.UnitCount,
			Value: 1.0,
		})
	}
}

func getMetricMiddlewareDefaults(definition Definition) mon.MetricData {
	defaults := make(mon.MetricData, 0)

	metric := &mon.MetricDatum{
		Priority:   mon.PriorityLow,
		MetricName: MetricApiRequestCount,
		Dimensions: mon.MetricDimensions{
			"path": definition.getAbsolutePath(),
		},
		Unit:  mon.UnitCount,
		Value: 0.0,
	}
	defaults = append(defaults, metric)

	metric = &mon.MetricDatum{
		Priority:   mon.PriorityLow,
		MetricName: "ApiStatus2XX",
		Dimensions: mon.MetricDimensions{
			"path": definition.getAbsolutePath(),
		},
		Unit:  mon.UnitCount,
		Value: 0.0,
	}
	defaults = append(defaults, metric)

	metric = &mon.MetricDatum{
		Priority:   mon.PriorityLow,
		MetricName: "ApiStatus3XX",
		Dimensions: mon.MetricDimensions{
			"path": definition.getAbsolutePath(),
		},
		Unit:  mon.UnitCount,
		Value: 0.0,
	}
	defaults = append(defaults, metric)

	metric = &mon.MetricDatum{
		Priority:   mon.PriorityLow,
		MetricName: "ApiStatus4XX",
		Dimensions: mon.MetricDimensions{
			"path": definition.getAbsolutePath(),
		},
		Unit:  mon.UnitCount,
		Value: 0.0,
	}
	defaults = append(defaults, metric)

	metric = &mon.MetricDatum{
		Priority:   mon.PriorityLow,
		MetricName: "ApiStatus5XX",
		Dimensions: mon.MetricDimensions{
			"path": definition.getAbsolutePath(),
		},
		Unit:  mon.UnitCount,
		Value: 0.0,
	}
	defaults = append(defaults, metric)

	return defaults
}
