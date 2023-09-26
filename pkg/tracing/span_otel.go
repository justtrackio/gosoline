package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type otelSpan struct {
	span trace.Span
}

func (o *otelSpan) AddAnnotation(key string, value string) {
	o.span.SetAttributes(attribute.String(key, value))
}

func (o *otelSpan) AddError(err error) {
	o.span.RecordError(err)
}

func (o *otelSpan) AddMetadata(key string, value interface{}) {
	o.span.AddEvent(fmt.Sprintf("%s:%v", key, value))
}

func (o *otelSpan) Finish() {
	o.span.End()
}

func (o *otelSpan) GetId() string {
	return o.span.SpanContext().SpanID().String()
}

func (o *otelSpan) GetTrace() *Trace {
	var parentID string
	spanCtx := o.span.SpanContext()
	if parentSpanCtx := o.GetParentSpanContext(); parentSpanCtx != nil {
		parentID = parentSpanCtx.SpanID().String()
	}

	return &Trace{
		TraceId:  spanCtx.TraceID().String(),
		Id:       spanCtx.SpanID().String(),
		ParentId: parentID,
		Sampled:  spanCtx.IsSampled(),
	}
}

func (o *otelSpan) GetParentSpanContext() *trace.SpanContext {
	roSpan, ok := o.span.(sdktrace.ReadOnlySpan)
	if !ok {
		return nil
	}

	parent := roSpan.Parent()

	return &parent

}

func newOtelSpan(ctx context.Context, span trace.Span) (context.Context, *otelSpan) {
	s := &otelSpan{
		span: span,
	}

	return ContextWithSpan(ctx, s), s
}
