package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type PostProcessor interface {
	Process(ctx context.Context) error
}

type PostProcessorFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (PostProcessor, error)
