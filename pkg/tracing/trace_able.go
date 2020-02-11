package tracing

import (
	"fmt"
	"strings"
)

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

	parts := []string{"Root=", trace.TraceId, ";Parent=", trace.Id, ";Sampled=", sampled}

	return strings.Join(parts, "")
}

func StringToTrace(traceId string) (*Trace, error) {
	trace := &Trace{}
	parts := strings.Split(traceId, ";")

	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("the trace id [%s] should consist of at least 2 parts", traceId)
	}

	root := strings.Split(parts[0], "=")
	if len(root) != 2 {
		return nil, fmt.Errorf("the root part [%s] of the trace id seems malformed", parts[0])
	}
	trace.TraceId = root[1]

	if len(parts) == 2 {
		err := parseSampled(parts[1], trace)
		return trace, err
	}

	if err := parseParentId(parts[1], trace); err != nil {
		return trace, err
	}

	if err := parseSampled(parts[2], trace); err != nil {
		return trace, err
	}

	return trace, nil
}

func parseParentId(parentId string, trace *Trace) error {
	parent := strings.Split(parentId, "=")

	if len(parent) != 2 {
		return fmt.Errorf("the parent part [%s] of the trace id seems malformed", parentId)
	}

	trace.ParentId = parent[1]

	return nil
}

func parseSampled(sampledStr string, trace *Trace) error {
	sampled := strings.Split(sampledStr, "=")

	if len(sampled) != 2 {
		return fmt.Errorf("the sampled part [%s] of the trace id seems malformed", sampledStr)
	}

	trace.Sampled = sampled[1] == "1"

	return nil
}
