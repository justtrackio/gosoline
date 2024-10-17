package tracing

import (
	"context"
	"fmt"
)

type MessageWithTraceEncoder struct {
	strategy TraceIdErrorStrategy
}

func NewMessageWithTraceEncoder(strategy TraceIdErrorStrategy) *MessageWithTraceEncoder {
	return &MessageWithTraceEncoder{
		strategy: strategy,
	}
}

func (m MessageWithTraceEncoder) Encode(ctx context.Context, _ interface{}, attributes map[string]string) (context.Context, map[string]string, error) {
	var trace *Trace

	if span := GetSpanFromContext(ctx); span != nil {
		trace = span.GetTrace()
	}

	if trace == nil {
		trace = GetTraceFromContext(ctx)
	}

	if trace != nil {
		attributes["traceId"] = TraceToString(trace)
	}

	return ctx, attributes, nil
}

func (m MessageWithTraceEncoder) Decode(ctx context.Context, _ interface{}, attributes map[string]string) (context.Context, map[string]string, error) {
	var ok bool

	if _, ok = attributes["traceId"]; !ok {
		return ctx, attributes, nil
	}

	trace, err := StringToTrace(attributes["traceId"])
	if err != nil {
		err := fmt.Errorf("the traceId attribute is invalid: %w", err)
		err = m.strategy.TraceIdInvalid(err)

		return ctx, attributes, err
	}

	ctx = ContextWithTrace(ctx, trace)
	delete(attributes, "traceId")

	return ctx, attributes, nil
}
