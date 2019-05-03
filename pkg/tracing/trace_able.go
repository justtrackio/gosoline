package tracing

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
