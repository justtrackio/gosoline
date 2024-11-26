package tracing

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

func init() {
	AddTracerProvider(ProviderLocal, func(ctx context.Context, config cfg.Config, logger log.Logger) (Tracer, error) {
		return NewLocalTracer(), nil
	})
}

type localTracer struct {
	traceIdSource uuid.Uuid
}

func NewLocalTracer() Tracer {
	return localTracer{
		traceIdSource: uuid.New(),
	}
}

func (t localTracer) StartSubSpan(ctx context.Context, _ string) (context.Context, Span) {
	return t.ensureLocalTrace(ctx), disabledSpan()
}

func (t localTracer) StartSpan(_ string) (context.Context, Span) {
	return t.ensureLocalTrace(context.Background()), disabledSpan()
}

func (t localTracer) StartSpanFromContext(ctx context.Context, _ string) (context.Context, Span) {
	return t.ensureLocalTrace(ctx), disabledSpan()
}

func (t localTracer) ensureLocalTrace(ctx context.Context) context.Context {
	if trace := GetTraceFromContext(ctx); trace != nil {
		return ctx
	}

	return contextWithLocalTraceId(ctx, t.traceIdSource.NewV4())
}
