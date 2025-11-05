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

func NewMetricController() *metricController {
	mw := metric.NewWriter()

	return &metricController{
		mw: mw,
		c:  mdl.Box(int64(0)),
	}
}

func (ctrl *metricController) Cur(c *gin.Context) {
	ctrl.apiCallCounterMetric(c.Request.Context(), "cur")
	c.JSON(http.StatusOK, &Response{
		Val: atomic.LoadInt64(ctrl.c),
	})
}

func (ctrl *metricController) Incr(c *gin.Context) {
	ctrl.apiCallCounterMetric(c.Request.Context(), "incr")
	val := atomic.AddInt64(ctrl.c, 1)
	ctrl.counterMetric(c.Request.Context(), float64(1))

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *metricController) OneK(c *gin.Context) {
	ctrl.apiCallCounterMetric(c.Request.Context(), "1k")
	val := atomic.AddInt64(ctrl.c, 1000)
	ctrl.counterMetric(c.Request.Context(), float64(1000))

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *metricController) MySummary(c *gin.Context) {
	ctrl.apiCallSummaryMetric(c.Request.Context(), float64(atomic.LoadInt64(ctrl.c))*1.1)
	val := atomic.AddInt64(ctrl.c, 1)

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *metricController) apiCallCounterMetric(ctx context.Context, handler string) {
	ctrl.mw.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "api_request",
		Dimensions: metric.Dimensions{
			"handler": handler,
		},
		Value: 1,
		Unit:  metric.UnitCount,
		Kind:  metric.KindCounter.Build(),
	})
}

func (ctrl *metricController) apiCallSummaryMetric(ctx context.Context, val float64) {
	ctrl.mw.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "my_summary",
		Value:      val,
		Kind:       metric.KindSummary.Build(),
	})
}

func (ctrl *metricController) counterMetric(ctx context.Context, val float64) {
	ctrl.mw.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "important_counter",
		Value:      val,
		Unit:       metric.UnitCount,
	})
}
