package controllers

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

type Response struct {
	Val int64 `json:"val"`
}

type metricController struct {
	mw metric.Writer
	c  *int64
}

func NewMetricController(ctx context.Context) *metricController {
	mw := metric.NewWriter(ctx)

	return &metricController{
		mw: mw,
		c:  mdl.Box(int64(0)),
	}
}

func (ctrl *metricController) Cur(c *gin.Context) {
	ctrl.apiCallCounterMetric("cur")
	c.JSON(http.StatusOK, &Response{
		Val: atomic.LoadInt64(ctrl.c),
	})
}

func (ctrl *metricController) Incr(c *gin.Context) {
	ctrl.apiCallCounterMetric("incr")
	val := atomic.AddInt64(ctrl.c, 1)
	ctrl.counterMetric(float64(1))

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *metricController) OneK(c *gin.Context) {
	ctrl.apiCallCounterMetric("1k")
	val := atomic.AddInt64(ctrl.c, 1000)
	ctrl.counterMetric(float64(1000))

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *metricController) MySummary(c *gin.Context) {
	ctrl.apiCallSummaryMetric(float64(atomic.LoadInt64(ctrl.c)) * 1.1)
	val := atomic.AddInt64(ctrl.c, 1)

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *metricController) apiCallCounterMetric(handler string) {
	ctrl.mw.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "api_request",
		Dimensions: metric.Dimensions{
			"handler": handler,
		},
		Value: 1,
		Unit:  metric.UnitPromCounter,
	})
}

func (ctrl *metricController) apiCallSummaryMetric(val float64) {
	ctrl.mw.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "my_summary",
		Value:      val,
		Unit:       metric.UnitPromSummary,
	})
}

func (ctrl *metricController) counterMetric(val float64) {
	ctrl.mw.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "important_counter",
		Value:      val,
		Unit:       metric.UnitCount,
	})
}
