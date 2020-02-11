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

func (m MessageWithTraceEncoder) Encode(ctx context.Context, attributes map[string]interface{}) (context.Context, map[string]interface{}, error) {
	var trace *Trace

	if span := GetSpanFromContext(ctx); span != nil {
		trace = span.GetTrace()
	}

	if trace == nil {
		return ctx, attributes, nil
	}

	attributes["traceId"] = TraceToString(trace)

	return ctx, attributes, nil
}

func (m MessageWithTraceEncoder) Decode(ctx context.Context, attributes map[string]interface{}) (context.Context, map[string]interface{}, error) {
	var ok bool
	var traceId string

	if _, ok = attributes["traceId"]; !ok {
		return ctx, attributes, nil
	}

	if traceId, ok = attributes["traceId"].(string); !ok {
		err := fmt.Errorf("the traceId attribute should be of type string to decode it")
		err = m.strategy.TraceIdInvalid(err)

		return ctx, attributes, err
	}

	trace, err := StringToTrace(traceId)

	if err != nil {
		err := fmt.Errorf("the traceId attribute is invalid: %w", err)
		err = m.strategy.TraceIdInvalid(err)

		return ctx, attributes, err
	}

	ctx = ContextWithTrace(ctx, trace)
	delete(attributes, "traceId")

	return ctx, attributes, nil
}
