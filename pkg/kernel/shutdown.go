package kernel

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ShutdownHandlerFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (ShutdownHandler, error)

// ShutdownHandler releases resources when the kernel exits.
//
//go:generate go run github.com/vektra/mockery/v2 --name ShutdownHandler
type ShutdownHandler interface {
	Shutdown(ctx context.Context) error
}
