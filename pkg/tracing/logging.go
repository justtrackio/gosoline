package tracing

import (
	"context"
	"github.com/applike/gosoline/pkg/mon"
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

type LoggerErrorHook struct{}

func NewLoggerErrorHook() *LoggerErrorHook {
	return &LoggerErrorHook{}
}

func (l LoggerErrorHook) Fire(_ string, _ string, err error, data *mon.Metadata) error {
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
