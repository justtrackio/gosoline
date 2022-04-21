package metric

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewMetricModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	switch config.GetString("metric.writer", "none") {
	case WriterTypeProm:
		return NewMetricServer(ctx, config, logger)
	default:
		return NewDaemon(ctx, config, logger)
	}
}
