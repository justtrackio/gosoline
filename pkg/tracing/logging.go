package tracing

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
)

func ContextTraceFieldsResolver(ctx context.Context) map[string]any {
	var trace *Trace

	if span := GetSpanFromContext(ctx); span != nil {
		trace = span.GetTrace()
	}

	if trace == nil {
		trace = GetTraceFromContext(ctx)
	}

	if trace != nil {
		return map[string]any{
			"trace_id": trace.GetTraceId(),
		}
	}

	return map[string]any{}
}

type LoggerErrorHandler struct{}

func NewLoggerErrorHandler() *LoggerErrorHandler {
	return &LoggerErrorHandler{}
}

func (h *LoggerErrorHandler) Channels() log.Channels {
	return log.Channels{}
}

func (h *LoggerErrorHandler) Level() int {
	return log.PriorityError
}

func (h *LoggerErrorHandler) Log(_ time.Time, _ int, _ string, _ []any, err error, data log.Data) error {
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
