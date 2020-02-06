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

type MessageWithTraceEncoder struct {
}

func NewMessageWithTraceEncoder() *MessageWithTraceEncoder {
	return &MessageWithTraceEncoder{}
}

func (m MessageWithTraceEncoder) Encode(ctx context.Context, attributes map[string]interface{}) (map[string]interface{}, error) {
	var trace *Trace

	if span := GetSpanFromContext(ctx); span != nil {
		trace = span.GetTrace()
	}

	if trace == nil {
		return attributes, nil
	}

	attributes["traceId"] = TraceToString(trace)

	return attributes, nil
}

func (m MessageWithTraceEncoder) Decode(ctx context.Context, attributes map[string]interface{}) (context.Context, error) {
	var ok bool
	var traceId string

	if _, ok = attributes["traceId"]; !ok {
		return ctx, nil
	}

	if traceId, ok = attributes["traceId"].(string); !ok {
		return ctx, fmt.Errorf("the traceId attribute should be of type string to decode it")
	}

	trace, err := StringToTrace(traceId)

	if err != nil {
		return ctx, fmt.Errorf("the traceId attribute is invalid: %w", err)
	}

	ctx = ContextWithTrace(ctx, trace)

	return ctx, nil
}
