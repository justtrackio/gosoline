package limit

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/log"
)

type loggingMiddleware struct {
	logger log.Logger
}

func NewLoggingMiddleware(logger log.Logger) MiddlewareFactory {
	return func() Middleware {
		return &loggingMiddleware{
			logger: logger.WithChannel("rate_limiter"),
		}
	}
}

func (l loggingMiddleware) getFields(i Invocation) log.Fields {
	return log.Fields{
		"trace_id": i.GetTraceId(),
		"name":     i.GetName(),
		"prefix":   i.GetPrefix(),
	}
}

func (l loggingMiddleware) OnTake(ctx context.Context, i Invocation) {
	l.logger.
		WithContext(ctx).
		WithFields(l.getFields(i)).
		Info("trying to take from limiter")
}

func (l loggingMiddleware) OnRelease(ctx context.Context, i Invocation) {
	l.logger.
		WithContext(ctx).
		WithFields(l.getFields(i)).
		Info("releasing request from limiting")
}

func (l loggingMiddleware) OnError(ctx context.Context, i Invocation) {
	l.logger.
		WithContext(ctx).
		WithFields(l.getFields(i)).
		Warn("error while getting rate limit")
}

func (l loggingMiddleware) OnThrottle(ctx context.Context, i Invocation) {
	l.logger.
		WithContext(ctx).
		WithFields(l.getFields(i)).
		Info("throttling request as rate limit was reached")
}
