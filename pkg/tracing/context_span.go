package tracing

import "context"

type contextSpanKeyType int

var contextSpanKey = new(contextSpanKeyType)

func ContextWithSpan(ctx context.Context, span Span) context.Context {
	return context.WithValue(ctx, contextSpanKey, span)
}

func GetSpanFromContext(ctx context.Context) Span {
	if ctx == nil {
		return nil
	}

	if span, ok := ctx.Value(contextSpanKey).(Span); ok {
		return span
	}

	return nil
}
