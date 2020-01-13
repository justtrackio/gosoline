package tracing

import "context"

type ContextKeyType int

var ContextKey = new(ContextKeyType)

func ContextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, ContextKey, span)
}

func GetSpan(ctx context.Context) Span {
	if ctx == nil {
		return disabledSpan()
	}

	if span, ok := ctx.Value(ContextKey).(Span); ok {
		return span
	}

	return disabledSpan()
}

func ContextTraceFieldsResolver(ctx context.Context) map[string]interface{} {
	span := GetSpan(ctx)

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
