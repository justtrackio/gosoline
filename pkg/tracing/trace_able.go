package tracing

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

//go:generate mockery --name TraceAble
type TraceAble interface {
	GetTrace() *Trace
}

type Trace struct {
	TraceId  string `json:"traceId"`
	Id       string `json:"id"`
	ParentId string `json:"parentId"`
	Sampled  bool   `json:"sampled"`
}

func (t *Trace) GetTraceId() string {
	return t.TraceId
}

func (t *Trace) GetId() string {
	return t.Id
}

func (t *Trace) GetParentId() string {
	return t.ParentId
}

func (t *Trace) GetSampled() bool {
	return t.Sampled
}

func TraceToString(trace *Trace) string {
	sampled := "0"

	if trace.Sampled {
		sampled = "1"
	}

	// we set "Parent" to our Id because this method is intended to forward a trace to a downstream service.
	// thus, setting us as the parent automatically creates a chain of parent/child traces
	parts := []string{"Root=", trace.TraceId, ";Parent=", trace.Id, ";Sampled=", sampled}

	return strings.Join(parts, "")
}

func StringToTrace(traceId string) (*Trace, error) {
	var err error

	variables := funk.Reduce(strings.Split(traceId, ";"), func(acc map[string]string, element string, i int) map[string]string {
		parts := strings.SplitN(element, "=", 2)
		if len(parts) == 2 {
			acc[parts[0]] = parts[1]
		} else {
			err = fmt.Errorf("a part [%s] of the trace id seems malformed", element)
		}

		return acc
	}, map[string]string{})

	trace := &Trace{
		TraceId: variables["Root"],
		// Self is set by the load balancer: https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-request-tracing.html
		Id:       variables["Self"],
		ParentId: variables["Parent"],
		Sampled:  variables["Sampled"] == "1",
	}

	if trace.TraceId == "" {
		return nil, fmt.Errorf("the trace id [%s] should contain a root part", traceId)
	}

	return trace, err
}

func GetTraceIdFromContext(ctx context.Context) *string {
	var trace *Trace

	if span := GetSpanFromContext(ctx); span != nil {
		trace = span.GetTrace()
	}

	if trace == nil {
		trace = GetTraceFromContext(ctx)
	}

	if trace == nil {
		return nil
	}

	return mdl.Box(TraceToString(trace))
}
