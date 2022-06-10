package kernel

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Position string

const (
	PositionBeginning Position = "beginning"
	PositionEnd       Position = "end"
)

type (
	MiddlewareFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (Middleware, error)
	Middleware        func(next MiddlewareHandler) MiddlewareHandler
	MiddlewareHandler func()
)

func BuildSimpeMiddleware(handler func(next MiddlewareHandler)) MiddlewareFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (Middleware, error) {
		return func(next MiddlewareHandler) MiddlewareHandler {
			return func() {
				handler(next)
			}
		}, nil
	}
}
