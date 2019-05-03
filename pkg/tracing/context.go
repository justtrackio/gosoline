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
