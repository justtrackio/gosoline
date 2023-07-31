package kernel

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Configurable interface {
	Init(context.Context, cfg.Config, log.Logger) error
}
