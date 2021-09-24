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
	Middleware func(ctx context.Context, config cfg.Config, logger log.Logger, next Handler) Handler
	Handler    func()
)
