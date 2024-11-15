package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

func init() {
	AddProvider("local", func(config cfg.Config, logger log.Logger) (Tracer, error) {
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

func (t localTracer) HttpHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if trace, err := StringToTrace(r.Header.Get(xray.TraceIDHeaderKey)); err == nil {
			ctx = ContextWithTrace(ctx, trace)
		} else {
			ctx = t.ensureLocalTrace(ctx)
		}

		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

func (t localTracer) ensureLocalTrace(ctx context.Context) context.Context {
	if trace := GetTraceFromContext(ctx); trace != nil {
		return ctx
	}

	trace := &Trace{
		TraceId:  fmt.Sprintf("goso:%s", t.traceIdSource.NewV4()),
		Id:       "00000000-0000-0000-0000-000000000000",
		ParentId: "00000000-0000-0000-0000-000000000000",
		Sampled:  false,
	}

	return ContextWithTrace(ctx, trace)
}
