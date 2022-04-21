package counter

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

type Response struct {
	Val int64 `json:"val"`
}

type counterController struct {
	mw metric.Writer
	c  *int64
}

func NewCounterController() *counterController {
	mw := metric.NewWriter()

	return &counterController{
		mw: mw,
		c:  mdl.Int64(0),
	}
}

func (ctrl *counterController) Cur(c *gin.Context) {
	ctrl.apiCallMetric("cur")
	c.JSON(http.StatusOK, &Response{
		Val: atomic.LoadInt64(ctrl.c),
	})
}

func (ctrl *counterController) Incr(c *gin.Context) {
	ctrl.apiCallMetric("incr")
	val := atomic.AddInt64(ctrl.c, 1)
	ctrl.counterMetric(float64(val))

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *counterController) Decr(c *gin.Context) {
	ctrl.apiCallMetric("decr")
	val := atomic.AddInt64(ctrl.c, -1)
	ctrl.counterMetric(float64(val))

	c.JSON(http.StatusOK, &Response{
		Val: val,
	})
}

func (ctrl *counterController) apiCallMetric(handler string) {
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

func (ctrl *counterController) counterMetric(val float64) {
	ctrl.mw.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "important_counter",
		Value:      val,
		Unit:       metric.UnitCount,
	})
}
