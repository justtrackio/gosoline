package tracing

import (
	"context"
)

type contextTraceKeyType int

var contextTraceKey = new(contextTraceKeyType)

func ContextWithTrace(ctx context.Context, trace *Trace) context.Context {
	return context.WithValue(ctx, contextTraceKey, trace)
}

func GetTraceFromContext(ctx context.Context) *Trace {
	if ctx == nil {
		return nil
	}

	if trace, ok := ctx.Value(contextTraceKey).(*Trace); ok {
		return trace
	}

	return nil
}
