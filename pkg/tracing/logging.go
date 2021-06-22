package tracing

import (
	"context"
	"github.com/applike/gosoline/pkg/log"
	"time"
)

func ContextTraceFieldsResolver(ctx context.Context) map[string]interface{} {
	span := GetSpanFromContext(ctx)

	if span == nil {
		return map[string]interface{}{}
	}

	traceId := span.GetTrace().GetTraceId()

	if traceId == "" {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"trace_id": span.GetTrace().GetTraceId(),
	}
}

type LoggerErrorHandler struct {
}

func NewLoggerErrorHandler() *LoggerErrorHandler {
	return &LoggerErrorHandler{}
}

func (h *LoggerErrorHandler) Channels() []string {
	return []string{}
}

func (h *LoggerErrorHandler) Level() int {
	return log.PriorityError
}

func (h *LoggerErrorHandler) Log(_ time.Time, _ int, _ string, _ []interface{}, err error, data log.Data) error {
	if err == nil {
		return nil
	}

	if data.Context == nil {
		return nil
	}

	span := GetSpanFromContext(data.Context)

	if span == nil {
		return nil
	}

	span.AddError(err)

	return nil
}
