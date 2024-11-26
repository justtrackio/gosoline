package tracing

import (
	"context"
	"fmt"
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

func contextWithLocalTraceId(ctx context.Context, uuidV4 string) context.Context {
	localTraceId := &Trace{
		TraceId:  fmt.Sprintf("goso:%s", uuidV4),
		Id:       "00000000-0000-0000-0000-000000000000",
		ParentId: "00000000-0000-0000-0000-000000000000",
		Sampled:  false,
	}

	return ContextWithTrace(ctx, localTraceId)
}
